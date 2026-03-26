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
}

export interface UserBalance {
  pulsePoints: number;
  spinCredits: number;
}
