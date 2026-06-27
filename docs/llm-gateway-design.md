# Baize 模型网关设计

> 关联文档：[设计 V1.1](design-v1.1.md) | 实现：`core/agent/llmgate/`、`github.com/wzhongyou/llmgate`

---

## 参考调研

| 产品 | Provider 数量 | 路由策略 | 缓存 | 成本追踪 | 可观测性 |
|------|-------------|---------|------|---------|---------|
| LiteLLM | 100+ | 延迟/成本/负载均衡/加权随机 | 语义缓存(Redis) | 每 token 定价表 + 预算限制 | Redis/DB 持久化 |
| OpenRouter | 200+ 模型 | 服务端自动（可用性+价格） | 无 | 响应头返回每次成本 | 无自托管 |
| Portkey | 50+ | Fallback、金丝雀测试、条件路由 | 语义缓存 | 按用户/项目预算 | gRPC + 持久化 |
| **llmgate（现状）** | ~27（中文生态覆盖强） | Primary-First、延迟阈值、时段切换 | 无 | 无 | 进程内统计（不持久） |

---

## llmgate 现状分析

### 优势

- **中文 LLM 覆盖**：DeepSeek、Qwen、Kimi、Baichuan、文心、GLM、混元、讯飞、MiniMax 等 15+ 国内 provider，这是 LiteLLM 等不具备的
- **模型特性完整**：ThinkingType（推理模式）、ContentParts（视觉）、tool use 完整支持，ReasoningContent 透传
- **韧性原语**：熔断器（5次失败/30s恢复）、指数退避重试（2次）
- **策略组合**：多种路由策略可组合（Primary-First + Latency + TimeBased）

### 关键缺口

| 缺口 | 影响 | 优先级 |
|------|------|--------|
| 无成本追踪（无定价表） | 无法感知 token 花费、无法限额 | P1 |
| 无 Prompt Cache 感知 | KV Cache 命中率优化无反馈 | P1 |
| 无负载均衡（多 key/多实例） | 单 API key 限速时无法分散 | P2 |
| 无语义缓存 | 相似请求重复付费 | P3 |
| 指标不持久化 | 无历史可分析 | P2 |

---

## 模型网关设计

### 架构层次

```
Baize Agent
    │  agent.LLMModel 接口
    ▼
llmgate Adapter（core/agent/llmgate/adapter.go）
    │  转换 agent.Message → core.Message，处理 vision、tools
    ▼
llmgate Gateway（github.com/wzhongyou/llmgate）
    │  路由策略 + 熔断 + 重试
    ▼
Provider（DeepSeek / Qwen / OpenAI / ...）
```

### 路由策略扩展

当前策略已能满足基础需求，补充两个高优先级策略：

**成本感知路由**（P1）：

```go
// 为每个 provider 配置每 token 价格
[providers.deepseek-v3]
input_price_per_1k  = 0.0014  // USD/1K input tokens
output_price_per_1k = 0.0028

// CostBasedStrategy: 优先选择最低成本 provider（在延迟阈值内）
```

**多 key 负载均衡**（P2）：

```go
// 同一 provider 配置多个 API key，RoundRobin 分散请求
[providers.qwen]
api_keys = ["sk-xxx1", "sk-xxx2", "sk-xxx3"]
strategy = "round_robin"
```

### Prompt Cache 感知（P1）

KV Cache 命中可降低 80%+ 的 input token 成本。llmgate 需要：

1. 在支持 prompt cache 的 provider（Anthropic、DeepSeek）上自动注入 `cache_control` 标记
2. 响应中解析 `cache_read_input_tokens` vs `cache_miss_input_tokens`
3. 在统计中区分缓存命中成本 vs 实际消耗成本

```go
type Usage struct {
    InputTokens           int
    OutputTokens          int
    CacheReadTokens       int  // 新增
    CacheMissTokens       int  // 新增
    TotalTokens           int
}
```

### 成本追踪（P1）

```go
type CostRecord struct {
    Provider    string
    Model       string
    InputCost   float64
    OutputCost  float64
    CachedCost  float64
    TotalCost   float64
    Timestamp   time.Time
}
```

写入本地 SQLite（与会话 DB 同一文件），支持按会话/按天/按 provider 汇总。

---

## 与 llmgate 的差距结论

llmgate 的核心功能（provider 覆盖、流式、工具调用、推理模式、熔断重试）**已满足 Baize V1.1 需求**，无需替换。

需要在 llmgate 上补充的能力（优先级排序）：

1. **成本追踪 + 定价表**（P1）— 最高优先，影响用户使用感知
2. **Prompt Cache 感知**（P1）— 直接影响 token 成本
3. **多 key 负载均衡**（P2）— 高并发/限速场景
4. **指标持久化**（P2）— 可观测性
5. **语义缓存**（P3）— 收益依赖使用模式

这些都是在现有 llmgate 架构上的增量扩展，不需要引入 LiteLLM 等外部网关。

---

## 实现优先级

| 优先级 | 内容 | 状态 |
|--------|------|------|
| P0 | provider 覆盖、流式、工具调用、熔断重试 | ✅（llmgate 已有）|
| P1 | 成本追踪 + 定价表 | 待实现 |
| P1 | Prompt Cache 感知（cache_read_tokens） | 待实现 |
| P2 | 多 key RoundRobin 负载均衡 | 待实现 |
| P2 | 指标持久化到 SQLite | 待实现 |
| P3 | 语义缓存（Redis 或本地向量） | 待实现 |
