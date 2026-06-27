import * as vscode from 'vscode';
import { BaizeAgentService } from './services/agent-service';

export function activate(context: vscode.ExtensionContext) {
  const agentService = new BaizeAgentService('http://127.0.0.1:9779');

  context.subscriptions.push(
    vscode.commands.registerCommand('baize.openChat', () => {
      // TODO: Open Baize chat webview
    }),
    vscode.commands.registerCommand('baize.explainCode', () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor) return;
      const selection = editor.document.getText(editor.selection);
      agentService.chat(selection || editor.document.getText()).catch(console.error);
    })
  );
}

export function deactivate() {}
