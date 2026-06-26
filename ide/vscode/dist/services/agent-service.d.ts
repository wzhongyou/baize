/**
 * Agent Service — VSCode Extension 和 Agent Runtime 之间的桥接层
 *
 * v0.2 更新：
 * - 多模型 Provider 支持
 * - Resilient client（自动重试/降级）
 * - Skills + Memory + Hooks 自动加载
 * - MCP 工具注册
 * - 文件变更追踪
 */
import type { AgentEvent, PermissionDecision } from '@cangjie/shared';
export type PermissionAskCallback = (tool: string, args: Record<string, unknown>) => Promise<PermissionDecision>;
export declare class AgentService {
    private agent;
    private currentConfig;
    private getConfig;
    run(userMessage: string, signal?: AbortSignal, onPermissionAsk?: PermissionAskCallback): AsyncGenerator<AgentEvent>;
    private createTrackedToolRegistry;
    private getEditorContext;
    private buildSystemPrompt;
    dispose(): void;
}
//# sourceMappingURL=agent-service.d.ts.map