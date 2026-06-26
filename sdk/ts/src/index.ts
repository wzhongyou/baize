/**
 * Baize SDK — TypeScript client for the Baize Agent API.
 *
 * Usage:
 *   import { BaizeClient } from '@cangjie/baize-sdk';
 *   const client = new BaizeClient('http://localhost:9779');
 *   for await (const event of client.chat({ message: 'hello' })) { ... }
 *
 * Protocol: REST + SSE, matches Baize's Go api/ package.
 */

// ── Types ──────────────────────────────────────────────────────────────────

export interface ApiResponse<T = unknown> {
  code: number;
  data: T;
  message?: string;
  request_id: string;
}

export interface HealthResponse {
  status: string;
  version: string;
}

export interface ChatRequest {
  session_id?: string;
  message: string;
  provider?: string;
  model?: string;
  max_steps?: number;
}

export interface ChatEvent {
  type: 'thought' | 'tool_call' | 'tool_result' | 'answer' | 'done' | 'error';
  content?: string;
  tool_name?: string;
  tool_args?: Record<string, unknown>;
  tokens?: number;
}

export interface ToolInfo {
  name: string;
  description: string;
  parameters: Record<string, unknown>;
  read_only: boolean;
  source: string;
}

export interface CallToolRequest {
  session_id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export interface CallToolResponse {
  content: string;
  error?: string;
}

export interface SessionInfo {
  id: string;
  title: string;
  workspace_root?: string;
  model?: string;
  step_count: number;
  total_tokens: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  role: string;
  content: string;
  tool_calls?: ToolCall[];
  tool_name?: string;
  timestamp: string;
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export interface SessionDetail extends SessionInfo {
  messages: Message[];
}

export interface MemorySearchResult {
  content: string;
  score: number;
  metadata?: Record<string, unknown>;
}

// ── Client ─────────────────────────────────────────────────────────────────

export class BaizeClient {
  private baseURL: string;

  constructor(baseURL = 'http://localhost:9779') {
    this.baseURL = baseURL.replace(/\/$/, '');
  }

  /** Check server health. */
  async health(): Promise<HealthResponse> {
    const resp = await this.get<HealthResponse>('/api/v1/health');
    return resp.data;
  }

  /**
   * Start a streaming agent chat.
   * Returns an async generator of ChatEvents.
   */
  async *chat(req: ChatRequest): AsyncGenerator<ChatEvent> {
    const response = await fetch(`${this.baseURL}/api/v1/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
      body: JSON.stringify(req),
      signal: AbortSignal.timeout?.(10 * 60 * 1000), // 10 minute timeout
    });

    if (!response.ok) {
      const err = await this.parseError(response);
      throw err;
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error('No response body');

    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6);
            try {
              const event = JSON.parse(data) as ChatEvent;
              yield event;
              if (event.type === 'done' || event.type === 'error') return;
            } catch {
              // skip malformed events
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  /** List registered tools. */
  async listTools(): Promise<ToolInfo[]> {
    const resp = await this.post<{ tools: ToolInfo[] }>('/api/v1/tools/list');
    return resp.data.tools;
  }

  /** Call a tool directly. */
  async callTool(req: CallToolRequest): Promise<CallToolResponse> {
    const resp = await this.post<CallToolResponse>('/api/v1/tools/call', req);
    return resp.data;
  }

  /** List all sessions. */
  async listSessions(): Promise<SessionInfo[]> {
    const resp = await this.get<{ sessions: SessionInfo[] }>('/api/v1/sessions');
    return resp.data.sessions;
  }

  /** Create a new session. */
  async createSession(title: string, workspace?: string): Promise<string> {
    const resp = await this.post<{ id: string }>('/api/v1/sessions', {
      title,
      workspace_root: workspace,
    });
    return resp.data.id;
  }

  /** Get a session with messages. */
  async getSession(id: string): Promise<SessionDetail> {
    const resp = await this.get<SessionDetail>(`/api/v1/sessions/${id}`);
    return resp.data;
  }

  /** Delete a session. */
  async deleteSession(id: string): Promise<void> {
    await this.del(`/api/v1/sessions/${id}`);
  }

  /** Search long-term memory. */
  async searchMemory(query: string, topK = 5): Promise<MemorySearchResult[]> {
    const resp = await this.post<{ results: MemorySearchResult[] }>('/api/v1/memory/search', {
      query,
      top_k: topK,
    });
    return resp.data.results;
  }

  /** Save to long-term memory. */
  async saveMemory(content: string, metadata?: Record<string, unknown>): Promise<void> {
    await this.post('/api/v1/memory/save', { content, metadata });
  }

  // ── Internal HTTP helpers ────────────────────────────────────────────

  private async get<T>(path: string): Promise<ApiResponse<T>> {
    const resp = await fetch(`${this.baseURL}${path}`);
    if (!resp.ok) throw await this.parseError(resp);
    return resp.json() as Promise<ApiResponse<T>>;
  }

  private async post<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    const resp = await fetch(`${this.baseURL}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!resp.ok) throw await this.parseError(resp);
    return resp.json() as Promise<ApiResponse<T>>;
  }

  private async del(path: string): Promise<void> {
    const resp = await fetch(`${this.baseURL}${path}`, { method: 'DELETE' });
    if (!resp.ok) throw await this.parseError(resp);
  }

  private async parseError(resp: Response): Promise<Error> {
    try {
      const body = await resp.json() as ApiResponse;
      if (body.message) {
        return new Error(`Baize API [${resp.status}]: ${body.message}`);
      }
    } catch {}
    return new Error(`Baize API HTTP ${resp.status}`);
  }
}
