# Contributing to Baize

感谢你对 Baize（白泽）的关注！无论是 Bug 修复、功能提案、文档改进还是使用反馈，都欢迎参与。

## 行为准则

本项目遵循 [Contributor Covenant 行为准则](CODE_OF_CONDUCT.md)。

## 如何贡献

### 报告 Bug

1. 使用 [Bug Report](https://github.com/wzhongyou/baize/issues/new?template=bug_report.md) 模板
2. 描述清晰：做了什么、期望什么、实际发生了什么
3. 提供复现步骤和环境信息（OS、Go 版本等）

### 功能提案

1. 先开 Issue 讨论，避免白做
2. 说清楚：解决什么问题、为什么重要、建议怎么做
3. 如果是大特性，参考 [路线图](docs/upgrade-roadmap.md) 看是否已有规划

### 提交代码

```bash
# 1. Fork + Clone
git clone https://github.com/YOUR_USERNAME/baize.git
cd baize

# 2. 创建分支
git checkout -b feat/my-feature

# 3. 开发 + 测试
go test ./...
go vet ./...
make lint

# 4. 提交（遵循 Conventional Commits）
git commit -m "feat: add something useful"

# 5. 推送 + 开 PR
git push origin feat/my-feature
```

### Commit 规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
feat:     新功能
fix:      Bug 修复
docs:     文档更新
refactor: 重构
test:     测试
chore:    构建/工具
```

### 代码风格

- `go fmt` 之后提交
- 导出函数/类型必须有 Go doc 注释
- 新功能尽量带测试
- 保持简单，不要过度抽象

## 开发环境

```bash
# 依赖
Go >= 1.25
Node.js >= 20（Web Dashboard 开发）

# 构建
make build

# 运行
./baize

# Web Dashboard 开发
cd web && npm install && npm run dev
```

## 项目结构

```
baize/
├── agent/       # Agent 核心抽象
├── tool/        # 工具系统
├── server/      # API Server
├── web/         # Web Dashboard (React)
├── cmd/baize/   # CLI 入口
└── docs/        # 文档
```

## 获取帮助

- [文档](docs/)
- [架构设计](docs/architecture.md)
- [Discussions](https://github.com/wzhongyou/baize/discussions)

---

感谢你的贡献！🎉
