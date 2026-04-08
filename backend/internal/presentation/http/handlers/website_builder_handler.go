package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// slugRe matches only lowercase letters, digits and hyphens — everything else is stripped.
var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// sanitizeSlug converts any string into a clean URL slug:
//
//	"Techvault Solutions!" → "techvault-solutions"
func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRe.ReplaceAllString(s, "")
	// Collapse consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

// ─── POST /api/v1/studio/website ─────────────────────────────────────────────
// Authenticated. Accepts WebsiteBuilderRequest JSON (now includes optional vanity_slug).
// Deducts 25 Pulse Points, calls Gemini, stores HTML.
// Returns generation_id + vanity_slug + shareable public URL.

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
		req.Photos = req.Photos[:6]
	}

	// ── Resolve vanity slug ──────────────────────────────────────────────────
	// Priority: user-provided slug → auto-generated from business_name → skip
	vanitySlug := ""
	if req.VanitySlug != "" {
		vanitySlug = sanitizeSlug(req.VanitySlug)
	} else if biz := req.Fields["business_name"]; biz != "" {
		vanitySlug = sanitizeSlug(biz)
	}

	// Ensure uniqueness — append a short random suffix if taken
	if vanitySlug != "" {
		taken, _ := h.studioSvc.SlugExists(r.Context(), vanitySlug)
		if taken {
			// append 4-char random hex
			vanitySlug = fmt.Sprintf("%s-%s", vanitySlug, randomHex(4))
		}
	}

	// Find the website-builder tool
	tool, err := h.studioSvc.FindToolBySlug(r.Context(), "website-builder")
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "website-builder tool not found"})
		return
	}

	// Deduct points + create generation record
	prompt := fmt.Sprintf("[website-builder] type=%s slug=%s fields=%d photos=%d",
		req.SiteType, vanitySlug, len(req.Fields), len(req.Photos))
	gen, err := h.studioSvc.RequestGeneration(r.Context(), userID, tool.ID, prompt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Persist slug immediately (before Gemini runs) so the URL is shareable right away
	if vanitySlug != "" {
		_ = h.studioSvc.SetVanitySlug(r.Context(), gen.ID, vanitySlug)
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

	// Call Gemini asynchronously (website generation can take 30-90s for rich HTML)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		var htmlOutput string
		var genErr error

		if h.gemini != nil && len(images) > 0 {
			htmlOutput, genErr = h.gemini.CompleteWithImages(ctx, systemPrompt, userPrompt, images)
		} else if h.gemini != nil {
			htmlOutput, genErr = h.gemini.Complete(ctx, systemPrompt, userPrompt)
		} else {
			genErr = fmt.Errorf("gemini adapter not configured")
		}

		if genErr != nil {
			_ = h.studioSvc.FailGeneration(ctx, gen.ID, genErr.Error())
			return
		}

		// Strip any markdown code blocks if Gemini wrapped the HTML
		htmlOutput = stripMarkdownCodeBlock(htmlOutput)

		// Inject "Built with Nexus" badge just before </body>
		badge := `<div style="position:fixed;bottom:12px;left:12px;z-index:9998;background:rgba(0,0,0,0.6);color:#fff;font-size:10px;padding:4px 8px;border-radius:20px;font-family:sans-serif;backdrop-filter:blur(4px);">⚡ Built with <a href="https://loyalty-nexus.vercel.app" style="color:#F5A623;text-decoration:none;">Nexus</a></div>`
		htmlOutput = strings.Replace(htmlOutput, "</body>", badge+"</body>", 1)

		_ = h.studioSvc.CompleteGeneration(ctx, gen.ID,
			"", // no output_url (HTML is self-contained)
			"", // no output_url_2
			htmlOutput,
			"gemini/website-builder",
			0,
			0,
		)
	}()

	// Return with slug-based public URL when available, UUID fallback otherwise
	publicPath := fmt.Sprintf("/s/%s", gen.ID.String())
	if vanitySlug != "" {
		publicPath = fmt.Sprintf("/s/%s", vanitySlug)
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"generation_id": gen.ID,
		"vanity_slug":   vanitySlug,
		"status":        "pending",
		"public_url":    publicPath,
		"message":       "Your website is being built — usually ready in 10-15 seconds",
	})
}

// ─── GET /api/v1/studio/website/check-slug ───────────────────────────────────
// Authenticated. Query param: ?slug=my-business-name
// Returns { slug, available, suggestion } — used for live slug validation in the wizard.

func (h *StudioHandler) CheckSlug(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("slug")
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug query param required"})
		return
	}

	clean := sanitizeSlug(raw)
	if clean == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"slug":      "",
			"available": false,
			"error":     "slug contains no valid characters",
		})
		return
	}

	taken, err := h.studioSvc.SlugExists(r.Context(), clean)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not check slug"})
		return
	}

	suggestion := clean
	if taken {
		suggestion = fmt.Sprintf("%s-%s", clean, randomHex(4))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"slug":       clean,
		"available":  !taken,
		"suggestion": suggestion,
	})
}

// ─── GET /s/{id} ─────────────────────────────────────────────────────────────
// Public. No auth. Resolves UUID or vanity slug, serves the generated HTML.

func (h *StudioHandler) ServeWebsite(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	// Try UUID first, then fall back to vanity slug lookup
	if genID, err := uuid.Parse(idStr); err == nil {
		g, err := h.studioSvc.FindGenerationByID(r.Context(), genID)
		if err != nil {
			serveErrorPage(w, "Website not found", "This link may have expired or doesn't exist.")
			return
		}
		serveGenerationHTML(w, idStr, g.Status, g.ErrorMessage, g.OutputText)
	} else {
		g, err := h.studioSvc.FindGenerationBySlug(r.Context(), idStr)
		if err != nil {
			serveErrorPage(w, "Website not found", "This link may have expired or doesn't exist.")
			return
		}
		serveGenerationHTML(w, idStr, g.Status, g.ErrorMessage, g.OutputText)
	}
}

func serveGenerationHTML(w http.ResponseWriter, id, status, errMsg, outputText string) {
	switch status {
	case "pending", "processing":
		serveLoadingPage(w, id)
		return
	case "failed":
		serveErrorPage(w, "Generation failed", errMsg)
		return
	}
	if outputText == "" {
		serveErrorPage(w, "Website unavailable", "This website has no content. Please regenerate.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Frame-Options", "ALLOWALL")
	_, _ = w.Write([]byte(outputText))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// randomHex returns n random hex bytes as a string (length = n*2).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"```html\n", "```html", "```\n", "```"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	s = strings.TrimSuffix(s, "```")
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
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>%s</title><meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{background:#0a0a0f;color:#fff;font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;text-align:center;padding:24px}
.card{background:rgba(255,255,255,0.05);border-radius:24px;padding:48px 32px;max-width:380px}
h2{font-size:22px;margin-bottom:12px}p{color:rgba(255,255,255,0.5)}</style>
</head><body><div class="card"><h2>%s</h2><p>%s</p></div></body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
}
