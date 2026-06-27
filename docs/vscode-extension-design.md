# Baize VSCode 插件技术设计

> 关联文档：[设计 V1.1](design-v1.1.md) | API 见 [server-design.md](server-design.md) | 会话协议见 [chat-protocol-design.md](chat-protocol-design.md)

---

## 参考调研

| 产品 | 技术方案 |
|------|---------|
| Claude Code | VSCode Extension API + WebView Panel，内嵌 Node.js agent，不走本地 server |
| Continue.dev | VSCode Extension + 本地 HTTP server，WebView 渲染 React UI |
| Cursor | 深度 fork VSCode，内嵌 AI 能力，非标准插件 |
| GitHub Copilot | VSCode Extension API，Language Model API 直调 |

Baize VSCode 插件选型：**Extension + WebView + 本地 HTTP client**，连接 `baize server`（localhost:9779）。

---

## 架构

```
VSCode
  ├── Extension Host (Node.js)
  │     ├── extension.ts          激活、注册命令、管理 server 进程
  │     ├── server-manager.ts     启动/停止 baize server 子进程
  │     ├── baize-client.ts       HTTP + SSE 客户端（fetch + EventSource）
  │     └── session-manager.ts   会话状态管理
  │
  └── WebView Panel
        ├── React UI              聊天界面、工具执行展示、权限确认弹窗
        ├── Markdown 渲染         answer 内容渲染（含代码高亮）
        └── ContentBlock 渲染     富内容块（image/code/html）
```

---

## 核心功能

### 1. Server 生命周期管理

插件激活时检查 `baize server` 是否已在运行（`GET /api/v1/health`），未运行则启动子进程：

```typescript
const proc = spawn('baize', ['server', '--port', '9779'], {
    cwd: workspace.rootPath,
    detached: false,  // 跟随 VSCode 退出
})
```

插件停用时通过 `proc.kill()` 关闭。

### 2. 聊天界面

WebView Panel（`createWebviewPanel`）展示对话，通过 `postMessage` 与 Extension Host 通信：

```
WebView ──postMessage({type:'send', text, images})──→ Extension Host
Extension Host ──postMessage({type:'event', ...})──→ WebView
```

支持：
- Markdown + 代码高亮渲染
- 图片粘贴（`ClipboardEvent` → base64 → `images` 字段）
- 工具执行气泡（tool_call + tool_result）
- 思考过程折叠展示（thought 事件）

### 3. SSE 流式接收

```typescript
const es = new EventSource(`http://localhost:9779/api/v1/chat`, {
    method: 'POST',  // 实际用 fetch + ReadableStream
})
// 每个 ChatEvent 通过 postMessage 推到 WebView
```

### 4. 权限确认

收到 `permission_request` 事件后，Extension Host 弹出 VSCode 原生 `window.showInformationMessage` 确认框，或在 WebView 内展示内联确认 UI：

```typescript
const decision = await vscode.window.showWarningMessage(
    `Agent 请求执行：${tool}\n命令：${cmd}`,
    '允许', '拒绝', '始终允许'
)
// 发 POST /api/v1/sessions/{id}/confirm
```

### 5. IDE 上下文注入

插件可以把当前编辑器上下文自动注入消息：
- 当前打开文件路径
- 选中代码（作为引用）
- 诊断错误（Problems 面板的错误）

通过 `vscode.window.activeTextEditor` 和 `vscode.languages.getDiagnostics` 获取。

### 6. 命令面板

注册 VSCode 命令：

| 命令 | 说明 |
|------|------|
| `baize.openChat` | 打开聊天面板 |
| `baize.explainCode` | 解释选中代码 |
| `baize.fixDiagnostic` | 修复当前诊断错误 |
| `baize.newSession` | 新建会话 |
| `baize.resumeSession` | 恢复历史会话 |

### 7. Inline Chat（计划中）

参考 GitHub Copilot，在编辑器内 `Ctrl+I` 唤起 inline chat，结果直接应用到光标位置。

---

## package.json 主要依赖

```json
{
  "engines": {"vscode": "^1.90.0"},
  "activationEvents": ["onStartupFinished"],
  "contributes": {
    "commands": [...],
    "keybindings": [{"key": "ctrl+shift+b", "command": "baize.openChat"}]
  }
}
```

运行时无 npm 依赖（纯 VSCode API + fetch，不引入 React 等库到 Extension Host；WebView 端打包 React）。

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P1 | server 生命周期管理 + health 检查 |
| P1 | WebView 聊天 UI + SSE 流式渲染 |
| P1 | 图片粘贴上传 |
| P1 | 权限确认（VSCode 原生弹窗） |
| P2 | IDE 上下文注入（文件/选中代码/诊断） |
| P2 | 命令面板命令 |
| P3 | Inline Chat |
| P3 | 历史会话列表 sidebar |
