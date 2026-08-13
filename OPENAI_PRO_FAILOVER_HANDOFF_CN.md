# GPT Pro 过载切号与 Codex 重连问题交接

## 1. 交接目标

修复 GPT Pro 正价分组在上游返回以下容量错误时不能静默切换账号的问题：

- `server_is_overloaded`
- `slow_down`
- 同类、明确可重试的上游容量错误

期望行为：首个语义输出之前发生容量错误时，不向客户端泄漏当前账号的前导事件或错误事件，由服务端排除故障账号并重新调度。已经输出正文后不得跨账号重放请求，只能返回 Codex 能识别为可重试的流内错误。

本文只描述源码修复和验证要求，不授权本地 Codex 操作服务器、数据库、线上配置或运行进程。构建产物由管理员手动发布。

## 2. 基线信息

- 源码目录：`custom-main`
- 分支：`custom/main`
- 排查时 HEAD：`c6c1139f148587747fdf18dd7a721df8baffddda`
- 已有 HTTP 修复提交：`c33c3208e307c53c82daebc0ba303c3f09b51308`
- 提交说明：`fix(gateway): 流内降载错误恢复 pre-output failover 并对客户端改写为可重试错误码`
- 线上程序和本目录源码不是同一构建：线上 `sub2api` 文件时间为 2026-08-11 06:26，上传源码文件时间为 12:16，线上行为也不包含上述 HTTP 修复。

排查时账号结构如下，账号配置可能已被管理员临时调整，修复不能依赖这些固定 ID：

| 账号 | 类型 | 原始 WS v2 配置 | 作用 |
| --- | --- | --- | --- |
| 28895、28897、28898、28899、28900 | OpenAI OAuth Pro | `enabled=true`, `ctx_pool` | 通常先被调度 |
| 28896（元老院 Pro） | OpenAI API Key | `enabled=false`, `off` | 预期的后备账号之一 |

账号 28896 当时处于可调度状态，并支持问题请求使用的 `gpt-5.6-sol`、`gpt-5.6-luna`、`gpt-5.6-terra`。未切换到它不是模型兼容或调度器候选不足，而是转发层没有向 handler 返回可切号错误。

## 3. 已确认的线上现象

典型请求：

1. Codex 通过 `/responses` 发起流式 `gpt-5.6-sol` 请求。
2. 首选 OAuth Pro 账号返回 `server_is_overloaded`。
3. 线上记录：

   ```text
   openai.forward_failed
   upstream_error_response_already_written=true
   error="upstream response failed: Our servers are currently overloaded..."
   ```

4. HTTP access log 仍为 `200`，因为 SSE 响应头或错误事件已经写出。
5. Codex 将该轮识别为异常流并重新发送请求，界面显示“正在重新连接”。

已关联到同一次客户端重试的样本：第一轮请求在 12:27:36 发出、12:27:38 过载失败；第二轮在 12:27:40 使用相同大小的请求体重新发出。该提示是客户端重试，不是服务端成功切号的 UI 提示。

## 4. 根因

### 4.1 当前源码的 HTTP/SSE 路径已有主体修复

相关文件：

- `backend/internal/service/openai_gateway_response_handling.go`
- `backend/internal/service/openai_gateway_passthrough.go`
- `backend/internal/service/openai_capacity_shed_test.go`

当前源码能够识别真实序列：

```text
response.created
response.in_progress
error(code=server_is_overloaded)
response.failed(code=server_is_overloaded)
```

`response.created`、`response.in_progress` 和可重试 `error` 不应算作语义输出。`response.failed` 到达时，如果没有向客户端输出，应返回 `UpstreamFailoverError`，且 recorder/body 为空。

已有回归测试：

- `TestOpenAIStreamCapacityShedErrorFramePrecedingFailedStillFailsOver`
- `TestOpenAIStreamCapacityShedAfterOutputRewritesCodeForClient`

注意：`openai_gateway_response_handling.go` 当前先执行 `applyOpenAIStreamFailedErrorPassthroughRule`，再执行 `openAIStreamFailedEventShouldFailover`。因此一条匹配 502/503 或 overloaded 文本的通用透传规则仍可能抢先终止切号。见 4.4。

### 4.2 普通 WS v2 转发路径缺少容量错误分类

相关文件：

- `backend/internal/service/openai_ws_forwarder_support.go`
- `backend/internal/service/openai_ws_forwarder_v2.go`
- `backend/internal/service/openai_gateway_forward.go`

`classifyOpenAIWSErrorEventFromRaw` 当前处理 rate limit、`server_error`、WS 协议错误等，但没有直接处理：

```text
server_is_overloaded
slow_down
```

这两个错误最终落入：

```text
reason=event_error
canFallback=false
```

随后 WS 转发器会把缓存的前导帧和错误帧写给客户端，返回普通 error，而不是 `UpstreamFailoverError`。handler 因此无法排除当前账号并选择下一个账号。

`openAIWSPayloadTransientStatus` 已经能把这两个 code 映射为 503，但该能力没有贯通到错误事件分类和账号切换。优先复用共享分类，避免 HTTP、普通 WS 和 ingress WS 各维护一套不同规则。

### 4.3 客户端 WS ingress 路径有同类缺口

相关文件：

- `backend/internal/service/openai_ws_forwarder_ingress.go`
- `backend/internal/handler/openai_gateway_handler.go`

ingress 当前只在尚未输出时把 `rate_limit_exceeded` 转换成 `UpstreamFailoverError`。它没有对 `server_is_overloaded`、`slow_down` 做相同转换，并且会立即向客户端写 `response.created` 等事件，导致后续即使识别出错误也已经不能安全切号。

handler 已有 `handleWSFailover`，能够：

- 报告当前账号调度失败；
- 将账号加入排除集合；
- 记录 `openai.websocket_upstream_failover_switching`；
- 重新进入账号选择循环。

缺失的是 service 层在正确边界返回 `UpstreamFailoverError`。

多轮 WS 请求必须额外考虑 `previous_response_id`：

- 第一轮、未产生语义输出、没有账号绑定状态时可以切号重放。
- 已绑定上一轮 response 的请求不能盲目跨账号重放。
- 复用现有 `previousResponseCanMove`、previous-response recovery 语义，不要为了过载切号破坏多轮会话一致性。

### 4.4 错误透传规则优先级与切号目标冲突

当前 HTTP/SSE 处理顺序中，通用错误透传规则早于容量 failover。若管理员重新开启一条匹配 502/503、`server_is_overloaded` 或 overloaded 文本的规则，该规则会再次直接向客户端写错误。

本需求的推荐规则：

- 首个语义输出前，明确的容量错误和限流错误优先进入服务端 failover。
- 通用透传规则不得抢占这类安全切号事件。
- 不可重试错误，例如内容策略、无效参数、上下文超限，仍按原有透传/错误处理语义执行。
- 如果产品确实需要“强制透传且禁止切号”，应使用一个明确的规则动作表达，不能让通用 503 关键词匹配隐式改变 failover 行为。

本地 Codex 需要为该优先级补测试，不能只依赖线上保持规则关闭。

## 5. 必须保持的行为边界

### 首个语义输出之前

- 缓存 `response.created`、`response.in_progress` 和可重试错误前导事件。
- 容量错误不得向客户端写任何本次账号的数据、响应 ID 或错误事件。
- 返回带原始状态、body、headers 的 `UpstreamFailoverError`。
- 标记为 request-scoped transient，按现有策略决定是否先做有限的同账号重试。
- 同账号重试耗尽后必须进入账号切换，不能写错误后结束。
- handler 排除失败账号，再重新调度；不硬编码“元老院 Pro”或任何账号 ID。

### 已经产生语义输出之后

- 禁止切号和整轮重放，避免重复文本、重复工具调用和重复计费。
- 将 `server_is_overloaded`、`slow_down` 改写为客户端可重试的 `server_error`。
- 保留原始错误 message，保持 Responses 流事件 schema 合法。
- 正常结束/失败边界必须明确，不能只关闭 socket 让客户端猜测。

### 不可重试错误

- `content_policy_violation`、无效参数、上下文窗口超限等不得被误判为容量错误。
- 不应跨账号重试确定性的请求错误。
- `cyber_policy` 记录逻辑与本次容量 failover 分类解耦，不要扩大风控触发范围。

## 6. 建议实现顺序

1. 提取或扩展一个共享的 OpenAI 容量错误分类函数，覆盖 HTTP/SSE 和两条 WS 路径。
2. 修复 `classifyOpenAIWSErrorEventFromRaw`，但不要只把 `canFallback` 改成 `true`；必须检查同账号 WS reconnect 耗尽后是否最终返回 `UpstreamFailoverError` 给 handler。
3. 修复普通 WS v2 的 pre-output staging，容量失败时丢弃本次 attempt 的所有缓存帧。
4. 修复 ingress WS：第一轮安全场景下缓存前导帧并返回 failover；对已绑定多轮状态保持保守策略。
5. 调整 HTTP/SSE 中容量 failover 与通用透传规则的优先级，并补明确测试。
6. 统一 after-output 的错误码改写，确保 Codex 执行退避重试而不是终止会话。
7. 运行定向测试、相关包测试和完整 backend 测试。

## 7. 必须新增的测试

建议至少覆盖以下用例：

1. 普通 WS v2：`created -> in_progress -> error(server_is_overloaded) -> failed`，首账号失败后切到第二账号，客户端没有收到首账号前导帧。
2. 普通 WS v2：同样覆盖 `slow_down`。
3. 普通 WS v2：同账号 reconnect 达到上限后仍进入账号切换，而不是写错误结束。
4. ingress WS 第一轮：过载发生在首 token 前，handler 排除首账号并重新调度。
5. ingress WS 多轮：存在不可迁移 `previous_response_id` 时不得跨账号盲目重放。
6. WS 已输出 token 后发生容量错误：不切号，错误 code 改写为 `server_error`。
7. HTTP/SSE 开启一条匹配 503/overloaded 的通用透传规则：pre-output 容量错误仍按约定进入 failover。
8. HTTP/SSE 与 WS 的 content policy、context window、invalid request 不发生误切号。
9. 保持现有限流测试通过：`rate_limit_exceeded` 行为不能回退。
10. 切号成功后只记录最终成功账号的正常结果，失败账号记录一次调度失败，不产生重复 usage。

现有但覆盖不足的测试位置：

- `backend/internal/handler/openai_gateway_handler_test.go` 中只有 WS `rate_limit_exceeded` 切号案例。
- `backend/internal/service/openai_ws_ratelimit_signal_test.go` 只覆盖 rate limit 信号。
- `backend/internal/service/openai_capacity_shed_test.go` 主要覆盖 HTTP/SSE。

## 8. 验证命令

在 `custom-main/backend` 执行：

```bash
go test ./internal/service -run 'TestOpenAIStreamCapacityShed|TestOpenAIWS|TestClassifyOpenAIWSError' -count=1
go test ./internal/handler -run 'TestOpenAIResponsesWebSocket|TestOpenAIFirstOutput' -count=1
go test ./internal/service ./internal/handler -count=1
go test ./... -count=1
```

建议再运行 race 测试覆盖 WS 连接池状态：

```bash
go test -race ./internal/service ./internal/handler -count=1
```

构建 Linux amd64 后端的仓库命令：

```bash
make build-linux-amd64
```

该命令定义在 `backend/Makefile`，输出为 `backend/bin/server-linux-amd64`。如果线上使用 embed 前端版本，应按现有生产发布流程构建，不能直接假设该文件可替换当前程序。

## 9. 验收标准

发布后使用可控测试账号制造一次首输出前过载，必须观察到：

```text
openai.upstream_failover_switching
```

或 WS ingress 对应的：

```text
openai.websocket_upstream_failover_switching
```

同时满足：

- 日志包含失败账号 ID、`upstream_status=503` 和递增的 `switch_count`。
- 同一客户端请求最终由另一个兼容账号完成。
- 客户端没有收到首账号的 response ID、`server_is_overloaded` 或 `slow_down`。
- Codex 不显示“正在重新连接”。
- 不再出现该场景对应的 `upstream_error_response_already_written=true`。
- 已输出正文后的人工故障测试不发生跨账号重放。
- 无重复工具调用、重复文本或重复计费记录。

## 10. 手动发布注意事项

1. 本地记录源码 HEAD、构建时间、Go 版本和产物 SHA-256。
2. 先在非线上端口或测试实例验证健康检查和上述过载案例。
3. 发布前备份当前二进制，并记录可立即回滚的路径。
4. 使用与现网一致的配置和工作目录，不要把源码目录内的 `.env` 或本地配置覆盖到服务器。
5. 在维护窗口进行原子替换和受控重启，确认只有目标进程被操作。
6. 发布后先验证版本/校验和，再观察切号日志、5xx、Codex 重连率和首 token 延迟。
7. 若出现响应重复、计费异常或 WS 重连风暴，立即回滚二进制；数据库配置无需随代码回滚。

## 11. 本次排查未做的操作

- 未修改业务源码。
- 未修改数据库或账号配置。
- 未编译或替换线上二进制。
- 未停止、重启或探测性接管主应用端口。
