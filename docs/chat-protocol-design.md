# Baize 智能体会话协议设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 实现：`protocol/types.go`

---

## 参考协议

| 协议 | 地址 | 参考点 |
|------|------|--------|
| Anthropic Messages API | https://docs.anthropic.com/en/api/messages | SSE streaming、content blocks、tool use、multimodal |
| OpenAI Chat Completions | https://platform.openai.com/docs/api-reference/chat | role-based messages、function calling、vision |
| AG-UI Protocol | https://docs.ag-ui.com/ | 16 种事件类型、前端工具调用、human-in-the-loop、状态同步 |

Baize 会话协议以 Anthropic Messages API 为主要参考，输出事件类型参考 AG-UI，保持与 OpenAI 兼容的 role/tool_call 结构。

---

## 设计原则

1. **单端点流式**：`POST /api/v1/chat` 一个端点处理所有对话，SSE 持续推送事件直到 `done`
2. **事件粒度最小化**：每个事件只描述一件事（delta、tool 调用、工具结果、完成），客户端按序组装
3. **内容类型扩展**：`answer` 事件默认 Markdown，富内容（图片、代码、HTML）走 `blocks` 字段，客户端按 type 渲染，未知 type 降级到 `fallback_text`
4. **向后兼容**：新增字段均为 `omitempty`，旧客户端忽略未知字段即可

---

## 请求：ChatRequest

`POST /api/v1/chat`，`Content-Type: application/json`

```json
{
  "session_id": "sess-abc123",
  "message": "帮我重构这个函数",
  "images": ["data:image/png;base64,..."],
  "provider": "deepseek",
  "model": "deepseek-r1",
  "max_steps": 30
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `session_id` | string | 否 | 空则自动创建新会话并返回生成的 ID |
| `message` | string | **是** | 用户文本消息 |
| `images` | []string | 否 | base64 图片，支持 `data:image/png;base64,...` 或裸 base64（自动补前缀）。当前支持截图场景 |
| `provider` | string | 否 | 覆盖 llmgate 配置的 provider，如 `"deepseek"`、`"qwen"` |
| `model` | string | 否 | 覆盖 provider 默认模型，如 `"deepseek-r1"` |
| `max_steps` | int | 否 | agent 最大执行步数，默认 30。防止无限循环 |

**images 设计原则**：第一版只支持图片（截图场景），不引入通用 content blocks 请求结构，保持简单。后续若需要文件上传，增加 `POST /api/v1/uploads` 端点，请求中改为 `upload_id` 引用。

---

## 响应：SSE 事件流

响应为 `text/event-stream`，每条 SSE 数据行格式：

```
data: {JSON}\n\n
```

客户端按 `type` 字段分发处理，未知 type 忽略即可。

### 事件类型总览

| type | 触发时机 | 关键字段 |
|------|----------|----------|
| `thought` | LLM 推理 delta（思考模型） | `content` |
| `tool_call` | LLM 决定调用工具 | `tool_name`, `tool_args` |
| `tool_result` | 工具执行完毕 | `content` 或 `blocks` |
| `answer` | LLM 回答 delta | `content` 或 `blocks` |
| `done` | 本轮全部完成 | `tokens` |
| `error` | 执行出错（不可恢复） | `content`（错误信息） |

### ChatEvent 字段说明

```json
{
  "type": "answer",
  "content": "根据代码分析，",
  "blocks": null,
  "tool_name": "",
  "tool_args": null,
  "tokens": 0
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 事件类型，见上表 |
| `content` | string | 纯文本或 Markdown 增量（delta）。`blocks` 非空时此字段为空 |
| `blocks` | []ContentBlock | 富内容块，MCP/Skill 工具返回时使用。优先级高于 `content` |
| `tool_name` | string | 仅 `tool_call` 事件有值，工具名称 |
| `tool_args` | object | 仅 `tool_call` 事件有值，工具参数 JSON |
| `tokens` | int | 仅 `done` 事件有值，本轮累计 token 消耗 |

**answer 流式原理**：`answer` 事件是 delta，客户端 append 拼接成完整回复。`done` 到来时回复已完整，无需再做处理。

### ContentBlock 字段说明

```json
{
  "type": "image",
  "data": "data:image/png;base64,...",
  "meta": {"fallback_text": "生成的图片"}
}
```

| 字段 | 类型 | 适用 type | 说明 |
|------|------|-----------|------|
| `type` | string | 所有 | `"text"` / `"image"` / `"code"` / `"html"` / 自定义 |
| `content` | string | text, code, html | 内容文本 |
| `lang` | string | code | 编程语言，如 `"go"`, `"python"` |
| `data` | string | image | base64 data URL |
| `meta` | object | 所有 | 扩展元数据；`fallback_text` 为未知 type 的降级文本 |

**富内容触发机制**：工具返回结果若以 `{"__baize_blocks":[...]}` 包裹，server 自动解包为 `blocks` 字段推送给客户端，普通字符串结果走 `content`，向后兼容。

---

## 完整 SSE 流示例

```
data: {"type":"thought","content":"用户想重构函数，先读文件..."}

data: {"type":"thought","content":"找到目标函数在第42行"}

data: {"type":"tool_call","tool_name":"file","tool_args":{"action":"read","path":"main.go"}}

data: {"type":"tool_result","content":"func foo() {\n  ..."}

data: {"type":"tool_call","tool_name":"file","tool_args":{"action":"edit","path":"main.go","old_string":"...","new_string":"..."}}

data: {"type":"tool_result","content":"ok"}

data: {"type":"answer","content":"已将 `foo` 函数重构为"}

data: {"type":"answer","content":"更清晰的实现，主要变化：\n\n1. 提取了..."}

data: {"type":"done","tokens":1823}
```

---

## 错误响应

非流式错误（请求格式错误等）返回标准 JSON：

```json
{
  "code": 1001,
  "message": "message is required",
  "request_id": "req-xyz"
}
```

流式过程中出错推送 `error` 事件后关闭流：

```
data: {"type":"error","content":"LLM timeout after 30s"}
```

| 错误码 | 含义 |
|--------|------|
| 1001 | 请求格式错误 |
| 1002 | 未授权 |
| 1003 | 资源不存在 |
| 2001 | 服务内部错误 |
| 3001 | Agent 执行错误 |
| 3002 | 工具执行错误 |
| 3003 | LLM 调用错误 |

---

## 与 AG-UI 协议的对比

AG-UI 定义了约 16 种事件类型，Baize 目前使用其中的核心子集：

| AG-UI 事件 | Baize 对应 | 说明 |
|------------|------------|------|
| `TEXT_MESSAGE_CHUNK` | `answer` delta | 文本回复增量 |
| `TOOL_CALL_START` + `TOOL_CALL_ARGS_DELTA` | `tool_call` | 合并为单事件 |
| `TOOL_CALL_RESULT` | `tool_result` | 工具结果 |
| `RUN_FINISHED` | `done` | 执行完成 |
| `RUN_ERROR` | `error` | 执行出错 |
| `STATE_DELTA` | 未实现 | 前端状态同步（P3） |
| `CUSTOM` | `blocks` 中 custom type | 自定义富内容 |

Human-in-the-loop（用户确认工具执行）通过 permission 系统在 server 侧阻塞，当前不走协议层事件；若未来需要前端 UI 参与确认，可参考 AG-UI `CONFIRM_TOOL_CALL` 模式扩展。
