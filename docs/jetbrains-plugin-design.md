# Baize JetBrains 插件技术设计

> 关联文档：[设计 V1.1](design-v1.1.md) | API 见 [server-design.md](server-design.md) | 会话协议见 [chat-protocol-design.md](chat-protocol-design.md)

---

## 参考调研

| 产品 | 技术方案 |
|------|---------|
| GitHub Copilot JetBrains | Kotlin Plugin + JetBrains AI API，内嵌 Node.js sidecar |
| Continue.dev JetBrains | Kotlin Plugin + 本地 HTTP server，JCef WebView |
| Cursor | 仅 VSCode，无 JetBrains 版 |
| JetBrains AI Assistant | 原生 Kotlin Plugin，Platform API 深度集成 |

Baize JetBrains 插件选型：**Kotlin Plugin + JCef WebView + 本地 HTTP client**，连接 `baize server`（localhost:9779），复用与 VSCode 插件相同的后端协议。

---

## 架构

```
IntelliJ Platform
  ├── Plugin (Kotlin)
  │     ├── BaizePlugin.kt        插件入口、ProjectService 注册
  │     ├── ServerManager.kt      启动/停止 baize server 子进程
  │     ├── BaizeClient.kt        HTTP + SSE 客户端（OkHttp + OkHttp SSE）
  │     ├── SessionManager.kt     会话状态
  │     └── BaizeToolWindow.kt    工具窗口注册
  │
  └── Tool Window (JCef WebView)
        ├── React UI              与 VSCode 插件共享同一 WebView 前端代码
        ├── JS Bridge             Kotlin ↔ JavaScript postMessage
        └── ContentBlock 渲染     富内容块
```

**前端复用**：JCef WebView 与 VSCode WebView 使用同一套 React 代码（通过 message 协议通信），只需适配消息桥接层，UI 逻辑不重复开发。

---

## 核心功能

### 1. Server 生命周期管理

`ProjectService`（随项目打开/关闭）管理 `baize server` 子进程：

```kotlin
class ServerManager(private val project: Project) : Disposable {
    private var process: Process? = null

    fun start() {
        process = ProcessBuilder("baize", "server", "--port", "9779")
            .directory(File(project.basePath!!))
            .start()
    }

    override fun dispose() { process?.destroy() }
}
```

### 2. Tool Window

注册为侧边栏工具窗口（`ToolWindowFactory`），内嵌 JCef WebView：

```kotlin
class BaizeChatPanel(project: Project) : SimpleToolWindowPanel(true) {
    val browser = JBCefBrowser()
    // 加载打包后的 React WebView
    browser.loadURL("file://.../webview/index.html")
}
```

### 3. Kotlin ↔ JavaScript 桥接

```kotlin
// Kotlin → WebView
browser.cefBrowser.executeJavaScript(
    "window.postMessage(${json}, '*')", "", 0
)

// WebView → Kotlin
val query = JBCefJSQuery.create(browser as JBCefBrowserBase)
query.addHandler { request ->
    handleWebViewMessage(request)
    null
}
```

### 4. SSE 流式接收

JetBrains 插件不能直接用 `EventSource`（无浏览器环境），用 OkHttp SSE：

```kotlin
val client = OkHttpClient()
val request = Request.Builder().url("http://localhost:9779/api/v1/chat")
    .post(body).build()

client.newCall(request).enqueue(object : Callback {
    override fun onResponse(call: Call, response: Response) {
        response.body?.source()?.let { source ->
            while (!source.exhausted()) {
                val line = source.readUtf8Line() ?: break
                if (line.startsWith("data: ")) {
                    val event = parseEvent(line.removePrefix("data: "))
                    sendToWebView(event)
                }
            }
        }
    }
})
```

### 5. 权限确认

收到 `permission_request` 事件后，弹出 JetBrains 原生对话框：

```kotlin
val result = Messages.showDialog(
    project,
    "Agent 请求执行：$tool\n命令：$cmd",
    "权限确认",
    arrayOf("允许", "拒绝", "始终允许"),
    0,  // default = 允许
    Messages.getWarningIcon()
)
// 发 POST /api/v1/sessions/{id}/confirm
```

### 6. IDE 上下文注入

通过 IntelliJ Platform API 获取：

```kotlin
val editor = FileEditorManager.getInstance(project).selectedTextEditor
val selectedText = editor?.selectionModel?.selectedText
val currentFile = editor?.virtualFile?.path
val diagnostics = InspectionManager.getInstance(project)
    // 获取当前文件的检查问题
```

### 7. 注册的 Action

| Action ID | 快捷键 | 说明 |
|-----------|--------|------|
| `Baize.OpenChat` | `Ctrl+Shift+B` | 打开聊天窗口 |
| `Baize.ExplainCode` | — | 解释选中代码（右键菜单） |
| `Baize.FixProblem` | — | 修复当前问题（Problems 视图） |
| `Baize.NewSession` | — | 新建会话 |

---

## plugin.xml 关键配置

```xml
<idea-plugin>
  <id>com.wzhongyou.baize</id>
  <depends>com.intellij.modules.platform</depends>

  <extensions defaultExtensionNs="com.intellij">
    <toolWindow id="Baize" anchor="right"
      factoryClass="com.wzhongyou.baize.BaizeToolWindowFactory"/>
    <projectService
      serviceImplementation="com.wzhongyou.baize.ServerManager"/>
  </extensions>

  <actions>
    <action id="Baize.OpenChat" class="com.wzhongyou.baize.OpenChatAction"
      text="Open Baize Chat" keymap="ctrl shift B"/>
  </actions>
</idea-plugin>
```

---

## 与 VSCode 插件的差异

| 维度 | VSCode | JetBrains |
|------|--------|-----------|
| 语言 | TypeScript | Kotlin |
| WebView | VSCode WebView API | JCef (Chromium Embedded) |
| SSE 客户端 | fetch + ReadableStream | OkHttp SSE |
| 权限弹窗 | `window.showWarningMessage` | `Messages.showDialog` |
| 进程管理 | Node.js `spawn` | `ProcessBuilder` |
| 前端代码 | 共享同一 React bundle | 共享同一 React bundle |

---

## 实现优先级

| 优先级 | 内容 |
|--------|------|
| P2 | server 生命周期管理 + Tool Window 框架 |
| P2 | JCef WebView + 消息桥接 |
| P2 | SSE 流式接收 + 权限确认 |
| P3 | IDE 上下文注入 |
| P3 | Action 注册 + 右键菜单 |
| P3 | 历史会话列表 |
