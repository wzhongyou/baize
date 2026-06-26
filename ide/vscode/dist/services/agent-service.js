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
import * as fs from 'node:fs';
import * as path from 'node:path';
import { CangjieAgent, createResilientClient, ToolRegistry, hooks, loadUserMemories, loadProjectMemories, discoverSkills, McpClient, } from '@cangjie/core';
import * as vscode from 'vscode';
export class AgentService {
    agent = null;
    currentConfig = null;
    getConfig() {
        const cfg = vscode.workspace.getConfiguration('cangjie');
        const provider = cfg.get('llm.provider', 'anthropic');
        const config = {
            llm: {
                provider,
                apiKey: cfg.get('llm.apiKey', '') ||
                    (provider === 'anthropic' ? (process.env.ANTHROPIC_API_KEY || process.env.ANTHROPIC_AUTH_TOKEN || '') :
                        provider === 'openai' ? (process.env.OPENAI_API_KEY || '') : (process.env.OPENAI_API_KEY || '')),
                model: cfg.get('llm.model', provider === 'anthropic' ? 'claude-sonnet-4-6' : 'gpt-4o'),
                maxTokens: cfg.get('llm.maxTokens', 8192),
                baseUrl: cfg.get('llm.baseUrl', '') || undefined,
            },
            permissions: {
                autoAllowReadOnly: cfg.get('autoAllowReadOnly', true),
                rules: [],
            },
            context: {
                maxHistoryTokens: cfg.get('context.maxHistoryTokens', 100000),
                compactionThreshold: cfg.get('context.compactionThreshold', 0.85),
                compactionStrategy: 'summarize',
            },
            provider: provider,
        };
        this.currentConfig = config;
        return config;
    }
    async *run(userMessage, signal, onPermissionAsk) {
        const config = this.getConfig();
        if (!config.llm.apiKey) {
            yield { type: 'error', error: '请设置 API Key（环境变量或 VSCode 配置 cangjie.llm.apiKey）' };
            return;
        }
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? process.cwd();
        // Resilient client with retry + fallback
        const { client: llm } = createResilientClient({
            provider: config.llm.provider,
            apiKey: config.llm.apiKey,
            model: config.llm.model,
            baseUrl: config.llm.baseUrl,
            maxTokens: config.llm.maxTokens,
        }, { maxRetries: 3, retryBaseMs: 1000 });
        // Load hooks
        hooks.loadFromWorkspace(workspaceRoot);
        // File change tracking
        const fileChanges = [];
        const tools = this.createTrackedToolRegistry(workspaceRoot, fileChanges);
        // Load MCP servers from config
        const mcpServers = vscode.workspace.getConfiguration('cangjie').get('mcp');
        if (mcpServers) {
            for (const [name, srv] of Object.entries(mcpServers)) {
                try {
                    const client = new McpClient(srv);
                    await client.connect();
                    for (const def of client.tools) {
                        tools.register({
                            definition: def,
                            async execute(args) {
                                return { content: await client.callTool(def.name, args) };
                            },
                        });
                    }
                }
                catch (err) {
                    console.error(`MCP ${name}: ${err.message}`);
                }
            }
        }
        // Build rich system prompt
        const systemPrompt = this.buildSystemPrompt(workspaceRoot);
        // Agent
        this.agent = new CangjieAgent(llm, tools, {
            config,
            workspaceRoot,
            sessionId: `vscode-${Date.now().toString(36)}`,
        });
        // Permission override
        if (onPermissionAsk) {
            this.agent.permission.onAsk(async (tool, args) => {
                return await onPermissionAsk(tool, args);
            });
        }
        const editorContext = this.getEditorContext();
        for await (const event of this.agent.run({
            prompt: editorContext ? `${userMessage}\n\n[当前编辑器上下文]\n${editorContext}` : userMessage,
            systemPrompt,
        }, signal)) {
            yield event;
        }
        // File change events
        for (const change of fileChanges) {
            yield {
                type: 'file_changed',
                filePath: change.filePath,
                preContent: change.preContent,
                postContent: change.postContent,
            };
        }
    }
    createTrackedToolRegistry(workspaceRoot, fileChanges) {
        const registry = new ToolRegistry();
        // Track write_file changes
        const originalWrite = registry.get('write_file');
        if (originalWrite) {
            const tracked = {
                definition: originalWrite.definition,
                execute: async (args, ctx) => {
                    const fp = path.resolve(workspaceRoot, args.file_path);
                    let pre = '';
                    try {
                        pre = fs.readFileSync(fp, 'utf-8');
                    }
                    catch { }
                    const result = await originalWrite.execute(args, ctx);
                    if (!result.error) {
                        const post = args.content;
                        if (pre !== post)
                            fileChanges.push({ filePath: fp, preContent: pre, postContent: post });
                    }
                    return result;
                },
            };
            registry.tools.set('write_file', tracked);
        }
        // Track edit_file changes
        const originalEdit = registry.get('edit_file');
        if (originalEdit) {
            const tracked = {
                definition: originalEdit.definition,
                execute: async (args, ctx) => {
                    const fp = path.resolve(workspaceRoot, args.file_path);
                    let pre = '';
                    try {
                        pre = fs.readFileSync(fp, 'utf-8');
                    }
                    catch { }
                    const result = await originalEdit.execute(args, ctx);
                    if (!result.error) {
                        let post = '';
                        try {
                            post = fs.readFileSync(fp, 'utf-8');
                        }
                        catch { }
                        if (pre !== post)
                            fileChanges.push({ filePath: fp, preContent: pre, postContent: post });
                    }
                    return result;
                },
            };
            registry.tools.set('edit_file', tracked);
        }
        return registry;
    }
    getEditorContext() {
        const editor = vscode.window.activeTextEditor;
        if (!editor)
            return '';
        const doc = editor.document;
        const sel = editor.selection;
        const text = doc.getText(sel);
        let ctx = `文件: ${doc.uri.fsPath}\n语言: ${doc.languageId}\n行数: ${doc.lineCount}\n光标行: ${sel.active.line + 1}`;
        if (text) {
            ctx += `\n\n选中代码:\n\`\`\`${doc.languageId}\n${text}\n\`\`\``;
        }
        return ctx;
    }
    buildSystemPrompt(workspaceRoot) {
        const parts = [];
        parts.push('You are Cangjie, a code agent running in VSCode.');
        parts.push('');
        // User + Project Memory
        try {
            const user = loadUserMemories();
            const project = loadProjectMemories(workspaceRoot);
            if (user.length)
                parts.push('## User Memory\n\n' + user.map((m) => m.content.body).join('\n\n'));
            if (project.length)
                parts.push('## Project Memory\n\n' + project.map((m) => m.content.body).join('\n\n'));
        }
        catch { }
        // Skills
        try {
            const skills = discoverSkills(workspaceRoot);
            if (skills.length)
                parts.push('## Available Skills\n\n' + skills.map((s) => `- ${s.name}: ${s.description}`).join('\n'));
        }
        catch { }
        parts.push('');
        parts.push('## Rules');
        parts.push('- Before writing code, read and understand the existing code first.');
        parts.push('- Prefer edit_file for small changes; use write_file for new files or full rewrites.');
        parts.push("- Keep responses in the user's language.");
        return parts.join('\n');
    }
    dispose() {
        this.agent = null;
    }
}
//# sourceMappingURL=agent-service.js.map