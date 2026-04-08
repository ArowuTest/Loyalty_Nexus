package services

// website_builder_service.go — AI-powered website generator
//
// Flow:
//   1. Frontend sends structured JSON: type + form fields + up to 6 base64 photos
//   2. We build a rich type-specific system prompt
//   3. Gemini 2.5 Flash generates a complete single-page HTML website
//   4. HTML is stored in ai_generations.output_text
//   5. Served publicly at GET /s/{generation_id}

import (
	"fmt"
	"strings"
	"time"
)

// WebsiteBuilderRequest is the structured input from the frontend multi-step form.
type WebsiteBuilderRequest struct {
	SiteType    string            `json:"site_type"`    // shop|corporate|professional|restaurant|portfolio|events|church|education
	VanitySlug  string            `json:"vanity_slug"`  // optional — user-chosen URL slug e.g. "techvault-solutions"
	Fields      map[string]string `json:"fields"`       // type-specific form fields
	Photos      []WebsitePhoto    `json:"photos"`       // up to 6 photos, client-compressed
}

// WebsitePhoto carries one product/content photo and its user-written caption.
type WebsitePhoto struct {
	Base64 string `json:"base64"` // JPEG, client-compressed to ~80KB
	Caption string `json:"caption"` // e.g. "Ankara dress, ₦15,000, available in 3 colours"
}

// BuildWebsitePrompt constructs the full system + user prompt for Gemini.
func BuildWebsitePrompt(req WebsiteBuilderRequest) (systemPrompt, userPrompt string) {
	systemPrompt = websiteBuilderSystemPrompt(req.SiteType)
	userPrompt   = websiteBuilderUserPrompt(req)
	return
}

// websiteBuilderSystemPrompt returns the design-focused system instruction for Gemini.
func websiteBuilderSystemPrompt(siteType string) string {
	typeSpecific := websiteTypeInstructions(siteType)
	return fmt.Sprintf(`You are a world-class web designer and copywriter specialising in mobile-first Nigerian business websites.

═══════════════════════ CORE RULES (non-negotiable) ═══════════════════════

OUTPUT: Return ONLY the complete HTML file. Start with <!DOCTYPE html>. 
No markdown. No code blocks. No explanation. Just the raw HTML.

SELF-CONTAINED: Zero external dependencies. 
- No CDN links. No Google Fonts. No external images.
- All CSS inside one <style> tag. All JS inside one <script> tag.
- Use system font stack: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif

MOBILE-FIRST: Design for a 375px screen first. Must look perfect on Tecno/Infinix/Samsung phones.

═══════════════════════ DESIGN STANDARDS ═══════════════════════

You are NOT generating a form dump. You are designing a real business website.

HERO SECTION (mandatory):
- Full-width, at least 100vh on mobile
- Bold gradient background (specific to business type — never plain white)
- Business name in large, bold typography (min 42px mobile)
- Compelling tagline — DO NOT echo the user's words back verbatim. 
  Transform "tailoring shop" into "Where every stitch tells your story"
  Transform "IT company" into "Powering the next generation of African businesses"
- Two CTA buttons: primary action + secondary (e.g. "Shop Now" + "Contact Us")

COLOUR PSYCHOLOGY (use these palette rules by type):
- shop/fashion: deep purple #2D1B69 + gold #F5A623 + white
- restaurant/food: warm charcoal #1A1A1A + amber #F97316 + cream #FFF8F0
- corporate/startup: deep navy #0F172A + electric blue #3B82F6 + white
- professional/services: forest green #064E3B + gold #D97706 + white
- portfolio: near-black #0A0A0A + cyan #06B6D4 + white
- events: deep purple #4C1D95 + pink #EC4899 + white
- church/religious: deep navy #1E3A5F + gold #F5A623 + white
- education: dark teal #134E4A + orange #F97316 + white

TYPOGRAPHY:
- Section headings: 28-32px bold, letter-spacing -0.5px
- Body: 16px, line-height 1.7
- Strong visual hierarchy — never a wall of text

SPACING: Generous padding. Each section gets min 80px top/bottom padding.

CARDS: Subtle box-shadow (0 4px 24px rgba(0,0,0,0.12)), border-radius 16px, 
hover effect with transform: translateY(-4px) and transition.

ANIMATIONS: CSS IntersectionObserver fade-in-up on every section.
Floating WhatsApp button (bottom-right, always visible, pulsing green glow).

PHOTOS: If base64 photos are provided, embed them as <img src="data:image/jpeg;base64,...">
in product/gallery cards. Never use placeholder images — only use what the user provides.

WHATSAPP INTEGRATION:
- All WhatsApp links: https://wa.me/234XXXXXXXXXX (strip leading 0, add 234)
- Pre-filled message relevant to the business
- Prominent green WhatsApp button on every product/service card
- Sticky floating WhatsApp FAB bottom-right

NIGERIAN CONTEXT:
- Use ₦ for all prices
- Nigerian phone format awareness
- Relevant local language if business name suggests it

FOOTER: Business name, brief tagline, contact details, "Built with Nexus" subtle badge.

═══════════════════════ SITE TYPE SPECIFIC ═══════════════════════

%s`, typeSpecific)
}

func websiteTypeInstructions(siteType string) string {
	switch siteType {
	case "shop":
		return `SHOP/CATALOGUE LAYOUT:
1. Hero — name, tagline, "Shop Now" + "Order on WhatsApp" CTAs
2. Featured Products — 2-3 column grid, each card: photo, name, price, "Order" WhatsApp button
3. About section — story of the business (write creatively from the details provided)
4. Info bar — location, hours, delivery info
5. Contact — WhatsApp + phone, map placeholder
6. Footer

Each product card MUST have an "Order on WhatsApp" button with pre-filled message.
If photos are provided, one photo per card. Caption becomes the product description.`

	case "corporate":
		return `CORPORATE/STARTUP LAYOUT:
1. Hero — company name, mission statement, "Get Started" + "Learn More" CTAs
2. Services — icon cards (use emoji as icons), 3-column grid, brief description each
3. About/Mission — company story, values, brief team mention if names provided
4. Why Choose Us — 3-4 key differentiators with icons
5. Contact — email, phone, WhatsApp, address
6. Footer with company name and tagline`

	case "professional":
		return `PROFESSIONAL SERVICES LAYOUT:
1. Hero — name, professional title, credentials tagline, "Book Consultation" CTA
2. Services — clean list/cards with brief descriptions and starting prices if provided
3. About — professional bio, qualifications, experience
4. Why Choose Me — 3 key value propositions
5. Process — how engagement works (3-4 steps)
6. Contact — all channels, office location, hours
7. Footer`

	case "restaurant":
		return `RESTAURANT/FOOD LAYOUT:
1. Hero — restaurant name, cuisine type, "View Menu" + "Order Now" CTAs, opening hours badge
2. Signature Dishes — large cards with photos, dish name, price, order button
3. About — story, specialities, atmosphere description
4. Menu preview — categorised (Starters/Mains/Drinks/Desserts) if details provided
5. Location & Hours — prominent, with Google Maps placeholder
6. Contact + Reservation CTA
7. Footer`

	case "portfolio":
		return `PORTFOLIO LAYOUT:
1. Hero — name, title/profession, "View Work" + "Hire Me" CTAs
2. Work Gallery — masonry/grid of portfolio photos with captions
3. About — professional story, skills, passion
4. Services/What I Offer — clean list with rates if provided
5. Skills — visual skill bars or icon tags
6. Contact — all channels + availability status
7. Footer`

	case "events":
		return `EVENTS LAYOUT:
1. Hero — business name, event types covered, "Get a Quote" CTA
2. Our Services — cards with service type, brief description, starting price
3. Gallery — photo grid of past events (if photos provided)
4. How It Works — 3-4 step process
5. Testimonial placeholder section
6. Pricing packages if provided
7. Contact + WhatsApp booking CTA
8. Footer`

	case "church":
		return `CHURCH/RELIGIOUS LAYOUT:
1. Hero — church/ministry name, tagline, "Join Us" + service times
2. Service Times — prominent schedule card
3. About — mission, vision, brief history
4. What We Offer — ministries, programmes, cell groups
5. Leadership — if names provided
6. Location — address, map placeholder
7. Give/Donate — bank details or online giving
8. Contact + Footer`

	case "education":
		return `EDUCATION LAYOUT:
1. Hero — school/centre name, tagline, "Enrol Now" CTA
2. Courses/Classes — cards with subject, level, schedule, fee
3. Why Choose Us — key benefits with icons
4. How It Works — enrolment process steps
5. About — background, qualifications, approach
6. Fees + Schedule overview
7. Contact + Enrolment CTA
8. Footer`

	default:
		return `GENERAL BUSINESS LAYOUT:
1. Hero — name, tagline, primary CTA
2. Services/Products — clean card grid
3. About
4. Contact with WhatsApp
5. Footer`
	}
}

func websiteBuilderUserPrompt(req WebsiteBuilderRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Create a %s website with the following details:\n\n", req.SiteType))

	// Write all form fields
	sb.WriteString("=== BUSINESS DETAILS ===\n")
	for key, val := range req.Fields {
		if val != "" {
			label := strings.ReplaceAll(key, "_", " ")
			sb.WriteString(fmt.Sprintf("%s: %s\n", label, val))
		}
	}

	// Photo captions
	if len(req.Photos) > 0 {
		sb.WriteString(fmt.Sprintf("\n=== PHOTOS PROVIDED: %d photos ===\n", len(req.Photos)))
		for i, p := range req.Photos {
			if p.Caption != "" {
				sb.WriteString(fmt.Sprintf("Photo %d: %s\n", i+1, p.Caption))
			} else {
				sb.WriteString(fmt.Sprintf("Photo %d: (no caption)\n", i+1))
			}
		}
		sb.WriteString("\nEmbed all photos as base64 inline images in the appropriate sections.\n")
		sb.WriteString("IMPORTANT: The photos follow this text prompt in the multimodal request. Use them in the website.\n")
	} else {
		sb.WriteString("\nNo user photos provided. You MUST generate relevant images using Pollinations AI URLs.\n")
		sb.WriteString("Use this exact URL format for every image in the website:\n")
		sb.WriteString("  https://image.pollinations.ai/prompt/[DESCRIPTION]?width=800&height=500&nologo=true&model=flux\n")
		sb.WriteString("Replace [DESCRIPTION] with a URL-encoded descriptive prompt relevant to the business.\n")
		sb.WriteString("Examples:\n")
		sb.WriteString("  - Hero background: https://image.pollinations.ai/prompt/modern%20professional%20Nigerian%20business%20office%20Lagos%20skyline?width=1200&height=600&nologo=true&model=flux\n")
		sb.WriteString("  - Product image: https://image.pollinations.ai/prompt/elegant%20ankara%20fashion%20dress%20boutique%20Nigeria?width=600&height=600&nologo=true&model=flux\n")
		sb.WriteString("  - Team/about: https://image.pollinations.ai/prompt/professional%20African%20business%20team%20smiling%20office?width=800&height=500&nologo=true&model=flux\n")
		sb.WriteString("REQUIREMENTS:\n")
		sb.WriteString("  - Generate at least 4-6 relevant Pollinations images throughout the website\n")
		sb.WriteString("  - Match image prompts to the specific business type, name and services provided\n")
		sb.WriteString("  - Use images in: hero section, about section, services/products cards, gallery\n")
		sb.WriteString("  - Make hero image a full-width background using CSS: background-image: url('https://image.pollinations.ai/...')\n")
		sb.WriteString("  - Add loading='lazy' to all <img> tags for performance\n")
	}

	sb.WriteString("\n=== GENERATION TIME ===\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("January 2006")))

	return sb.String()
}
