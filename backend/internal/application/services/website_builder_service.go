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

// ─── Design Variation System ─────────────────────────────────────────────────

// designVariant holds curated style/layout combinations so every generated
// website looks meaningfully different. One is picked at random per request.
type designVariant struct {
	layoutStyle     string // overall layout feel
	heroStyle       string // hero section treatment
	cardStyle       string // card/section style
	fontPersonality string // typography tone
	accentApproach  string // how accent colour is used
	animationStyle  string // motion/animation feel
	imageStyle      string // Pollinations image style modifier
}

var corporateVariants = []designVariant{
	{
		layoutStyle:     "asymmetric grid with large left-rail text and right visual",
		heroStyle:       "full-bleed video-still hero with bold white text overlay and gradient scrim",
		cardStyle:       "glassmorphism cards with blur backdrop and thin border",
		fontPersonality: "modern geometric — large tracking, caps headlines",
		accentApproach:  "electric blue used only on CTAs and line accents, rest is monochrome",
		animationStyle:  "slide-in-from-left on scroll, staggered children",
		imageStyle:      "cinematic wide angle professional photography",
	},
	{
		layoutStyle:     "centered single-column with generous whitespace",
		heroStyle:       "split hero: left text, right large circular image frame",
		cardStyle:       "elevated white cards with thick left border accent stripe",
		fontPersonality: "editorial serif for headings, clean sans for body",
		accentApproach:  "warm gradient from brand colour used as section backgrounds",
		animationStyle:  "fade-up with scale from 0.95, subtle parallax on hero",
		imageStyle:      "warm editorial photography with slight film grain",
	},
	{
		layoutStyle:     "bento grid layout with mixed size tiles",
		heroStyle:       "dark hero with animated gradient mesh background, no image",
		cardStyle:       "bento boxes with rounded-3xl, different sizes, some dark some light",
		fontPersonality: "bold variable font, extra large numbers as decorative elements",
		accentApproach:  "neon accent on dark background, high contrast",
		animationStyle:  "pop-in animation with spring physics, hover lift",
		imageStyle:      "high contrast product photography on dark background",
	},
	{
		layoutStyle:     "magazine-style with sidebar and main content",
		heroStyle:       "overlapping elements hero — text overlaps image with mix-blend-mode",
		cardStyle:       "minimal bordered cards with icon + text, no shadows",
		fontPersonality: "clean humanist sans, medium weight, relaxed line height",
		accentApproach:  "muted earth tones with one bold accent pop",
		animationStyle:  "gentle float animation on key elements, smooth reveal",
		imageStyle:      "natural light lifestyle photography",
	},
	{
		layoutStyle:     "full-width sections with alternating image-text rows",
		heroStyle:       "particle/geometric animated hero background built in CSS",
		cardStyle:       "icon-first cards with large emoji/icon, minimal text",
		fontPersonality: "condensed bold for impact headlines, wide for subtext",
		accentApproach:  "duotone effect — brand colour overlaid on images",
		animationStyle:  "counter animations on stats, typewriter on headline",
		imageStyle:      "aerial drone photography Nigeria cityscape",
	},
}

var shopVariants = []designVariant{
	{
		layoutStyle:     "Pinterest-style masonry product grid",
		heroStyle:       "full-screen product lifestyle shot with floating price badges",
		cardStyle:       "product cards with hover zoom, quick-add overlay",
		fontPersonality: "playful rounded font, friendly tone",
		accentApproach:  "gradient price tags, bold sale badges",
		animationStyle:  "card flip on hover showing product details",
		imageStyle:      "bright studio product photography white background",
	},
	{
		layoutStyle:     "magazine editorial with featured product hero",
		heroStyle:       "luxury editorial — dark background, single hero product large",
		cardStyle:       "minimal cards with product name under, price subtle",
		fontPersonality: "elegant thin serif for brand, clean sans for prices",
		accentApproach:  "gold accent on dark — premium feel",
		animationStyle:  "parallax scroll on hero, fade in product grid",
		imageStyle:      "fashion editorial photography dramatic lighting",
	},
	{
		layoutStyle:     "colourful grid with category strips",
		heroStyle:       "bright colourful hero with illustrated elements and bold CTA",
		cardStyle:       "colourful category cards, rounded, playful",
		fontPersonality: "bold playful with lots of colour",
		accentApproach:  "multiple accent colours — one per category",
		animationStyle:  "bounce in, wiggle on hover CTA",
		imageStyle:      "vibrant colourful product flat lay photography",
	},
}

// getVariants returns the right slice for a site type.
func getVariants(siteType string) []designVariant {
	switch siteType {
	case "shop":
		return shopVariants
	default:
		return corporateVariants
	}
}

// pickVariant selects a pseudo-random variant using the business name as seed
// so the SAME business always gets the SAME variant (deterministic per business)
// but DIFFERENT businesses get different variants.
func pickVariant(siteType, businessName string) designVariant {
	variants := getVariants(siteType)
	if len(variants) == 0 {
		return designVariant{}
	}
	// Hash the business name to pick a consistent variant.
	h := 0
	for _, c := range businessName {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return variants[h%len(variants)]
}

// ─── Prompt Construction ──────────────────────────────────────────────────────

// BuildWebsitePrompt constructs the full system + user prompt for Gemini.
func BuildWebsitePrompt(req WebsiteBuilderRequest) (systemPrompt, userPrompt string) {
	businessName := req.Fields["business_name"]
	systemPrompt = websiteBuilderSystemPrompt(req.SiteType, businessName)
	userPrompt   = websiteBuilderUserPrompt(req)
	return
}

// websiteBuilderSystemPrompt returns the design-focused system instruction for Gemini.
func websiteBuilderSystemPrompt(siteType, businessName string) string {
	typeSpecific := websiteTypeInstructions(siteType)
	variant := pickVariant(siteType, businessName)

	variantInstructions := ""
	if variant.layoutStyle != "" {
		variantInstructions = fmt.Sprintf(`

═══════════════════════ UNIQUE DESIGN DIRECTION (MANDATORY) ═══════════════════════

This website MUST use this specific design style — do NOT default to a generic template:

LAYOUT STYLE: %s
HERO TREATMENT: %s
CARD STYLE: %s
TYPOGRAPHY PERSONALITY: %s
ACCENT COLOUR APPROACH: %s
ANIMATION STYLE: %s
IMAGE STYLE (use this in all Pollinations prompts): add "%s" to every image prompt

These are MANDATORY design constraints. The output must feel like a professional
design agency made this specific choice for this specific business.
DO NOT produce a generic corporate website — follow these exact style directions.`,
			variant.layoutStyle,
			variant.heroStyle,
			variant.cardStyle,
			variant.fontPersonality,
			variant.accentApproach,
			variant.animationStyle,
			variant.imageStyle,
		)
	}

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

%s%s`, typeSpecific, variantInstructions)
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
		sb.WriteString("IMAGE STYLE MODIFIER: Append the image style from your design direction to every Pollinations prompt.\n")
		sb.WriteString("  For example if style is 'cinematic wide angle', append ', cinematic wide angle' to each image prompt.\n")
		sb.WriteString("REQUIREMENTS:\n")
		sb.WriteString("  - Generate at least 4-6 relevant Pollinations images throughout the website\n")
		sb.WriteString("  - Match image prompts to the specific business type, name and services provided\n")
		sb.WriteString("  - Use images in: hero section, about section, services/products cards, gallery\n")
		sb.WriteString("  - Make hero image a full-width background using CSS: background-image: url('https://image.pollinations.ai/...')\n")
		sb.WriteString("  - Add loading='lazy' to all <img> tags for performance\n")
	}

	sb.WriteString("\n=== GENERATION TIME ===\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("January 2006")))

	sb.WriteString("\nCREATIVITY LEVEL: Be bold and creative. Do NOT produce a standard template. Make this website feel hand-crafted and unique.\n")
	sb.WriteString("Temperature hint: Push design boundaries within the brief provided.\n")

	return sb.String()
}
