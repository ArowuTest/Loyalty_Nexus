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
