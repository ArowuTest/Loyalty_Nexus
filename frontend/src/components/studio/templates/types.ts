import { StudioTool } from '../../../types/studio';

export interface GeneratePayload {
  prompt: string;
  aspect_ratio?: string;
  duration?: number;
  voice_id?: string;
  language?: string;
  vocals?: boolean;
  lyrics?: string;
  style_tags?: string[];
  negative_prompt?: string;
  image_url?: string;
  document_url?: string;  // FEAT-01: pre-uploaded PDF/TXT CDN URL for knowledge tools
  extra_params?: Record<string, unknown>;
}

export interface TemplateProps {
  tool: StudioTool;
  onSubmit: (payload: GeneratePayload) => void;
  isLoading: boolean;
  userPoints: number;
  /** Optional pre-loaded image URL — populated when user clicks "Animate This" or "Edit Photo" on a result */
  preloadImageUrl?: string;
  /** Optional pre-loaded video URL — populated when user clicks "Extend Video" on a result */
  preloadVideoUrl?: string;
}
