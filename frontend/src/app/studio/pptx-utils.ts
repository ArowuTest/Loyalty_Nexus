/**
 * PPTX generation utilities for AI Studio Slide Deck tool
 * Uses pptxgenjs to create real, downloadable PowerPoint files from
 * the structured JSON returned by the AI backend.
 *
 * Design goals:
 *  - Professional dark-themed slides matching the Nexus brand
 *  - Handles the exact JSON schema returned by the slide-deck AI prompt:
 *    { title, subtitle, slides: [{ number, title, bullets[], speaker_notes }] }
 *  - Falls back gracefully if JSON is malformed
 *  - Client-side only (no server round-trip needed)
 */

// ─── Types ────────────────────────────────────────────────────────────────────

export interface SlideData {
  number?: number;
  title?: string;
  subtitle?: string;
  bullets?: string[];
  notes?: string;
  speaker_notes?: string;
}

export interface SlideDeckData {
  title?: string;
  subtitle?: string;
  slides: SlideData[];
}

// ─── Colour palette (Nexus brand) ─────────────────────────────────────────────

const BRAND = {
  bg:           '0A0B14',   // near-black background
  bgSlide:      '0F1020',   // slightly lighter slide bg
  gold:         'C9A227',   // brand gold
  goldLight:    'E8C547',   // lighter gold for bullets
  white:        'FFFFFF',
  textMuted:    '8892A4',   // muted body text
  accentPurple: '6C63FF',   // accent for dividers
  accentBlue:   '3B82F6',   // secondary accent
  slideNum:     '2A2D45',   // slide number chip bg
};

// ─── Parse helper ─────────────────────────────────────────────────────────────

/**
 * Parse raw AI output_text into a SlideDeckData object.
 * Handles both array format and { title, slides: [] } format.
 */
export function parseSlideDeckJSON(raw: string): SlideDeckData | null {
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return { slides: parsed };
    }
    if (parsed && Array.isArray(parsed.slides)) {
      return parsed as SlideDeckData;
    }
    return null;
  } catch {
    return null;
  }
}

// ─── Main export ──────────────────────────────────────────────────────────────

/**
 * Generate and download a .pptx file from slide deck JSON text.
 * @param rawJson   The output_text string from the AI backend
 * @param filename  Optional filename (without extension)
 */
export async function downloadAsPPTX(
  rawJson: string,
  filename = 'nexus-slide-deck',
): Promise<void> {
  const data = parseSlideDeckJSON(rawJson);
  if (!data || data.slides.length === 0) {
    throw new Error('Could not parse slide deck data. Please try regenerating.');
  }

  // Dynamic import so pptxgenjs is only loaded when needed (code-split)
  const PptxGenJS = (await import('pptxgenjs')).default;
  const pptx = new PptxGenJS();

  // ── Presentation-level settings ──────────────────────────────────────────
  pptx.layout   = 'LAYOUT_WIDE';   // 16:9 widescreen
  pptx.author   = 'Nexus AI Studio';
  pptx.company  = 'Loyalty Nexus';
  pptx.subject  = data.title ?? 'AI-Generated Presentation';
  pptx.title    = data.title ?? 'Nexus Presentation';

  // ── Define master slide (dark background) ────────────────────────────────
  pptx.defineSlideMaster({
    title: 'NEXUS_MASTER',
    background: { color: BRAND.bgSlide },
    objects: [
      // Bottom gold accent bar
      {
        rect: {
          x: 0, y: 6.9, w: '100%', h: 0.08,
          fill: { color: BRAND.gold },
        },
      },
      // Nexus branding watermark (bottom-right)
      {
        text: {
          text: 'Nexus AI Studio',
          options: {
            x: 8.5, y: 6.95, w: 3, h: 0.2,
            fontSize: 7,
            color: BRAND.textMuted,
            align: 'right',
          },
        },
      },
    ],
  });

  // ── Slide 1: Title slide ──────────────────────────────────────────────────
  const titleSlide = pptx.addSlide({ masterName: 'NEXUS_MASTER' });

  // Background gradient overlay (simulated with a semi-transparent rect)
  titleSlide.addShape(pptx.ShapeType.rect, {
    x: 0, y: 0, w: '100%', h: '100%',
    fill: { color: BRAND.bg },
  });

  // Gold accent line (left edge)
  titleSlide.addShape(pptx.ShapeType.rect, {
    x: 0, y: 1.5, w: 0.12, h: 3.5,
    fill: { color: BRAND.gold },
  });

  // Main title
  titleSlide.addText(data.title ?? 'Presentation', {
    x: 0.5, y: 1.8, w: 11.5, h: 1.6,
    fontSize: 40,
    bold: true,
    color: BRAND.white,
    fontFace: 'Calibri',
    align: 'left',
    charSpacing: 0.5,
  });

  // Subtitle
  if (data.subtitle) {
    titleSlide.addText(data.subtitle, {
      x: 0.5, y: 3.5, w: 9, h: 0.8,
      fontSize: 18,
      color: BRAND.gold,
      fontFace: 'Calibri',
      align: 'left',
      italic: true,
    });
  }

  // Slide count badge
  titleSlide.addText(`${data.slides.length} slides`, {
    x: 0.5, y: 4.5, w: 2, h: 0.4,
    fontSize: 10,
    color: BRAND.textMuted,
    fontFace: 'Calibri',
    align: 'left',
  });

  // ── Content slides ────────────────────────────────────────────────────────
  data.slides.forEach((slide, idx) => {
    const s = pptx.addSlide({ masterName: 'NEXUS_MASTER' });
    const slideNum = slide.number ?? idx + 1;

    // Slide number chip (top-right)
    s.addShape(pptx.ShapeType.rect, {
      x: 11.5, y: 0.15, w: 0.8, h: 0.35,
      fill: { color: BRAND.slideNum },
      line: { color: BRAND.gold, width: 0.5 },
    });
    s.addText(String(slideNum).padStart(2, '0'), {
      x: 11.5, y: 0.15, w: 0.8, h: 0.35,
      fontSize: 9,
      color: BRAND.gold,
      bold: true,
      align: 'center',
      valign: 'middle',
    });

    // Slide title
    s.addText(slide.title ?? `Slide ${slideNum}`, {
      x: 0.4, y: 0.18, w: 10.8, h: 0.75,
      fontSize: 24,
      bold: true,
      color: BRAND.white,
      fontFace: 'Calibri',
      align: 'left',
    });

    // Gold divider line under title
    s.addShape(pptx.ShapeType.rect, {
      x: 0.4, y: 0.95, w: 11.5, h: 0.025,
      fill: { color: BRAND.gold },
    });

    // Subtitle (if present on first-level slides)
    if (slide.subtitle) {
      s.addText(slide.subtitle, {
        x: 0.4, y: 1.0, w: 11, h: 0.4,
        fontSize: 13,
        color: BRAND.gold,
        italic: true,
        fontFace: 'Calibri',
      });
    }

    // Bullet points
    const bullets = slide.bullets ?? [];
    if (bullets.length > 0) {
      const bulletRows = bullets.map((b) => ({
        text: b,
        options: {
          bullet: { type: 'bullet' as const, characterCode: '25B8' }, // ▸
          fontSize: 15,
          color: BRAND.white,
          fontFace: 'Calibri',
          paraSpaceBefore: 6,
        },
      }));

      const yStart = slide.subtitle ? 1.45 : 1.15;
      const maxH   = 5.2;

      s.addText(bulletRows, {
        x: 0.5, y: yStart, w: 11.3, h: maxH,
        valign: 'top',
        autoFit: true,
      });
    }

    // Speaker notes (hidden in presentation, visible in notes pane)
    const notes = slide.speaker_notes ?? slide.notes;
    if (notes) {
      s.addNotes(notes);
    }
  });

  // ── Thank-you / end slide ─────────────────────────────────────────────────
  const endSlide = pptx.addSlide({ masterName: 'NEXUS_MASTER' });

  endSlide.addShape(pptx.ShapeType.rect, {
    x: 0, y: 0, w: '100%', h: '100%',
    fill: { color: BRAND.bg },
  });

  endSlide.addShape(pptx.ShapeType.rect, {
    x: 4.5, y: 2.8, w: 3.5, h: 0.08,
    fill: { color: BRAND.gold },
  });

  endSlide.addText('Thank You', {
    x: 0, y: 2.2, w: '100%', h: 1,
    fontSize: 44,
    bold: true,
    color: BRAND.white,
    align: 'center',
    fontFace: 'Calibri',
  });

  endSlide.addText('Generated by Nexus AI Studio', {
    x: 0, y: 3.2, w: '100%', h: 0.5,
    fontSize: 14,
    color: BRAND.gold,
    align: 'center',
    italic: true,
  });

  // ── Download ──────────────────────────────────────────────────────────────
  const date = new Date().toISOString().split('T')[0];
  await pptx.writeFile({ fileName: `${filename}-${date}.pptx` });
}
