export type ToolCategory = 'Chat' | 'Create' | 'Learn' | 'Build';

export interface StudioTool {
  id: string;
  name: string;
  description: string;
  category: ToolCategory;
  pointCost: number;
  iconName: string;
  isActive: boolean;
  isNew?: boolean;
  examplePrompt?: string;
}

export interface UserBalance {
  pulsePoints: number;
  spinCredits: number;
}
