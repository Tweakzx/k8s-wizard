export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  model?: string;
  clarification?: ClarificationRequest;
  actionPreview?: ActionPreview;
  status?: 'needs_info' | 'needs_confirm' | 'executed';
  suggestions?: Suggestion[];
}

export interface NavItem {
  id: string;
  label: string;
  icon: string;
}

export interface QuickAction {
  label: string;
  command: string;
}

export interface ChatRequest {
  content: string;
  namespace?: string;
  formData?: Record<string, any>;
  confirm?: boolean;
  sessionId?: string;
}

export interface ChatResponse {
  result: string;
  message?: string;
  error?: string;
  model?: string;
  clarification?: ClarificationRequest;
  actionPreview?: ActionPreview;
  status?: 'needs_info' | 'needs_confirm' | 'executed';
  suggestions?: Suggestion[];
}

export interface ClarificationRequest {
  type: 'form' | 'options' | 'confirm';
  title: string;
  description?: string;
  fields: ClarificationField[];
  action?: string;
}

export interface ClarificationField {
  key: string;
  label: string;
  type: 'text' | 'select' | 'number' | 'textarea';
  placeholder?: string;
  default?: string | number;
  options?: { label: string; value: string }[];
  required: boolean;
  group?: string;
}

export interface ActionPreview {
  type: 'create' | 'delete' | 'scale' | 'get';
  resource: string;
  namespace: string;
  yaml?: string;
  params: Record<string, any>;
  dangerLevel: 'low' | 'medium' | 'high';
  summary: string;
}

export interface ModelInfo {
  provider: string;
  id: string;
  name: string;
}

export interface ModelInfoResponse {
  current: string;
  models: ModelInfo[];
  config: Record<string, string>;
}

export interface ConfigResponse {
  version: string;
  models: {
    primary: string;
  };
  providers: Record<string, ProviderInfo>;
}

export interface ProviderInfo {
  name: string;
  models: string[];
  baseUrl: string;
}

export interface Suggestion {
  type: string;
  action: string;
  resource: string;
  name: string;
  namespace: string;
  reason: string;
  confidence: number;
  existing: boolean;
  id?: string;
}
