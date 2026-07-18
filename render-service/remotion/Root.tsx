import React from 'react';
import { Composition } from 'remotion';
import { VideoSlideshow, slideshowSchema, PER_CLIP_FRAMES, FPS } from './VideoSlideshow';

// Aspect-ratio → pixel dimensions (portrait default for social).
function dims(ar: string): { width: number; height: number } {
  if (ar === '16:9') return { width: 1920, height: 1080 };
  if (ar === '1:1') return { width: 1080, height: 1080 };
  return { width: 1080, height: 1920 }; // 9:16 default
}

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="video-slideshow"
        component={VideoSlideshow}
        schema={slideshowSchema}
        fps={FPS}
        // Placeholders — overridden per-render by calculateMetadata from props.
        durationInFrames={PER_CLIP_FRAMES * 3}
        width={1080}
        height={1920}
        defaultProps={{
          images: [] as string[],
          caption: '',
          aspectRatio: '9:16',
          musicUrl: '',
        }}
        calculateMetadata={({ props }) => {
          const { width, height } = dims(props.aspectRatio || '9:16');
          const n = Math.max(1, (props.images || []).length);
          return {
            width,
            height,
            durationInFrames: n * PER_CLIP_FRAMES,
            fps: FPS,
          };
        }}
      />
    </>
  );
};
