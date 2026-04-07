package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// ─── POST /api/v1/studio/website ─────────────────────────────────────────────
// Authenticated. Accepts WebsiteBuilderRequest JSON.
// Deducts 25 Pulse Points, calls Gemini with multimodal prompt, stores HTML.
// Returns generation_id + shareable public URL.

func (h *StudioHandler) BuildWebsite(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(middleware.ContextUserID).(string)
	userID, err := uuid.Parse(uid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user"})
		return
	}

	var req services.WebsiteBuilderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.SiteType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "site_type is required"})
		return
	}
	if len(req.Photos) > 6 {
		req.Photos = req.Photos[:6] // enforce max 6
	}

	// Find the website-builder tool
	tool, err := h.studioSvc.FindToolBySlug(r.Context(), "website-builder")
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "website-builder tool not found"})
		return
	}

	// Deduct points + create generation record
	prompt := fmt.Sprintf("[website-builder] type=%s fields=%d photos=%d", req.SiteType, len(req.Fields), len(req.Photos))
	gen, err := h.studioSvc.RequestGeneration(r.Context(), userID, tool.ID, prompt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Build prompts
	systemPrompt, userPrompt := services.BuildWebsitePrompt(req)

	// Extract base64 images
	images := make([]string, 0, len(req.Photos))
	for _, p := range req.Photos {
		if p.Base64 != "" {
			images = append(images, p.Base64)
		}
	}

	// Call Gemini asynchronously (website generation takes 5-15 seconds)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		var html string
		var genErr error

		if h.gemini != nil && len(images) > 0 {
			html, genErr = h.gemini.CompleteWithImages(ctx, systemPrompt, userPrompt, images)
		} else if h.gemini != nil {
			html, genErr = h.gemini.Complete(ctx, systemPrompt, userPrompt)
		} else {
			genErr = fmt.Errorf("gemini adapter not configured")
		}

		if genErr != nil {
			_ = h.studioSvc.FailGeneration(ctx, gen.ID, genErr.Error())
			return
		}

		// Strip any markdown code blocks if Gemini wrapped the HTML
		html = stripMarkdownCodeBlock(html)

		// Inject "Built with Nexus" badge just before </body>
		badge := `<div style="position:fixed;bottom:12px;left:12px;z-index:9998;background:rgba(0,0,0,0.6);color:#fff;font-size:10px;padding:4px 8px;border-radius:20px;font-family:sans-serif;backdrop-filter:blur(4px);">⚡ Built with <a href="https://loyalty-nexus.vercel.app" style="color:#F5A623;text-decoration:none;">Nexus</a></div>`
		html = strings.Replace(html, "</body>", badge+"</body>", 1)

		_ = h.studioSvc.CompleteGeneration(ctx, gen.ID,
			"", // no output_url (HTML is self-contained)
			"", // no output_url_2
			html,
			"gemini/website-builder",
			0,
			0,
		)
	}()

	// Return immediately with generation ID
	publicURL := fmt.Sprintf("/s/%s", gen.ID.String())
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"generation_id": gen.ID,
		"status":        "pending",
		"public_url":    publicURL,
		"message":       "Your website is being built — usually ready in 10-15 seconds",
	})
}

// ─── GET /s/{id} ─────────────────────────────────────────────────────────────
// Public. No auth. Serves the generated HTML directly.

func (h *StudioHandler) ServeWebsite(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	genID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	gen, err := h.studioSvc.FindGenerationByID(r.Context(), genID)
	if err != nil {
		serveErrorPage(w, "Website not found", "This link may have expired or doesn't exist.")
		return
	}

	switch gen.Status {
	case "pending", "processing":
		serveLoadingPage(w, idStr)
		return
	case "failed":
		serveErrorPage(w, "Generation failed", gen.ErrorMessage)
		return
	}

	if gen.OutputText == "" {
		serveErrorPage(w, "Website unavailable", "This website has no content. Please regenerate.")
		return
	}

	// Serve the HTML directly
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	_, _ = w.Write([]byte(gen.OutputText))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```html or ``` prefix/suffix
	for _, prefix := range []string{"```html\n", "```html", "```\n", "```"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func serveLoadingPage(w http.ResponseWriter, id string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("loading").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Building your website...</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0a0a0f;color:#fff;font-family:-apple-system,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;text-align:center;padding:24px}
.card{background:rgba(255,255,255,0.05);border:1px solid rgba(255,255,255,0.1);border-radius:24px;padding:48px 32px;max-width:380px;width:100%}
.spinner{width:56px;height:56px;border:3px solid rgba(245,166,35,0.2);border-top:3px solid #F5A623;border-radius:50%;animation:spin 1s linear infinite;margin:0 auto 24px}
@keyframes spin{to{transform:rotate(360deg)}}
h2{font-size:22px;font-weight:800;margin-bottom:12px}
p{color:rgba(255,255,255,0.5);font-size:14px;line-height:1.6}
.badge{margin-top:32px;font-size:12px;color:rgba(255,255,255,0.25)}
</style>
<meta http-equiv="refresh" content="4">
</head><body>
<div class="card">
  <div class="spinner"></div>
  <h2>Building your website...</h2>
  <p>Our AI is designing your site right now.<br>This usually takes 10–15 seconds.</p>
  <p class="badge">⚡ Powered by Nexus AI Studio</p>
</div>
</body></html>`))
	_ = tmpl.Execute(w, map[string]string{"id": id})
}

func serveErrorPage(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>%s</title><meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{background:#0a0a0f;color:#fff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;text-align:center;padding:24px}
.card{background:rgba(255,255,255,0.05);border-radius:24px;padding:48px 32px;max-width:380px}
h2{font-size:22px;margin-bottom:12px}p{color:rgba(255,255,255,0.5)}</style>
</head><body><div class="card"><h2>%s</h2><p>%s</p></div></body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
}
