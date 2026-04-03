export type ToolCategory =
  | 'Knowledge & Research'
  | 'Image & Visual'
  | 'Video & Animation'
  | 'Audio & Voice'
  | 'Code & Data'
  | 'Build & Create'
  | 'Document & Business'
  | 'Music & Entertainment'
  | 'Language & Translation'
  | 'Vision'
  | 'Chat'
  | 'Build'
  | 'Create';

// ── UIConfig type system ──────────────────────────────────────────────────────
// The backend returns ui_template (string slug) and ui_config (JSON object)
// for every tool. The frontend reads these to render the correct input form.

export type UITemplate =
  | 'chat'
  | 'music-composer'
  | 'image-creator'
  | 'image-editor'
  | 'video-creator'
  | 'video-animator'
  | 'voice-studio'
  | 'transcribe'
  | 'vision-ask'
  | 'knowledge-doc';

export interface AspectRatioOption {
  label: string;
  value: string;
  icon?: string;
}

export interface VoiceOption {
  id: string;
  name: string;
  tone: string;
  category: string;
}

export interface LanguageOption {
  code: string;
  label: string;
}

export interface KnowledgeField {
  key: string;
  label: string;
  type: 'text' | 'textarea' | 'select';
  required: boolean;
  placeholder?: string;
  rows?: number;
  options?: string[];
  default?: string;
}

export interface UIConfig {
  // shared
  prompt_placeholder?: string;
  output_hint?: string;
  generation_warning?: string;
  // music
  genre_tags?: string[];
  duration_options?: number[];
  default_duration?: number;
  show_vocals_toggle?: boolean;
  default_vocals?: boolean;
  show_lyrics_box?: boolean;
  lyrics_placeholder?: string;
  // image / video
  aspect_ratios?: AspectRatioOption[];
  default_aspect?: string;
  style_tags?: string[];
  show_negative_prompt?: boolean;
  negative_prompt_placeholder?: string;
  show_style_tags?: boolean;
  duration_options_video?: number[];
  default_duration_video?: number;
  // upload
  upload_label?: string;
  upload_accept?: string[];
  max_file_mb?: number;
  max_duration_mins?: number;
  // voice / transcribe
  voices?: VoiceOption[];
  default_voice?: string;
  languages?: LanguageOption[];
  default_language?: string;
  show_language_selector?: boolean;
  show_speaker_labels?: boolean;
  max_chars?: number;
  // vision
  prompt_optional?: boolean;
  // knowledge
  fields?: KnowledgeField[];
  output_format?: 'text' | 'document' | 'audio';
  // chat
  show_history?: boolean;
  // music v2
  show_bpm?: boolean;
  show_energy?: boolean;
  max_duration?: number;
  // image v2
  show_quality_toggle?: boolean;
  // image editor v2
  edit_suggestions?: string[];
  show_edit_prompt?: boolean;   // false = hide edit instruction (e.g. bg-remover)
  output_note?: string;         // optional note shown below upload zone
  // video v2
  camera_movements?: Array<{ label: string; icon?: string; value: string }>;
  show_audio_direction?: boolean;   // show audio/sound direction field (video-veo)
  show_music_style?: boolean;       // show music style field (video-jingle)
  show_image_upload?: boolean;      // show optional image upload (video-jingle)
  image_upload_optional?: boolean;  // whether image upload is optional
  image_upload_label?: string;      // label for image upload zone
  image_upload_hint?: string;       // hint text inside image upload zone
  // voice v2
  show_speed_control?: boolean;
  show_format_selector?: boolean;
  // transcribe v2
  show_output_format?: boolean;
  // vision v2
  example_questions?: string[];
  // translate
  translate_languages?: LanguageOption[];
}

export interface StudioTool {
  id: string;
  slug: string;
  name: string;
  description: string;
  category: string;
  point_cost: number;
  is_active: boolean;
  icon?: string;
  provider?: string;
  sort_order?: number;
  entry_point_cost: number;
  refund_window_mins: number;
  refund_pct: number;
  is_free: boolean;
  ui_template: UITemplate;   // ← NEW
  ui_config: UIConfig;       // ← NEW
}

export interface AIGeneration {
  id: string;
  user_id: string;
  tool_id: string;
  tool_slug: string;
  tool_name?: string;
  prompt: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  output_url?: string;
  output_text?: string;
  error_message?: string;
  provider?: string;
  points_deducted: number;
  created_at: string;
  updated_at: string;
  expires_at?: string;
  disputed_at?: string;
  refund_granted: boolean;
  refund_pts: number;
}

export interface UserBalance {
  pulsePoints: number;
  spinCredits: number;
}
