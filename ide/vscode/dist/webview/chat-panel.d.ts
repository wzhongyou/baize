/**
 * Chat Webview Panel
 *
 * Extension Host 侧的 Chat 面板管理。
 * 负责 Webview ↔ AgentService 之间的消息转发和事件流推送。
 */
import * as vscode from 'vscode';
export declare class ChatPanel {
    static currentPanel: ChatPanel | undefined;
    private readonly panel;
    private readonly agentService;
    private readonly context;
    private disposables;
    private abortController;
    private pendingPermissions;
    private constructor();
    static createOrShow(context: vscode.ExtensionContext): ChatPanel;
    /** 从外部命令触发（如 explainCode） */
    sendMessage(content: string): void;
    /** 权限确认回调 */
    private createPermissionCallback;
    private handlePermissionReply;
    /** 处理用户输入 → 启动 Agent */
    private handleUserMessage;
    /** 将 AgentEvent 转为 Webview 消息 */
    private postAgentEvent;
    private abortAgent;
    private postToWebview;
    /** 内联 HTML + JS */
    private getHtml;
    private dispose;
}
//# sourceMappingURL=chat-panel.d.ts.map