import React from 'react';
import {
  AbsoluteFill, Img, Audio, Sequence, useCurrentFrame, useVideoConfig,
  interpolate, spring,
} from 'remotion';
import { z } from 'zod';

export const FPS = 30;
export const PER_CLIP_FRAMES = 75; // 2.5s per image
const FADE = 15;                    // crossfade frames

export const slideshowSchema = z.object({
  images: z.array(z.string()).min(1).max(8),
  caption: z.string().optional().default(''),
  aspectRatio: z.enum(['9:16', '1:1', '16:9']).optional().default('9:16'),
  musicUrl: z.string().optional().default(''),
});

type Props = z.infer<typeof slideshowSchema>;

// One image clip: Ken-Burns zoom + crossfade in/out.
const Slide: React.FC<{ src: string; index: number }> = ({ src, index }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // Fade in over FADE frames, fade out over the last FADE frames.
  const opacity = interpolate(
    frame,
    [0, FADE, PER_CLIP_FRAMES - FADE, PER_CLIP_FRAMES],
    [0, 1, 1, 0],
    { extrapolateLeft: 'clamp', extrapolateRight: 'clamp' },
  );

  // Subtle Ken-Burns: zoom 1.0 → 1.08, alternate pan direction by index.
  const zoom = spring({ frame, fps, from: 1.0, to: 1.08, durationInFrames: PER_CLIP_FRAMES });
  const panX = interpolate(frame, [0, PER_CLIP_FRAMES], [index % 2 === 0 ? -20 : 20, index % 2 === 0 ? 20 : -20]);

  return (
    <AbsoluteFill style={{ opacity, backgroundColor: '#000' }}>
      <Img
        src={src}
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'cover',
          transform: `scale(${zoom}) translateX(${panX}px)`,
        }}
      />
    </AbsoluteFill>
  );
};

export const VideoSlideshow: React.FC<Props> = ({ images, caption, musicUrl }) => {
  return (
    <AbsoluteFill style={{ backgroundColor: '#0c0c10' }}>
      {images.map((src, i) => (
        <Sequence key={i} from={i * PER_CLIP_FRAMES} durationInFrames={PER_CLIP_FRAMES}>
          <Slide src={src} index={i} />
        </Sequence>
      ))}

      {/* Caption bar (bottom) */}
      {caption ? (
        <AbsoluteFill style={{ justifyContent: 'flex-end', alignItems: 'center', padding: 60 }}>
          <div
            style={{
              background: 'linear-gradient(90deg, rgba(217,70,239,0.9), rgba(34,211,238,0.9))',
              color: '#fff',
              fontFamily: 'sans-serif',
              fontWeight: 800,
              fontSize: 52,
              lineHeight: 1.2,
              padding: '18px 34px',
              borderRadius: 20,
              textAlign: 'center',
              maxWidth: '86%',
              boxShadow: '0 8px 40px rgba(0,0,0,0.5)',
            }}
          >
            {caption}
          </div>
        </AbsoluteFill>
      ) : null}

      {musicUrl ? <Audio src={musicUrl} /> : null}
    </AbsoluteFill>
  );
};
