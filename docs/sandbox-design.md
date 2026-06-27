# Baize 沙箱设计

> 关联文档：[设计 V1.1](design-v1.1.md) | agent 执行流程见 [agent-engine-design.md](agent-engine-design.md)
> 沙箱实现：`github.com/wzhongyou/carrel`

---

## 参考调研

| 产品 | macOS | Linux | 隔离强度 |
|------|-------|-------|---------|
| Codex CLI | `sandbox-exec` Seatbelt（`(deny default)` + 动态 WRITABLE_ROOT） | bubblewrap + seccomp（`ptrace`/`io_uring` 等拦截） + Landlock（辅助） | OS 强制 |
| Claude Code | `sandbox-exec` 可选（settings.json） | 无 | 软策略 |
| OpenCode | 无 | 无 | 软策略（tree-sitter AST） |

---

## carrel 现状评估

`github.com/wzhongyou/carrel` 架构已就绪，缺两个 OS 级 backend：

| 能力 | 状态 |
|------|------|
| 危险命令策略规则（50+ 规则） | ✅ |
| 用户确认流程（MemoryApprover） | ✅ |
| 审计日志（AuditEvent） | ✅ |
| 资源限制（timeout/output） | ✅（rlimit 未连接） |
| macOS Seatbelt backend | ❌ 待实现 |
| Linux bubblewrap + seccomp backend | ❌ 待实现 |

Backend SPI 已定义：

```go
type Backend interface {
    Prepare(ctx, workspace, tempDir string) (interface{}, error)
    Run(ctx, handle interface{}, cmd Command) (Result, error)
    Destroy(ctx, handle interface{}) error
}
```

---

## 沙箱策略

与 agent 角色绑定：

| carrel Profile | agent 角色 | 文件系统 | 网络 |
|----------------|-----------|---------|------|
| `read-only` | explore | 只读，限 workspace | 全拒 |
| `workspace-write` | edit, test | workspace 读写，系统只读 | 全拒 |
| `break-glass` | 用户明确授权 | 无限制 | 允许 |

---

## macOS Backend：Seatbelt

参考 Codex `seatbelt.rs`，`Run()` 把命令包装为：

```
/usr/bin/sandbox-exec -p <policy> -DWORKSPACE=/path/to/repo -- <cmd>
```

**Policy 组成**（运行时动态拼接）：

```scheme
; 基础：closed by default
(deny default)
(allow process-exec process-fork)
(allow signal (target same-sandbox))
(allow file-write-data (literal "/dev/null"))
(allow pseudo-tty)

; workspace 写权限（参数注入）
(allow file-read* file-write* (subpath (param "WORKSPACE")))

; 平台只读路径（read-only profile 时不加 workspace 写权限）
(allow file-read* (subpath "/usr/lib"))
(allow file-read* (subpath "/usr/local/lib"))
(allow file-read* (subpath "/opt/homebrew/lib"))
(allow file-read* (subpath "/System/Library/Frameworks"))
(allow file-read* (subpath "/tmp"))
(allow file-write* (subpath "/tmp"))

; 网络（break-glass 才开启）
; (allow network-outbound)
```

保护路径（`.git`、`.baize` 在 workspace 内强制只读）：

```scheme
(deny file-write* (subpath (param "WORKSPACE_GIT")))
```

---

## Linux Backend：bubblewrap + seccomp

参考 Codex `bwrap.rs` + `linux-sandbox`。`Run()` 包装为：

```
bwrap \
  --ro-bind / /          # 全局只读底座
  --dev /dev
  --bind <workspace> <workspace>   # workspace 可写
  --ro-bind <workspace>/.git <workspace>/.git  # .git 保护
  --tmpfs /tmp
  --unshare-user --unshare-pid
  --unshare-net            # 网络隔离（break-glass 时去掉）
  --new-session --die-with-parent
  -- <cmd>
```

**seccomp filter**（`go-seccomp-bpf`，纯 Go，无 CGO）：

```go
// 始终拒绝的危险 syscall
denied := []string{
    "ptrace",
    "process_vm_readv", "process_vm_writev",
    "io_uring_setup", "io_uring_enter", "io_uring_register",
}

// 网络隔离时额外拒绝
if networkIsolated {
    denied = append(denied,
        "connect", "bind", "listen", "accept", "accept4",
        "sendto", "sendmmsg", "recvmmsg",
    )
    // socket 仅允许 AF_UNIX
}
```

`PR_SET_NO_NEW_PRIVS` 在 seccomp 安装前设置。

---

## Backend 自动选择

```go
// backend/auto.go
func Auto(cfg Config) Backend {
    switch runtime.GOOS {
    case "darwin": return seatbelt.New(cfg)
    case "linux":  return bwrap.New(cfg)
    default:       return process.New(cfg)  // 软策略 fallback
    }
}
```

`process` backend 是当前实现（`Setpgid` + `Pdeathsig`），作为不支持平台的安全回退。

---

## 与 Baize 集成

`ShellTool.Execute` 替换为 `carrel.Sandbox.Run`：

```go
// core/tool/builtin/shell.go
result, err := s.sandbox.Run(ctx, carrel.Command{
    Args: []string{"sh", "-c", cmd},
    CWD:  s.WorkspaceRoot,
})
```

`Sandbox` 实例在 `buildToolRegistry` 时构建，Profile 由当前 agent 角色决定（通过 ctx 传递）。

---

## 实现优先级

| 优先级 | 内容 | 状态 |
|--------|------|------|
| P0 | process backend（软策略+资源限制） | ✅（carrel 已有）|
| P0 | 策略规则（50+ 危险命令规则） | ✅ |
| P0 | MemoryApprover + 确认流程 | ✅ |
| P1 | macOS Seatbelt backend | 待实现（carrel）|
| P1 | Linux bubblewrap + seccomp backend | 待实现（carrel）|
| P1 | ShellTool 接入 carrel | 待实现（baize）|
| P2 | rlimit 连接（pids/openfiles） | 待实现 |
| P3 | bubblewrap + Landlock 双重文件系统隔离 | 待实现 |
