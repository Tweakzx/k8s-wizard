import type {
  ChatRequest,
  ChatResponse,
  ModelInfoResponse,
  ConfigResponse,
} from '../types';

const API_BASE = (import.meta as any).env?.VITE_API_URL || '/api';

const ENV_API_TOKEN =
  (import.meta as any).env?.VITE_API_TOKEN ||
  (import.meta as any).env?.VITE_AUTH_TOKEN ||
  '';

function resolveAuthToken(): string {
  const runtimeToken =
    (globalThis as any).__K8S_WIZARD_API_TOKEN__ ||
    (globalThis as any).__APP_CONFIG__?.apiToken ||
    '';
  return runtimeToken || ENV_API_TOKEN;
}

function withAuthHeaders(headers: HeadersInit = {}): HeadersInit {
  const merged = new Headers(headers);
  if (!merged.has('Authorization')) {
    const token = resolveAuthToken();
    if (token) {
      merged.set('Authorization', token.startsWith('Bearer ') ? token : `Bearer ${token}`);
    }
  }
  return merged;
}

export const api = {
  // 健康检查
  async checkHealth(): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE.replace('/api', '')}/health`, {
        headers: withAuthHeaders(),
      });
      const data = await response.json();
      return data.status === 'ok';
    } catch (error) {
      console.error('Health check failed:', error);
      return false;
    }
  },

  // 发送消息
  async sendMessage(content: string): Promise<{ result: string; model: string }> {
    const request: ChatRequest = { content };

    const response = await fetch(`${API_BASE}/chat`, {
      method: 'POST',
      headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const errorData = (await response.json()) as ChatResponse;
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    const data = (await response.json()) as ChatResponse;
    return {
      result: data.result,
      model: data.model || 'unknown',
    };
  },

  // 发送聊天请求（支持表单补充与确认执行）
  async sendChat(request: ChatRequest): Promise<ChatResponse> {
    const response = await fetch(`${API_BASE}/chat`, {
      method: 'POST',
      headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const errorData = (await response.json()) as ChatResponse;
      return { error: errorData.error || `服务器错误: ${response.status}`, result: '' };
    }

    return (await response.json()) as ChatResponse;
  },

  // 获取资源
  async getResources(type: string = 'pod', namespace: string = 'default'): Promise<string> {
    const params = new URLSearchParams({
      type,
      namespace,
    });
    const response = await fetch(`${API_BASE}/resources?${params.toString()}`, {
      headers: withAuthHeaders(),
    });

    if (!response.ok) {
      const errorData = (await response.json()) as ChatResponse;
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    const data = (await response.json()) as ChatResponse;
    return data.result;
  },

  // 获取模型信息
  async getModelInfo(): Promise<ModelInfoResponse> {
    const response = await fetch(`${API_BASE}/config/model`, {
      headers: withAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: Failed to get model info`);
    }

    return (await response.json()) as ModelInfoResponse;
  },

  // 获取完整配置
  async getConfig(): Promise<ConfigResponse> {
    const response = await fetch(`${API_BASE}/config`, {
      headers: withAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: Failed to get config`);
    }

    return (await response.json()) as ConfigResponse;
  },

  // 切换模型
  async setModel(model: string): Promise<{ success: boolean; model: string; message?: string }> {
    const response = await fetch(`${API_BASE}/config/model`, {
      method: 'PUT',
      headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ model }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || `HTTP ${response.status}`);
    }

    const data = await response.json();
    return {
      success: data.success,
      model: data.model,
      message: data.message,
    };
  },
}
