# Baize 编辑器产品方向讨论

> 状态：探索阶段，尚未立项

---

## 背景

Baize 当前定位是 AI 编程 Agent（CLI + API Server）。本文讨论是否以及如何在此基础上构建类 Cursor 的独立编辑器产品。

---

## 竞品参考：OpenCode 架构

OpenCode 是最接近的参考案例，monorepo 结构，核心与客户端完全分离：

```
packages/core      — agent 逻辑（无 UI 依赖）
packages/server    — HTTP 薄包装层
packages/sdk/js    — 公开 SDK（含 createOpencodeClient / createOpencodeServer）
packages/tui       — 终端 UI，通过 SDK HTTP 通信
packages/app       — Web/桌面 UI（SolidJS）
packages/desktop   — Electron 包装
```

TUI、Web、桌面 App 都通过 HTTP 连接同一个 server，VSCode 插件直接起 TUI 进程复用。**核心只有一份，任何客户端都能接**。

---

## Baize 的映射关系

Baize 现在的结构已经接近：

```
baize/
  core/       — agent 引擎（无 UI 依赖）✅
  server/     — HTTP 层 ✅
  cli/        — TUI + CLI（直连 core）
  protocol/   — 共享类型（接近 SDK 作用）
```

如果要做独立编辑器产品，关系可以是：

```
baize/           — AI 后端引擎（Go，保持现状）
baize-editor/    — 编辑器前端（Tauri + Monaco 或 Electron）
                   通过 HTTP 连接 baize server
```

baize 成为"AI 后端"，编辑器是前端，通过 `/api/v1/chat` 协议解耦。

---

## 类 Cursor 产品需要做的事

### 编辑器基础层（最重）

三条路：

| 路线 | 代表 | 工程量 | 生态 |
|------|------|--------|------|
| Fork VSCode | Cursor | 极高（跟主线维护） | 最好（所有 VSCode 插件） |
| Monaco + Electron/Tauri | OpenCode app | 中 | 一般（无 LSP 生态） |
| 自建编辑器 | Zed（Rust）| 最高 | 从零 |

**推荐**：Monaco + Tauri（Rust 包装，比 Electron 轻），3-6 个月可出可用版本。

### 代码理解层（核心壁垒）

Cursor 真正的技术壁垒——知道"哪些上下文跟当前问题相关"：

- 代码库索引：AST 解析 + 符号图（Tree-sitter）
- 向量化：代码片段 embedding + 语义搜索
- 上下文构建：当前文件 + 相关文件 + 符号定义 + git diff 智能组合

Baize 目前只有 grep（关键词搜索），语义搜索是空白。

### AI 交互模式

| 功能 | 难度 | Baize 现状 |
|------|------|-----------|
| Chat 面板 | 低 | ✅ 已有 |
| Agent 模式（多文件） | 中 | ✅ 已有 |
| Inline Chat（Cmd+K）| 中 | ❌ 需编辑器集成 |
| Tab 补全（FIM）| 高 | ❌ 需 FIM 模型 |
| Diff 视图（接受/拒绝）| 中 | ❌ 需编辑器集成 |

---

## 结论与建议

**不需要新仓库**。把 `baize` 定位为 AI 后端引擎，前端编辑器单独建仓库 `baize-editor` 通过 HTTP 接入。

优先级建议：
1. **P0**：先把 baize CLI/Server 做扎实（V1.1 路线图），这是所有客户端的基础
2. **P1**：VSCode / JetBrains 插件（已有设计文档），最快触达用户
3. **P2**：baize-editor（Monaco + Tauri），需要代码库索引能力先就位
4. **P3**：Tab 补全（需要专用 FIM 模型或 API）

**编辑器的核心差异化**不在于编辑器本身，而在于代码理解质量。先投资代码库索引和语义搜索，再包一个编辑器外壳，顺序不能反。
