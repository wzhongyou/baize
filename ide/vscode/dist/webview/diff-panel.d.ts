/**
 * Diff Review Panel
 *
 * Agent 任务完成后展示文件变更的 Diff 审查面板。
 * 用户可逐文件 Accept / Reject。
 */
import * as vscode from 'vscode';
export interface FileChange {
    filePath: string;
    preContent: string;
    postContent: string;
}
export declare class DiffPanel {
    static currentPanel: DiffPanel | undefined;
    private readonly panel;
    private readonly changes;
    private readonly onAcceptAll;
    private disposables;
    private constructor();
    static createOrShow(context: vscode.ExtensionContext, changes: FileChange[], onAcceptAll?: () => void): DiffPanel;
    /** 撤销某个文件的修改 */
    private rejectChange;
    private postToWebview;
    private dispose;
    private getHtml;
}
//# sourceMappingURL=diff-panel.d.ts.map