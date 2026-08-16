# Sub2API 自定义功能保护清单

> 用途：下次从上游合并代码到 `custom/main` 时，用这份清单逐项核对，避免当前项目自定义功能再次被覆盖或丢失。
>
> 合并原则：保留上游 `main` 新功能，同时保留本项目自定义功能；合并完成后不要只看是否能编译，还要按本文的“合并后必测清单”逐项验证。

## 1. Git / 合并约束

- 当前自定义分支：`custom/main`
- 上游/项目远端应指向：`https://github.com/liaoruikang/sub2api`
- 合并上游时要求：
  - 不推送远端，除非明确要求。
  - 不使用 destructive git 操作丢弃本地自定义变更。
  - 冲突解决时不能简单选择 `theirs` 覆盖本地功能。
  - 合并后必须检查本文列出的功能链路。

## 2. 必须保留的自定义功能概览

### 2.1 在线生图菜单与系统设置开关

功能要求：

- 用户端存在“在线生图”菜单入口。
- 管理员可在系统设置中启用/关闭“在线生图”。
- 默认开启。
- 关闭后：
  - 用户侧边栏隐藏 `/images` 菜单。
  - 用户直接访问 `/images` 会被路由守卫拦截并重定向。
- 该开关必须通过后端设置持久化，保存后刷新页面仍然生效。

关键 setting key：

- `image_playground_enabled`

关键文件：

- 后端：
  - `backend/internal/service/domain_constants.go`
  - `backend/internal/service/setting_parse.go`
  - `backend/internal/service/settings_view.go`
  - `backend/internal/service/setting_update.go`
  - `backend/internal/service/setting_public.go`
  - `backend/internal/handler/dto/settings.go`
  - `backend/internal/handler/admin/setting_handler.go`
  - `backend/internal/handler/admin/setting_handler_update.go`
  - `backend/internal/handler/admin/setting_handler_audit.go`
  - `backend/internal/handler/setting_handler.go`
- 前端：
  - `frontend/src/api/admin/settings.ts`
  - `frontend/src/types/index.ts`
  - `frontend/src/stores/app.ts`
  - `frontend/src/utils/featureFlags.ts`
  - `frontend/src/components/layout/AppSidebar.vue`
  - `frontend/src/router/meta.d.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/views/admin/SettingsView.vue`
  - `frontend/src/i18n/locales/zh/admin/settings.ts`
  - `frontend/src/i18n/locales/en/admin/settings.ts`

容易丢失点：

- `parseSettings` 必须把 `image_playground_enabled` 解析进管理端 `SystemSettings`，否则保存后回显会失效。
- `GetPublicSettings` 和 `PublicSettingsInjectionPayload` 都必须包含该字段，否则菜单首屏/刷新后状态可能不一致。
- 前端 `FeatureFlags.imagePlayground` 必须是 `opt-out`，默认开启，只有后端明确返回 `false` 才隐藏。

### 2.2 视频任务菜单与系统设置开关

功能要求：

- 用户端存在“视频任务”菜单入口。
- 管理员可在系统设置中启用/关闭“视频任务”。
- 默认开启。
- 关闭后：
  - 用户侧边栏隐藏 `/videos` 菜单。
  - 用户直接访问 `/videos` 会被路由守卫拦截并重定向。
- 该开关必须通过后端设置持久化，保存后刷新页面仍然生效。

关键 setting key：

- `video_jobs_enabled`

关键文件同“在线生图菜单与系统设置开关”。

容易丢失点：

- `parseSettings` 必须把 `video_jobs_enabled` 解析进管理端 `SystemSettings`。
- public settings 接口和 HTML 注入 payload 都必须包含该字段。
- 前端 `FeatureFlags.videoJobs` 必须是 `opt-out`。

### 2.3 在线生图高级模式

功能要求：

- 在线生图页面支持“高级模式”。
- 高级模式展示并允许编辑：
  - 接口路径。
  - JSON 请求参数。
- 支持路径：
  - `/v1/images/generations`
  - `/v1/images/edits`
- 左侧普通参数面板与高级 JSON 参数需要同步：
  - 普通表单变化时，高级 JSON 自动更新。
  - 高级 JSON 合法修改后，同步回左侧表单。
  - 高级模式下实际生成时必须使用编辑后的 JSON，而不是只展示不提交。
- 非法输入必须提示并阻止生成：
  - 非法路径。
  - 非法 JSON。
  - JSON 不是 object。
  - 缺少或非法 `api_key_id`。
  - `model` 为空。
  - `prompt` 为空。
  - `n` 不是 1-10 的整数。
  - `output_compression` 不是 0-100 的整数。
  - `/v1/images/edits` 没有参考图。
  - `/v1/images/generations` 携带参考图。
- 高级模式中可以查看最后一次请求和响应/错误详情。

关键文件：

- `frontend/src/views/user/ImagePlaygroundView.vue`
- `frontend/src/api/imagePlayground.ts`
- `frontend/src/i18n/locales/zh/misc.ts`
- `frontend/src/i18n/locales/en/misc.ts`

关键函数/状态：

- `imageGeneratePayload`
- `generateImageAdvanced`
- `extractImageGenerationErrorMessage`
- `advancedEnabled`
- `advancedRequestPath`
- `advancedRequestBodyText`
- `advancedRequestError`
- `handleAdvancedRequestPathInput`
- `handleAdvancedRequestBodyInput`
- `refreshAdvancedValidation`
- `applyAdvancedRequestBody`
- `advancedBodyForBatch`
- `lastAdvancedRequest`
- `lastAdvancedResponse`
- `lastAdvancedError`

容易丢失点：

- `imageGeneratePayload` 需要导出供页面构造高级 JSON。
- `generateImageAdvanced` 必须保留；高级模式非流式生成要把编辑后的 JSON body 真实提交到 `/user/images/generations`。
- 带参考图时，高级 JSON 需要通过 multipart 提交，并追加 `image` 文件，后端会转到 `/v1/images/edits`。
- 高级模式校验错误必须参与 `generateDisabled`。
- 高级路径和 JSON 输入必须显式触发 `handleAdvancedRequestPathInput` / `handleAdvancedRequestBodyInput`，不能只依赖 watcher，否则可能不稳定同步。
- 合法高级 JSON 必须通过 `refreshAdvancedValidation` / `applyAdvancedRequestBody` 立即回写普通表单。
- 高级 JSON 回写表单时不能被 API Key 默认模型 watcher 覆盖 `model`。
- 分批生成时必须通过 `advancedBodyForBatch` 改写每批 `n`，不能丢失原有分批逻辑。
- 成功响应要写入 `lastAdvancedResponse`。
- 失败错误要写入 `lastAdvancedError`。

### 2.4 在线生图失败展示上游错误消息

功能要求：

- 在线生图生成失败时，不只显示通用“图片生成失败”。
- 如果上游返回错误原因，需要直接显示上游错误消息。
- 高级模式中可以查看错误 payload。

关键文件：

- `frontend/src/api/imagePlayground.ts`
- `frontend/src/views/user/ImagePlaygroundView.vue`

关键函数：

- `extractImageGenerationErrorMessage(error)`

错误提取优先级应包含：

- `error.response?.data?.error?.message`
- `error.response?.data?.message`
- `error.response?.data?.detail`
- `error.error?.message`
- `error.message`

### 2.5 在线生图长耗时请求不应被前端 30 秒超时中断

功能要求：

- 图片生成可能超过 30 秒。
- 在线生图请求不能使用普通 axios 30 秒超时。

关键文件：

- `frontend/src/api/imagePlayground.ts`

关键点：

- `IMAGE_GENERATION_TIMEOUT_MS = 0`
- `generateImage` 中 axios 请求应使用该 timeout。

### 2.6 流式生图进度与计费相关逻辑

功能要求：

- 流式生图支持实时进度展示。
- 流式与非流式生成都应保留图片结果和计费元数据。
- 计费元数据 `_sub2api_image_playground` 不能在合并中丢失。

关键文件：

- `frontend/src/api/imagePlayground.ts`
- `frontend/src/views/user/ImagePlaygroundView.vue`
- `backend/internal/server/routes/gateway_key_billing_test.go`
- `backend/internal/server/routes/user_test.go`

关键类型：

- `ImagePlaygroundCostMetadata`
- `ImagePlaygroundGenerateResponse`

### 2.7 OpenAI 账号调度 / 最高调度模式相关自定义逻辑

功能要求：

- 保留 OpenAI 账号调度相关自定义逻辑。
- 保留最高调度模式、权重、TTFT、队列、错误率、quota headroom、上游成本等配置。
- 不要在合并时被上游调度逻辑覆盖。

关键文件：

- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- `backend/internal/service/openai_account_scheduler_test.go`
- `backend/internal/service/openai_account_scheduler_upstream_cost_test.go`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/api/admin/settings.ts`
- `frontend/src/i18n/locales/zh/admin/accounts.ts`
- `frontend/src/i18n/locales/en/admin/accounts.ts`

关键 setting / 字段前缀：

- `openai_advanced_scheduler_*`
- `openai_low_upstream_rate_priority_enabled`
- `openai_oauth_scheduling_rate_multiplier`

### 用户标签驱动专属分组授权

功能要求：

- 管理员可以创建、重命名、删除用户标签，并给用户分配一个或多个标签。
- 标准专属 Group 可以绑定一个或多个用户标签；用户拥有任意一个绑定标签时自动获得该 Group。
- 手工授权和标签派生授权取并集；标签派生权限不能写入 `user_allowed_groups`。
- 用户分组配置中，标签命中的专属 Group 显示“标签名专属”，并且不可通过 checkbox 撤销。
- 标签专属分组的展示文案必须直接拼接为“标签名专属”（多个标签用顿号分隔），不得渲染可见的 `+` 号；用户分组配置、API Key 分组列表和公共分组组件必须保持一致。
- 管理员修改用户 API Key 的分组后，更新响应中的 Group 必须包含与列表接口一致的 `tags` / `tag_ids`，保存后的本地列表不得先退化为“专属”再依赖刷新恢复“标签名专属”。
- 删除标签、移除用户标签或移除 Group 标签后，派生授权自动撤销，但不能删除仍存在的手工授权。
- subscription Group 不能通过标签绕过 active subscription 检查。

关键文件：

- 后端：`backend/migrations/221_add_user_tag_group_authorization.sql`、`backend/internal/service/user_tag.go`、`backend/internal/repository/user_tag_repo.go`、`backend/internal/service/api_key_service.go`、`backend/internal/service/api_key_auth_cache_impl.go`、`backend/internal/service/admin_group.go`、`backend/internal/service/admin_service_apikey_test.go`
- 前端：`frontend/src/components/admin/user/UserTagManagementModal.vue`、`frontend/src/components/admin/user/UserTagMultiSelect.vue`、`frontend/src/components/admin/user/UserAllowedGroupsModal.vue`、`frontend/src/components/admin/user/UserApiKeysModal.vue`、`frontend/src/components/admin/user/__tests__/UserAllowedGroupsModal.spec.ts`、`frontend/src/components/common/GroupAuthorizationBadge.vue`、`frontend/src/views/admin/GroupsView.vue`、`frontend/src/views/admin/UsersView.vue`

容易丢失点：

- 标签授权查询必须过滤已删除、非 active、非 standard 或非 exclusive Group，并对多标签 OR 查询去重。
- auth snapshot 需要包含标签派生 Group，缓存版本变化后旧快照必须回源。
- API Key 主 Group、有序次级 Group、Model Plaza 和 Available Channels 必须使用有效授权并集。
- 管理员 API Key 分组更新链路必须在构造响应前补齐 Group 标签关系，不能直接返回只含 `is_exclusive` 的校验对象。

## 2.8 账号池模式扩展到 OAuth 与批量编辑

功能要求：

- OAuth 类型账号复用 API Key 已有池模式字段。
- OAuth/API Key 账号开启池模式后，遇到配置的状态码可按同账号重试次数重试。
- `pool_mode_retry_status_codes` 需要保留三种语义：
  - `nil`：未配置，回退默认 `[401, 403, 429]`。
  - 空数组：管理员显式置空，不按状态码触发同账号重试。
  - 非空数组：使用管理员配置的状态码列表。
- 批量修改账号弹窗必须支持批量开启/关闭池模式，并可批量设置同账号重试次数和重试状态码。

关键文件：

- 后端：
  - `backend/internal/service/account.go`
  - `backend/internal/service/ratelimit_service.go`
  - `backend/internal/service/ratelimit_service_401_test.go`
  - `backend/internal/service/account_pool_retry_status_codes_test.go`
- 前端：
  - `frontend/src/components/account/BulkEditAccountModal.vue`
  - `frontend/src/components/account/CreateAccountModal.vue`
  - `frontend/src/components/account/EditAccountModal.vue`
  - `frontend/src/i18n/locales/zh/admin/accounts.ts`
  - `frontend/src/i18n/locales/en/admin/accounts.ts`

关键字段：

- `pool_mode`
- `pool_mode_retry_count`
- `pool_mode_retry_status_codes`

容易丢失点：

- `Account.IsPoolMode()` 必须允许 `AccountTypeOAuth`，不能只允许 API Key/Bedrock。
- OAuth 账号 401 不应直接临时不可调度；开启池模式且状态码命中配置时应进入池模式重试。
- 批量编辑关闭池模式时需要写入 `pool_mode=false`、`pool_mode_retry_count=0`、`pool_mode_retry_status_codes=[]`。

### 2.9 最高调度轮转

功能要求：

- 账号管理页右上角存在“最高调度轮转”入口，位于“更多操作”附近。
- 可配置：
  - 是否启用轮转。
  - 分组范围；不选择具体分组表示全部分组和未分组账号。
  - 账号类型范围：OAuth、API Key，可多选。
  - 轮转数量 `n`。
- 开启轮转后，在配置范围内自动维持指定数量的最高调度账号。
- 候选账号必须满足：状态正常、调度可用，并且未处于临时不可调度、过载、限流重置等待、过期自动暂停等不可调度状态。
- 开启轮转后禁止手动开关范围内账号的最高调度模式。
- 开启轮转时按差量校准最高调度账号，不要每次都把范围内所有账号全部关闭后重新轮转。
- 关闭轮转时必须清空范围内所有最高调度状态。
- 当前符合条件的最高调度账号数量不等于 `n` 时，账号状态/调度/限流/过载/分组等被动事件需要触发差量补位或关闭多余账号；不要用定时器轮询。
- 最高调度列支持点击切换手动最高调度；调度开关旁不应再有额外最高调度按钮。
- 列表支持按最高调度排序，最高调度账号排最前。
- 弹窗保存成功后需要关闭弹窗。
- 顶部入口按钮启用态只改变文字颜色，不改变原按钮背景/边框样式。

关键 setting key：

- `highest_scheduling_rotation_config`

关键 extra key：

- `highest_scheduling_mode`

关键文件：

- 后端：
  - `backend/internal/service/account_highest_scheduling_rotation.go`
  - `backend/internal/service/admin_account.go`
  - `backend/internal/service/admin_service.go`
  - `backend/internal/service/domain_constants.go`
  - `backend/internal/service/wire.go`
  - `backend/cmd/server/wire_gen.go`
  - `backend/internal/repository/account_repo.go`
  - `backend/internal/server/routes/admin.go`
  - `backend/internal/handler/admin/account_handler.go`
  - `backend/internal/handler/admin/admin_service_stub_test.go`
- 前端：
  - `frontend/src/views/admin/AccountsView.vue`
  - `frontend/src/api/admin/accounts.ts`
  - `frontend/src/i18n/locales/zh/admin/accounts.ts`
  - `frontend/src/i18n/locales/en/admin/accounts.ts`

关键结构/函数：

- `HighestSchedulingRotationConfig`
- `HighestSchedulingRotationState`
- `HighestSchedulingRotationReconciler`
- `InjectHighestSchedulingRotationReconciler`
- `ShouldReconcileHighestSchedulingRotation`
- `ReconcileHighestSchedulingRotation`
- `IsHighestSchedulingRotationManagedAccount`
- `highestSchedulingRotationCandidate`
- `accountUpdateAffectsHighestSchedulingRotation`
- `reconcileHighestSchedulingRotationForAccounts`
- `highestSchedulingRotationEnabled`
- `highestSchedulingRotationForm`
- `saveHighestSchedulingRotation`
- `handleToggleHighestScheduling`
- `isHighestSchedulingRotationManagedAccount`

容易丢失点：

- 生产中运行时账号仓库和管理端账号仓库是两个实例；同一个轮转 reconciler 必须同时注入两边，否则管理端关闭调度/改状态不会触发补位。
- `NewHighestSchedulingRotationReconciler(accountRepository, settingService)` 仍以运行时账号仓库作为轮转 core 的主仓库；`ProvideAdminService(..., adminAccountRepository, ..., highestSchedulingRotationReconciler)` 负责给管理端仓库注入同一个 reconciler。
- 仓库层被动事件不能只在“变更账号本身是最高调度账号”时触发；需要在每次相关账号状态/调度变化后检查当前符合条件的最高调度账号数量是否低于 `n`，不满才执行 reconcile。
- `ShouldReconcileHighestSchedulingRotation` 统计的是“仍符合候选条件的最高调度账号”，不能把调度已关闭但仍带 highest extra 的账号算作满足数量。
- 服务层不要再重复无条件触发 reconcile，避免一次状态变化导致多次轮转。
- 前端轮转开启后，范围内账号最高调度列按钮需要禁用手动切换。
- “最高调度轮转”短标题不要退回“最高轮转”。

### 2.10 后端 Windows/Linux 编译兼容修复

功能要求：

- 保留后端构建脚本里针对 Go compiler nilcheckelim ICE 的规避参数。

关键文件：

- `backend/build-linux-amd64.ps1`

关键点：

- 构建参数中需要保留：
  - `-gcflags=all=-d=ssa/nilcheckelim/off`

### 2.11 分组长上下文定价与账号开关三态保护

功能要求：

- 分组 `long_context_pricing_enabled` 控制是否应用模型长上下文阶梯价格。
- OpenAI 账号自身的长上下文开关是额外 gate；Grok 等没有该账号开关的平台必须传 `nil`，仅由分组开关决定，不能硬编码为 `false` 否决官方阶梯价格。
- 预估计费与正式用量记账必须共用 `openAILongContextBillingGate`，避免展示金额与实际扣费口径不一致。

关键文件：

- `backend/internal/service/openai_gateway_usage.go`
- `backend/internal/service/model_pricing_resolver.go`
- `backend/internal/service/openai_gateway_record_usage_test.go`

### 2.12 API Key 多分组绑定与 Gateway Failover

功能要求：

- 一个 API Key 可以绑定多个 Group。
- `group_ids` 的顺序由管理员配置决定，第一次请求固定从 `group_ids[0]` 开始；读取关联关系时必须按 `api_key_groups.position` 排序，不能按 Group ID 或数据库默认顺序排序。
- 保留旧客户端的 `group_id` 兼容语义：未传 `group_ids` 时不修改已有绑定，`group_id > 0` 绑定单个 Group，`group_id == 0` 解绑全部；同时传入时必须保证 `group_ids[0] == group_id`。
- Group 绑定必须校验用户权限、platform 一致性和 subscription type 一致性；重复 Group ID 去重并保留首次出现的位置。
- 当前 Group 必须先完成同账号重试、账号级 failover 和账号池耗尽处理，只有当前 Group 的账号池真正耗尽或 Group 级可恢复准入/计费错误时，才按配置顺序切换到下一个 Group。
- 账号切换和 Group 切换共用同一个请求级切换预算；同账号重试和利润 veto 不消费预算。每个 Group 的 `FailedAccountIDs`、同账号重试计数和利润 veto 状态必须独立。
- 每次切换 Group 都要重新执行完整准入：subscription、API Key/Group context、composite target、channel mapping、Claude Code only、Group 用户并发槽和 billing eligibility。
- sticky session、usage、billing 和异步 usage 闭包必须归属当前成功 Group，不能继续使用初始 Group 或可变 API Key 快照。
- 流式请求一旦写出有效语义内容，禁止账号或 Group failover；仅写出 compact SSE keepalive 时仍允许 failover。
- invalid-request fallback 是独立且最多一次的 fallback attempt，不得递归 fallback、错误推进普通 Group 顺序；启动 fallback 前也必须遵守请求级预算。
- 保留现有 OAuth/API Key pool mode、最高调度模式、最高调度轮转、OpenAI scheduler、利润控制、Bedrock、Responses compact/SSE 和既有协议兼容逻辑。

关键文件：

- 数据模型与迁移：
  - `backend/ent/schema/api_key.go`
  - `backend/ent/schema/api_key_group.go`
  - `backend/ent/schema/group.go`
  - `backend/migrations/194_add_api_key_groups.sql`
- API Key service/repository/DTO：
  - `backend/internal/service/api_key.go`
  - `backend/internal/service/api_key_service.go`
  - `backend/internal/repository/api_key_repo.go`
  - `backend/internal/handler/dto/types.go`
  - `backend/internal/handler/dto/mappers.go`
- Gateway failover：
  - `backend/internal/handler/group_attempt.go`
  - `backend/internal/handler/failover_loop.go`
  - `backend/internal/handler/ordered_group_failover.go`
  - `backend/internal/handler/gateway_handler.go`
  - `backend/internal/handler/gateway_handler_chat_completions.go`
  - `backend/internal/handler/gateway_handler_responses.go`
  - `backend/internal/service/gateway_scheduling.go`

关键结构/函数：

- `api_key_groups(api_key_id, group_id, position)`
- `CreateAPIKeyRequest.GroupIDs`
- `UpdateAPIKeyRequest.GroupIDs`
- `resolveOrderedAPIKeyGroups`
- `applyAPIKeyGroupAttemptWithBase`
- `OrderedGroupFailoverState`
- `RequestFailoverBudget`
- `NewFailoverStateWithBudget`
- `NewOrderedGroupFailoverStateWithBudget`
- `isRetryableGroupBillingError`

容易丢失点：

- 不要用 `group_id` 覆盖或替代完整的有序 `group_ids` 关联；`group_id` 只能作为第一个 Group 的 legacy mirror。
- 不要在 Group 切换时复用旧 subscription、channel mapping、并发槽、sticky key 或 usage 闭包；这些状态必须按当前 Group 重新装配。
- 不要把 User/API Key 限流、余额不足、billing service unavailable、客户端取消或确定性请求错误错误地当作 Group 可恢复错误。
- 不要在已经写出有效 SSE/JSON 语义内容后切换 Group，也不要把 compact keepalive 当作有效响应内容。
- 共享预算不能污染 Group-local 失败集合；fallback 不得循环回普通 Group failover。

### 2.13 分组变更监测公告

功能要求：

- 全局时区解析必须保持 `TZ`（非空）> `TIMEZONE`（非空）> 配置文件 > `Asia/Shanghai` 默认值的优先级。配置加载需要保留 `AllowEmptyEnv(true)` 对显式空密钥的兼容，但空 `TIMEZONE` 不能覆盖配置文件或默认时区，否则分组自然日统计、监测周期和公告时间会发生偏移。

- 公告设置中可以启用/停用分组变更监测；监测内容支持“价格变化”和“状态变化”多选，至少选择一项，历史配置缺少该字段时默认只监测价格变化。
- 监测范围支持全部分组或多个指定分组，监测周期单位为秒。
- 公告状态、通知方式和展示条件复用公告模型语义；监测有效期通过 `duration_days` 配置。
- 状态变化包含分组新增、复制新增、启用、停用和删除；事件必须在对应管理操作成功后记录，失败的创建、复制、更新或删除不能产生公告，幂等恢复已有复制分组时不得重复记录。
- 每个监测周期内所有已选类型的变化必须合并为一条 Markdown 公告；价格与状态同时变化时仍只创建一条逻辑公告，并分别使用价格表格和状态表格直观展示。
- 公告标题根据实际明细使用“分组价格调整通知”“分组状态调整通知”或“分组价格与状态调整通知”；用户端过滤无权限明细后必须同步重算标题，不能通过标题泄露无权限分组的变更类型。
- 每条变化必须保存为带 `group_id` 和 `change_type` 的结构化明细，状态事件还需保存变更前后状态及发布时的权限信息；用户端不得依赖 Markdown 反解析权限。
- 公告内容必须逐项展示分组名称、涨价/降价方向和旧倍率到新倍率（如 `0.07 -> 0.08`），并突出方向。
- 用户查看价格公告时必须按当前分组权限动态过滤明细：公开标准分组所有用户可见；标准专属分组取手工授权与标签派生授权并集；订阅分组必须有有效订阅。用户后来获得权限时，可以看到仍在有效期内的历史公告对应明细；权限撤销后立即隐藏。
- 停用或删除后的专属分组不能因不再出现在 active Group 列表而丢失公告受众；删除事件必须在删除前快照手工授权、有效订阅和标签关系，标签受众仍按用户当前标签动态判断。
- 价格公告已读状态必须按 `(announcement_id, user_id, group_id)` 记录。用户读过公开明细后又获得专属分组权限时，该逻辑公告必须重新显示为未读；不能只使用公告级 `announcement_reads`。
- 服务启动后后台轮询；首次检查只建立基线，保存配置、停用或修改范围后必须重新建立基线，不能把期间变化误报。
- 管理端必须显示距离下一次实际监测的实时倒计时；时间基准由后端返回的 `next_check_at` 与 `server_time` 决定，配置保存后应立即重排周期，不能仅按页面打开时间自行估算。

关键 setting key：

- `group_price_monitor_config`

关键文件：

- 后端：`backend/internal/service/group_price_monitor.go`、`backend/internal/service/admin_group.go`、`backend/internal/handler/admin/group_price_monitor_handler.go`、`backend/internal/server/routes/admin.go`、`backend/internal/service/wire.go`、`backend/cmd/server/wire.go`
- 公告权限与持久化：`backend/internal/service/announcement_service.go`、`backend/internal/repository/announcement_repo.go`、`backend/internal/repository/announcement_read_repo.go`、`backend/internal/repository/group_repo.go`、`backend/migrations/225_extend_group_price_monitor_events.sql`、`backend/ent/schema/announcement_group_price_change.go`、`backend/ent/schema/announcement_group_price_read.go`
- 前端：`frontend/src/views/admin/AnnouncementsView.vue`、`frontend/src/api/admin/announcements.ts`、`frontend/src/types/index.ts`

界面约束：

- 分组变更监测配置必须通过公告列表工具栏入口打开独立弹窗，不能长期嵌在公告列表上方占用页面空间。
- 监测内容复选项必须延续现有表单、边框、选中态及深色模式风格；前后端都必须拒绝空选择。
- 指定分组选择必须复用账号编辑弹窗使用的公共 `frontend/src/components/common/GroupSelector.vue`，不能在公告页复制一套分组选择器；“全部分组”作为独立总开关。
- 启用配置保存后必须立即建立当前倍率基线；服务启动读取已启用配置时也必须建立基线，不能吞掉第一次后续实际价格变化。
- 系统自动价格监测公告的 `created_by` 必须为空，管理员手工公告由服务写入管理员 ID；管理端公告列表必须区分“系统发布”和“管理员发布”。
- 用户公告 DTO 必须透传 `created_by`；公告铃列表、公告详情和弹窗都必须依据该字段标识“系统发布”或“管理员发布”，不能根据标题猜测来源。
- 价格变化公告内容必须使用 Markdown 表格，展示变动数量、分组、调整类型、调整前和调整后；方向（▲ 涨价/▼ 降价）及新倍率必须突出，分组名称保持纯文本，不能包裹 `**`。状态变化使用独立 Markdown 表格展示新增/启用/停用/删除以及变更前后状态。
- 分组价格监测启用开关必须通过状态绑定控制轨道颜色和滑块位移，并保留平滑过渡动画；不能依赖嵌套元素上的 `peer-checked` 选择器。
- 已启用时，公告列表工具栏和价格监测设置弹窗都要显示逐秒更新的下一次监测倒计时；到期后重新读取后端运行时计划，停用时不显示虚假的下一次时间。
- 分组价格监测不再让管理员填写开始/结束时间，前端只提交有效天数；监测配置本身不得固化 `starts_at` / `ends_at`，每条自动公告发布时以后端当前时间作为开始时间，并按 `duration_days` 日历天计算该公告的结束时间。
- 管理员修改分组倍率时必须记录每一次实际变更，监测周期结束时合并发布；不能只依赖固定间隔轮询，否则同一周期内的快速变更（如 `0.13 -> 0.12 -> 0.13`）会被最终值抵消。
- 用户公告接口必须根据当前权限重新渲染价格公告正文，禁止把包含无权限分组的完整 Markdown 返回给客户端；标记已读接口也必须重新鉴权，权限查询失败时不得退化为全部可见。
- 用户打开公告铃铛或页面从后台恢复时必须强制刷新公告，失去权限的公告应从列表和弹窗队列移除；同一公告因后来新增分组权限重新变为未读时，允许再次弹窗。

回归检查：

- 首次检查不发公告；同一周期多个分组变化、以及价格与状态混合变化均合并为一条公告。
- 历史配置默认只选择价格变化；前后端拒绝未选择任何监测内容；只选价格或只选状态时不得发布另一类型事件。
- 分组创建、启用、停用和删除均产生正确的状态明细；失败操作不产生明细，轮询基线不得把状态切换重复识别为价格变化。
- 涨价和降价均显示方向、分组名称及精确倍率变化。
- 混合公开、手工专属、标签专属和订阅分组的同周期公告，对每个用户只显示其当前有权访问的明细。
- 先读公开明细再获得专属或标签权限时，历史公告重新未读；撤销权限后对应明细立即消失，重新获得权限时保留已有分组级已读记录。
- 后台服务清理时停止监测协程；不停止或重启主服务。
- 启用状态下倒计时从后端实际计划递减，刷新页面或客户端时钟有偏差时仍保持准确；修改周期后立即按新周期重排。
- 系统价格公告和管理员手工公告在管理端及用户端均显示正确的发布来源。

### 2.14 Seedance API Key 原生视频接入

功能要求：

- `seedance` 是独立的原生平台，仅支持 API Key 账号；默认上游地址为 `https://model.service-inference.ai`，账号凭证必须使用 Bearer API Key。
- Seedance 账号创建和编辑时必须支持从上游同步模型；模型列表使用 NewAPI/OpenAI 兼容的 `GET {base_url}/v1/models`、Bearer API Key，并复用账号代理、TLS 指纹和自定义请求头。
- 必须保留文档中的三类素材接口（通用资产组、SD 资产、Doubao SD 资产）以及视频生成、任务查询/列表、用量查询、最后一帧文件接口；不能把 Seedance 当作 Anthropic/OpenAI 复合渠道处理。
- 视频模型必须使用后端维护的官方白名单，并按账号渠道校验：通用 Dreamina、`-hc` 高并发渠道和 `doubao-seedance-*` 渠道不能互相越权。
- 资产与任务必须绑定用户、API Key、分组和实际上游账号；读取或复用 `asset://` 引用时必须重新校验归属，不能因共享账号泄露其他用户资源。
- 视频请求必须经过现有并发、RPM、额度和提示词安全审计链路；只有任务完成后通过原子去重记录用量，不能对轮询重复计费。
- 账号选择必须使用现有负载感知调度；仅在明确的 429/5xx 上游响应时切换候选账号，不能对不确定的 POST 传输错误自动重放。
- 分组需配置 480p、720p、1080p 三档视频单价后才允许创建/更新 Seedance 分组；Seedance 不得加入 Anthropic/OpenAI 的复合平台匹配集合。

关键文件：

- 后端：`backend/internal/service/seedance.go`、`backend/internal/service/seedance_service.go`、`backend/internal/service/upstream_models.go`、`backend/internal/service/upstream_models_test.go`、`backend/internal/service/account_test_service.go`、`backend/internal/service/account_test_service_seedance_test.go`、`backend/internal/handler/seedance_handler.go`、`backend/internal/repository/seedance_repo.go`、`backend/migrations/223_seedance_media.sql`、`backend/migrations/224_seedance_quota_platform.sql`
- 调度与分组：`backend/internal/service/scheduler_snapshot_service.go`、`backend/internal/service/admin_group.go`、`backend/internal/service/account.go`
- 前端：`frontend/src/components/account/CreateAccountModal.vue`、`frontend/src/components/account/EditAccountModal.vue`、`frontend/src/components/account/ModelWhitelistSelector.vue`、`frontend/src/components/account/__tests__/ModelWhitelistSelector.spec.ts`、`frontend/src/components/admin/account/AccountTestModal.vue`、`frontend/src/api/admin/accounts.ts`、`frontend/src/composables/useModelWhitelist.ts`、`frontend/src/types/index.ts`
- 视频预览 CSP：`backend/internal/config/config.go`、`backend/internal/server/middleware/security_headers.go`、`backend/internal/server/middleware/security_headers_test.go`、`deploy/config.example.yaml`

回归检查：

- Seedance 账号创建/编辑校验 API Key 和合法 HTTPS Base URL，管理端模型列表及账号测试消费统一模型对象。
- Seedance 创建页和编辑页均显示“同步上游支持模型”；同步结果以账号上游 `/v1/models` 的实际返回为准，去重后加入白名单，失败时不得回显上游响应正文或密钥。
- Seedance 分组在调度快照中拥有独立平台桶，原生 `/v1` 路由可完成素材上传、视频生成和任务轮询；其他平台请求不应匹配 Seedance 账号。
- 同一 API Key 只能读取自己的资源和任务；同一任务多次轮询只产生一次用量记录；发布阶段可按既定流程备份并替换根目录可执行文件，但不得主动停止或重启线上服务。
- Seedance 完成任务的使用记录必须透传 `video_count`、`video_resolution` 和 `video_duration_seconds`，并将任务创建到完成的耗时写入 `duration_ms`；前端视频行应显示视频图标、数量、分辨率和输出时长，不得把视频误显示成 `0/0` token。分组列表的 `seedance` 平台名称必须在中英文 i18n 中可见。
- Seedance 完成响应中的 `task.usage` 是 token 用量的首选来源（包括 `total_tokens` / `completion_tokens`）；只有完成响应没有 token 字段时，才允许回退到兼容用量接口，不能因该接口不存在而丢弃上游已返回的 token 数。
- 用户视频任务控制台必须在认证后的统一接口中同时汇总 Grok job 和 Seedance task；Seedance 刷新任务前必须重新校验用户与 API Key 归属。
- Seedance 账号连接测试必须真实调用 `/v1/video/generate` 并轮询任务，随后通过管理端 SSE 暴露视频预览；仅调用任务列表探针不能视为视频能力测试。实际账号测试弹窗必须提供可编辑的视频提示词和完整默认示例，提交管理员修改后的提示词；任务轮询上限不得低于 5 分钟，创建任务响应体必须在进入长轮询前关闭。
- 管理端账号测试返回的 Seedance/Grok 视频可能是签名 HTTPS URL、`data:` URL 或 `blob:` URL；运行时 CSP 必须保留 `media-src 'self' data: blob: https:`，包括对旧自定义 CSP 的兼容增强，否则浏览器会在加载视频前直接拦截预览。

### 2.15 OpenAI OAuth 账号 SessionID 控制

功能要求：

- 仅 OpenAI OAuth 母账号可配置 SessionID 控制；创建、编辑和批量编辑均需支持。最大 SessionID 数量最小为 3、默认 35，空闲过期时间支持秒、分钟、天，默认 1 天。
- 每个账号使用独立槽位记录下游显式 SessionID 的最后请求时间。未满时原子接纳新 SessionID；默认关闭“槽位满时自动轮换”，关闭时满槽后已在槽位内的 SessionID 仍可刷新并调度，新 SessionID 必须跳过该账号并继续尝试其他候选账号。已在槽位内或暂存命中的 SessionID 遇到归属账号并发已满时，不得继续硬粘连排队，应尝试其他有空闲并发的候选账号；目标受控账号接纳成功后必须原子迁移唯一槽位归属。只有所有合适候选均无空闲并发时才回退现有排队策略。
- OpenAI OAuth 母账号可单独开启“槽位满时自动轮换”，用于账号较少、严格槽位容易快速耗尽的场景。开启后，满槽遇到新 SessionID 时必须在同一 Lua 原子操作中把最后请求时间最早的活跃槽位移入同账号请求暂存区，保留唯一归属并把调度偏好改为立即生效的暂存偏好，再接纳新会话；暂存周期等于账号配置的 SessionID 过期时间，暂存会话回访时优先原账号。活跃槽位总数不得超过配置上限，暂存槽位不计入活跃容量。若管理员将上限调低到当前槽位数以下，应一次轮换足够数量恢复到新上限。该开关默认关闭，不能改变现有严格限制语义。
- 同一个“下游 API Key + 原始 SessionID”哈希在所有受控账号间必须只有一个槽位归属。账号 failover 时通过同一个 Lua 脚本完成接纳和转移；自动轮换关闭时，普通跨账号尝试遇到目标满槽必须保留旧归属，不能先删后拒绝。自动轮换开启时，目标账号先原子把最久未请求槽位移入目标账号暂存区，再从旧账号移除当前 SessionID 并转入目标账号。
- 服务启动时必须扫描旧版 `openai_session_limit:account:*` 数据并建立唯一归属；若同一哈希已存在于多个账号，保留最后活跃时间最新的一条并删除其他槽位，相同时间时使用账号 ID 确定性裁决。
- SessionID 超过配置的空闲时间后必须从活跃槽位移入该账号暂存区，立即释放活跃容量；暂存保留时间为额外一个相同的空闲过期周期。暂存期内该 SessionID 再次请求时优先调度原账号：原账号有空槽则原子移回活跃槽位；原账号已满且自动轮换关闭时删除暂存和调度偏好并继续选择其他账号，自动轮换开启时则淘汰最久未请求的活跃槽位并原子回迁。暂存期结束后清理暂存归属；关闭控制或删除账号时应同时清理活跃槽位、暂存槽位和相关归属键。
- Spark 影子账号不能独立配置，调度到影子账号时必须读取母账号配置并与母账号共享同一组槽位。
- 本功能限制的是下游显式 SessionID，并且按“下游 API Key + 原始 SessionID”哈希隔离；SessionID 控制的 Redis 槽位与归属不得存储原始值，也不得把仅由请求内容或内部 fallback 推导出的 sticky hash 当成真实 SessionID。
- 显式标识至少覆盖 `session-id`、`session_id`、`conversation_id`、OpenCode 会话头、CodeBuddy 会话头及 `prompt_cache_key`。外部请求没有显式 SessionID 时必须完全绕过 SessionID 控制，开启控制的账号仍按关闭控制时的普通流程参与调度，不查询母账号、不读写槽位且不因 SessionID 控制跳过账号；内部探测和模型发现没有外部请求标记时同样不受影响。
- 管理员使用记录必须把持久化的 `usage_logs.session_id` 作为独立 SessionID 列展示，不能用每次请求唯一的 `request_id` 冒充会话标识。该列默认可见并紧邻账号列，支持复制及 Excel 导出，便于核对同一 SessionID 是否始终调度到同一账号；没有显式 SessionID 的历史或当前请求显示 `-`。用量落库必须复用本次调度已解析的下游显式值，除请求头外还要覆盖请求体 `prompt_cache_key`；不得把内容指纹、Grok `previous_response_id`、WebSocket 连接兜底等仅供内部 sticky 调度的种子写入 `usage_logs.session_id`。
- SessionID 控制与 Codex 指纹收敛互相独立：指纹收敛可改变上游看到的会话/设备标识，本地槽位仍必须按下游原始显式 SessionID 计数。Codex 指纹收敛默认必须为 `off`，创建、编辑和批量编辑账号时不能因为启用 SessionID 控制而隐式开启收敛；已有账号显式配置的 `device` / `session` / `full` 模式必须原样保留。
- Redis 使用独立键空间：活跃槽位 `openai_session_limit:account:{accountID}`、暂存槽位 `openai_session_limit:staged:{accountID}`、物理唯一归属定位 `openai_session_limit:owner:{sessionIDHash}` 和带活跃/暂存截止时间的限时调度元数据 `openai_session_limit:preferred:{sessionIDHash}`；不得与 Anthropic 的 `session_limit:account:{accountID}` 共用数据。调度查询只能在暂存窗口返回原账号，活跃窗口仍使用既有调度顺序，并在“活跃周期 + 一个暂存周期”结束时准时失效；唯一归属定位要比共享 ZSET 多保留清理缓冲，防止偏好失效后跨账号接纳留下重复物理成员。接纳、跨账号转移、活跃转暂存、暂存回迁、满槽剔除、过期清理、存在刷新和数量判断必须由 Lua 脚本原子完成；不得把原始 SessionID 写入 Redis key 或 value。
- Redis 异常时不得让受控账号失败开放；应仅跳过当前受控账号并继续调度其他账号。所有候选账号均不可用时再返回无可用账号错误。
- SessionID 拒绝发生在统一 OpenAI 调度出口。已抢到的并发槽必须释放，旧 sticky 绑定必须删除，账号加入本次请求排除集合后重新执行现有调度。有效的暂存归属偏好必须先于 `previous_response_id`、普通 sticky、最高调度和负载均衡尝试原账号，但仍必须通过账号状态、分组、模型、能力、传输、利润控制、代理隔离和 Group 准入；活跃槽位不得获得这项额外优先级，偏好账号不可用时回到既有调度链路。受控显式 SessionID 命中的暂存偏好、普通 sticky 或 `previous_response_id` 账号并发已满时，也必须让出硬粘连并继续负载调度；跨账号接纳继续通过 Lua 原子迁移，不能产生双重物理归属。
- 账号列表容量列复用现有会话容量徽标展示当前槽位数、上限和过期时间；运行态数量继续使用 `active_sessions`，但查询 OpenAI 独立键空间。

关键 extra key：

- `openai_session_control_enabled`
- `openai_session_max_count`
- `openai_session_idle_timeout_seconds`
- `openai_session_slot_rotation_enabled`

关键文件：

- 后端配置、调度与用量会话字段：`backend/internal/service/account_openai_session_control.go`、`backend/internal/service/openai_session_control.go`、`backend/internal/service/openai_account_scheduler.go`、`backend/internal/service/openai_gateway_scheduling.go`、`backend/internal/service/openai_ws_forwarder_support.go`、`backend/internal/service/session_id.go`、`backend/internal/service/usage_log.go`、`backend/internal/handler/dto/types.go`、`backend/internal/handler/dto/mappers.go`、`backend/internal/repository/usage_log_repo_query.go`
- 后端缓存与管理：`backend/internal/service/session_limit_cache.go`、`backend/internal/repository/session_limit_cache.go`、`backend/internal/repository/scheduler_cache.go`、`backend/internal/service/admin_account.go`、`backend/internal/service/admin_service.go`、`backend/internal/handler/admin/account_handler.go`、`backend/internal/handler/dto/mappers.go`
- 外部入口：`backend/internal/handler/openai_embeddings.go`、`backend/internal/handler/openai_live.go`、`backend/internal/service/openai_live.go`
- 前端：`frontend/src/components/account/OpenAISessionControlFields.vue`、`frontend/src/components/account/openaiSessionControl.ts`、`frontend/src/components/account/CreateAccountModal.vue`、`frontend/src/components/account/EditAccountModal.vue`、`frontend/src/components/account/BulkEditAccountModal.vue`、`frontend/src/components/account/AccountCapacityCell.vue`、`frontend/src/components/common/Toggle.vue`、`frontend/src/components/admin/usage/UsageTable.vue`、`frontend/src/views/admin/UsageView.vue`、`frontend/src/types/index.ts`

回归检查：

- 同一账号并发提交多于上限的不同 SessionID，Redis 最终接纳数量不能超过上限；已接纳 SessionID 在满槽时仍可刷新。
- 自动轮换默认关闭时必须保持严格拒绝新 SessionID；开启后，满槽新请求应把最久未请求的活跃槽位移入同账号暂存区，保留唯一归属、建立暂存偏好并接纳新会话，最终活跃数量仍等于上限。配置上限调低后首次接纳也必须一次轮换到新上限；创建、编辑和批量编辑均应正确保存该开关。
- 同一 SessionID 并发调度或 failover 到多个受控账号后，所有账号活跃槽位与暂存槽位合计只能保留一条；普通跨账号目标有容量时原子转移，目标满槽且自动轮换关闭时旧账号槽位及归属不得丢失，自动轮换开启时应原子把目标最旧槽位移入暂存区后完成唯一归属转移。
- 活跃槽位过期后应立即不计入容量并进入暂存；暂存命中时高级调度和旧调度都优先选择原账号并回迁，不能被 `previous_response_id`、最高调度或普通负载顺序抢先。
- 暂存命中但原账号活跃槽位已满时，自动轮换关闭应原子删除该暂存成员及偏好后切换其他账号；自动轮换开启应在原账号淘汰最久未请求槽位并完成回迁。调度偏好自然到期后再跨账号接纳，也不得在旧账号留下重复物理成员。
- 启动迁移遇到旧版跨账号重复槽位时只保留最后活跃的一条，并为所有现存槽位建立带过期时间的唯一归属键。
- 槽位满或 Redis 异常时，调度器释放当前账号槽并切换其他候选账号；不能直接结束，也不能反复选择同一账号。显式 SessionID 所属账号并发满时，高级调度、旧版批量负载调度、旧版直接选择和 `previous_response_id` 硬粘连都必须继续尝试空闲账号并在接纳时迁移槽位；无显式 SessionID 的请求则不得进入 SessionID 控制或访问其 Redis 槽位。
- 不同下游 API Key 使用相同原始 SessionID 时占用不同槽位；除受权限控制、专用于会话关联的 `usage_logs.session_id` 字段外，运行日志和 Redis 不出现原始 SessionID。
- 到期活跃槽位会在后续接纳或容量查询时原子转入暂存，暂存到期后再清理；关闭控制、批量关闭和删除账号会同时清理活跃槽位和暂存槽位。
- 母账号和 Spark 影子账号共享母账号槽位，影子账号不能通过批量编辑获得独立配置。
- OpenAI SessionID 键空间与 Anthropic 会话限制完全隔离，现有 Anthropic 会话数量、窗口费用及 RPM 展示不受影响。
- 账号创建、编辑和批量编辑的 Codex 指纹收敛缺省值保持 `off`；SessionID 控制开关与指纹模式分别保存，互不覆盖。
- HTTP Responses、Chat Completions、Messages、Images、Embeddings、WebSocket 和 Live 外部入口均不能绕过控制；内部账号探测仍可正常选号。
- 使用记录在请求仅通过请求体携带 `prompt_cache_key` 时必须保存该显式 SessionID；请求只命中内容指纹、Grok `previous_response_id` 或 WebSocket 内部兜底时仍须保存为空，不能泄露内部调度种子。

### 2.16 邀请返利支付宝提现

功能要求：

- 系统设置的“邀请返利”区域可独立开启返利提现，并配置最低提现金额和百分比手续费；关闭提现只禁止新申请，不能隐藏历史记录或阻止管理员处理已有申请。
- 用户在“返利额度转余额”下方可提交支付宝提现申请，并通过弹窗按状态查看历史记录；支付宝账号必须在数据库中加密保存，用户接口只返回脱敏账号，管理员提现列表才返回解密后的完整账号。
- 用户可维护最多 10 个支付宝提现账号，支持新增、修改、设为默认和删除；提现表单必须选择已保存账号，不能要求用户每次申请都重新输入。用户账号列表只返回脱敏值，完整账号与历史提现快照继续使用 AES-256-GCM 加密存储。
- 账号簿首次上线时必须把每个已有提现用户最近一次申请中的加密账号迁入并设为默认。删除或修改账号簿记录不能改变历史提现申请；删除默认账号后如仍有其他账号，必须自动提升一个新默认账号。
- 提现申请传入 `withdrawal_account_id` 时必须同时校验账号归属，禁止引用其他用户账号；申请创建时把所选账号的密文和脱敏值复制为独立快照。旧客户端直接提交 `alipay_account` 的兼容路径可以保留，但新用户界面必须使用账号 ID。
- 提现金额使用总额口径：例如申请 `$100`、手续费 `1%`，从可用返利中冻结并最终扣除 `$100`，手续费快照为 `$1`，管理员实际转账金额快照为 `$99`。申请后的设置变更不能影响历史申请的费率、手续费或实际转账额。
- 创建申请必须在同一数据库事务中原子执行“可用返利减少 + 冻结返利增加 + 插入 pending 申请”；余额不足时三者都不得发生。并发申请必须通过条件更新防止返利透支。
- 管理员确认手动转账后，申请状态改为 `paid` 并从冻结返利中扣除申请总额；管理员驳回时必须填写原因，状态改为 `rejected`，并把申请总额从冻结返利退回可用返利。
- `paid` / `rejected` 是不可逆终态；管理员处理必须锁定申请行并只允许 `pending` 转换一次。创建和管理端处理接口必须使用 `Idempotency-Key`，避免重复点击、网络重试或并发操作造成重复冻结、扣除或退回。
- 管理端“邀请返利”菜单下必须保留“用户提现”页面，支持状态、日期和用户/申请单号筛选，并清晰展示申请总额、手续费率/金额、实际转账金额、支付宝账号、处理人、处理时间和驳回原因。

关键 setting key：

- `affiliate_withdrawal_enabled`
- `affiliate_withdrawal_min_amount`
- `affiliate_withdrawal_fee_rate`

关键文件：

- 数据库与事务：`backend/migrations/226_affiliate_withdrawals.sql`、`backend/migrations/227_affiliate_withdrawal_accounts.sql`、`backend/internal/repository/affiliate_withdrawal_repo.go`、`backend/internal/repository/affiliate_withdrawal_account_repo.go`
- 服务与设置：`backend/internal/service/affiliate_withdrawal.go`、`backend/internal/service/affiliate_withdrawal_account.go`、`backend/internal/service/affiliate_service.go`、`backend/internal/service/domain_constants.go`、`backend/internal/service/setting_features.go`、`backend/internal/service/setting_parse.go`、`backend/internal/service/setting_update.go`、`backend/internal/service/settings_view.go`
- API 与装配：`backend/internal/handler/user_handler.go`、`backend/internal/handler/admin/affiliate_handler.go`、`backend/internal/server/routes/user.go`、`backend/internal/server/routes/admin.go`、`backend/internal/service/wire.go`、`backend/cmd/server/wire_gen.go`
- 前端：`frontend/src/views/user/AffiliateView.vue`、`frontend/src/components/user/AffiliateWithdrawalAccountsDialog.vue`、`frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/api/user.ts`、`frontend/src/api/admin/affiliates.ts`、`frontend/src/api/admin/settings.ts`、`frontend/src/components/layout/AppSidebar.vue`、`frontend/src/router/index.ts`、`frontend/src/types/index.ts`

回归检查：

- 申请 `$100` 且手续费为 `1%` 后，可用返利减少 `$100`、冻结返利增加 `$100`，记录中的手续费为 `$1`、实际转账额为 `$99`；不得只冻结 `$99` 或额外扣除 `$101`。
- 余额不足和低于最低金额时不创建记录、不改变可用/冻结返利；两笔并发申请不能透支同一份可用返利。
- 驳回后整笔申请总额退回可用返利；确认已提现后整笔申请总额从冻结返利扣除；同一申请再次处理返回冲突且不再改变余额。
- 用户列表不返回密文或完整支付宝账号；管理员列表可读取完整账号，数据库不得存明文。
- 用户可复用默认账号提交多次提现；账号列表只显示脱敏值，跨用户账号 ID 被拒绝。修改或删除已使用账号后，历史提现中的账号快照保持不变。
- 首个账号自动成为默认账号；切换默认后只有一个默认账号；删除默认账号会提升剩余账号。已有历史提现用户升级后自动获得最近一次账号，且无需重新输入即可申请。
- 关闭提现开关后，新申请被拒绝，用户历史记录和管理员 pending 处理仍可正常使用。

## 3. 合并后必测清单

### 3.1 系统设置保存验证

1. 管理员进入系统设置。
2. 关闭“在线生图”，保存。
3. 刷新系统设置页面，确认“在线生图”仍为关闭。
4. 用户侧边栏确认“在线生图”菜单隐藏。
5. 直接访问 `/images`，确认会被重定向。
6. 重新开启“在线生图”，保存并刷新，确认菜单恢复。
7. 对“视频任务”重复同样流程：关闭、保存、刷新、菜单隐藏、直访 `/videos` 被拦截、重新开启恢复。

### 3.2 public settings 验证

检查 `/api/v1/settings/public` 响应必须包含：

```json
{
  "image_playground_enabled": true,
  "video_jobs_enabled": true
}
```

当后台关闭开关后，应返回：

```json
{
  "image_playground_enabled": false,
  "video_jobs_enabled": false
}
```

HTML 注入的 `window.__APP_CONFIG__` 也应包含这两个字段，避免菜单首屏状态错误。

### 3.3 在线生图高级模式验证

1. 打开 `/images`。
2. 勾选“高级模式”。
3. 确认能看到接口路径和 JSON 请求参数。
4. 修改 JSON 中的：
   - `prompt`
   - `model`
   - `size`
   - `n`
5. 确认左侧表单同步更新。
6. 输入非法 JSON，确认出现错误提示且生成按钮禁用。
7. 输入非法路径，确认出现错误提示且生成按钮禁用。
8. 正常生成一次，确认高级模式能看到请求和响应 JSON。
9. 模拟上游失败，确认页面展示上游错误消息，高级模式展示错误详情。

### 3.4 账号池模式与最高调度轮转验证

1. 新建或编辑 OAuth 账号，确认可配置池模式、同账号重试次数和重试状态码。
2. 批量编辑 OAuth/API Key 账号，确认可批量开启/关闭池模式，并保存 retry count / retry status codes。
3. 开启最高调度轮转，选择指定分组、OAuth/API Key 类型和轮转数量 `n`，保存后弹窗应关闭。
4. 刷新账号列表，确认范围内最高调度账号数量按配置分配，最高调度账号排序在前。
5. 轮转开启后，范围内账号的最高调度列按钮应禁用手动切换。
6. 关闭当前最高调度账号的调度或将状态改为异常后，如果符合条件的最高调度账号数量低于 `n`，应立即补位到其他符合条件账号。
7. 非当前最高调度账号状态变化时也允许触发检查，但只有当前符合条件的最高调度账号数量不等于 `n` 时才真正执行轮转。
8. 关闭最高调度轮转后，范围内账号最高调度状态应被清空。
9. 顶部“最高调度轮转”按钮开启态只改变文字颜色，不改变原按钮背景/边框。

### 3.5 API Key 多 Group 与 Gateway Failover 验证

1. 创建一个 API Key 并绑定至少两个 Group，确认详情和更新接口返回完整且按配置顺序排列的 `group_ids` / `groups`，并且 `group_id` 等于第一个 Group。
2. 调整 Group 顺序后发起请求，确认第一次尝试固定使用新的第一个 Group，而不是按 Group ID 或名称排序。
3. 让第一个 Group 出现可恢复的 subscription、Group RPM、账号池耗尽或账号级 failover 耗尽，确认先完成当前 Group 内账号处理，再按顺序切换到下一个 Group。
4. 确认切换 Group 时重新执行 subscription、billing、channel mapping、composite target、Claude Code only 和 Group 用户并发准入；前一个 Group 的失败账号与利润 veto 不得影响后一个 Group。
5. 确认账号切换与 Group 切换共用请求级预算；同账号重试、利润 veto、无下一个 Group、客户端取消、已写语义响应和不可重试错误不消费预算。
6. 流式请求在 Group1 只写 compact keepalive 时仍允许切换；写出有效语义 SSE/JSON 后不得切换或拼接 Group2 响应。
7. 配置 invalid-request fallback，确认 fallback 只执行一次、使用完整 Group attempt、失败后不递归 fallback 也不错误推进普通 Group 顺序，并遵守共享预算。
8. 确认成功 Group 的 sticky session、billing、usage 和审计记录都使用成功 Group，异步 usage 闭包不捕获初始或可变 API Key。

后端定向测试：

```bash
cd backend
go test ./internal/handler -run 'TestRequestFailoverBudget|TestOrderedGroupFailoverState|TestIsRetryableGroupBillingError' -count=1
go test ./internal/service -run 'OpenAICompact' -count=1
```

### 3.6 类型和测试命令

前端优先按当前偏好跑 typecheck / lint-only，不强制 build：

```bash
eval "$(fnm env --use-on-cd --shell bash)"
pnpm --dir frontend typecheck
```

后端相关测试：

```bash
cd backend
go test ./internal/handler ./internal/handler/admin
GOMAXPROCS=2 go test -tags=unit -run TestSettingService_ImagePlaygroundAndVideoJobsFeatureSwitches -count=1 -p=1 ./internal/service
```

如果 Windows 本地磁盘或内存不足导致完整 service 测试失败，可先用上述低并发定向测试确认本功能；在服务器或空间充足环境再跑完整测试。

### 3.7 OpenAI SessionID 控制验证

1. 新建或编辑 OpenAI OAuth 母账号，开启 SessionID 控制，确认默认上限为 35、默认过期时间为 1 天；低于 3 的上限无法保存。
2. 批量选择 OpenAI OAuth 母账号，确认可统一开启、关闭并设置上限和过期时间；混入 API Key、其他平台或 Spark 影子账号时后端拒绝整批写入。
3. 将上限设为 3，用三个不同显式 SessionID 请求同一账号后，账号列表容量徽标显示 `3/3`；三个已有 ID 继续可用，第四个 ID 调度到其他空闲账号。
4. 等待配置的空闲时间后再次请求，确认旧槽位释放且新 SessionID 可以进入；关闭控制后容量徽标和 Redis 槽位清除。
5. 开启完全指纹收敛后重复测试，确认上游指纹可保持收敛，但本地不同下游 SessionID 仍分别计数。
   同时确认新建账号和批量编辑的指纹收敛缺省值仍为 `off`，单独开启 SessionID 控制不会改变指纹模式。
6. 让同一 SessionID 因过载或上游错误从账号 A 切换到账号 B，确认 B 有容量时 A 的槽位被原子移除、B 只保留一条；B 已满时切换被拒绝且 A 的原槽位仍存在。
7. 等待账号 A 的 SessionID 达到空闲过期时间，确认活跃计数释放且哈希进入 A 的暂存区；在额外一个过期周期内再次请求，确认高级调度和旧调度都优先选择 A，并将其从暂存区原子移回活跃槽位。
8. 关闭自动轮换时，暂存期内先用其他 SessionID 填满账号 A，再让暂存 SessionID 请求，确认 A 的暂存成员和偏好被删除、请求继续调度到其他账号；等待暂存期自然结束后重复请求，确认不再偏好 A 且所有账号间仍只有一个物理槽位成员。
9. 开启“槽位满时自动轮换”后，用第四个 SessionID 请求已满的账号，确认最久未请求的旧会话从活跃槽位移入请求暂存区、仍归属原账号且立即获得暂存偏好；第四个会话进入活跃槽位，活跃数量保持上限。旧会话在暂存期回访时应优先原账号，并把另一最久未请求的活跃会话轮换进暂存区。
10. 打开管理员使用记录，确认 SessionID 列默认显示在账号列旁边；同一会话的多条记录可直接对照账号，SessionID 可复制且 Excel 导出包含该列，请求 ID 仍保持独立列和原默认隐藏规则。分别用请求头和请求体 `prompt_cache_key` 发起请求，两者都应落库；只使用内容指纹、Grok `previous_response_id` 或 WebSocket 内部兜底时该列应为 `-`。

后端定向测试：

```bash
cd backend
go test ./internal/service ./internal/repository ./internal/handler/admin -run 'Test(OpenAISession|NormalizeOpenAISession|MergeAndValidateOpenAISession)' -count=1
go test -tags unit ./internal/service -run 'TestAdminService_BulkUpdateAccounts_(OpenAISessionControl|DisablingOpenAISessionControl)' -count=1
```

### 3.8 邀请返利提现验证

1. 在系统设置开启返利提现，设置最低金额 `$10`、手续费 `1%`，保存并刷新，确认配置正确回显。
2. 使用可用返利至少 `$100` 的用户申请提现 `$100`，确认页面展示冻结 `$100`、手续费 `$1`、预计到账 `$99`，提交后可用返利减少 `$100` 且冻结返利增加 `$100`。
3. 管理员在“邀请返利 > 用户提现”找到申请，确认支付宝完整账号与金额快照正确；标记已提现后冻结返利减少 `$100`，重复处理被拒绝。
4. 再提交一笔申请并填写原因驳回，确认整笔申请总额退回可用返利，用户提现记录展示状态和驳回原因。
5. 关闭提现开关，确认用户不能新建申请，但仍能查看历史记录，管理员仍能处理关闭前的 pending 申请。
6. 打开“提现账号”，新增两个支付宝账号，确认完整账号不会出现在列表/API 响应中；切换默认账号后提现表单自动选中该账号。
7. 使用保存账号提交提现后修改或删除该账号，确认历史申请仍显示原脱敏账号，管理员仍能读取并向申请快照中的完整账号转账。
8. 尝试提交其他用户的 `withdrawal_account_id`，确认请求被拒绝且可用/冻结返利均不变化；删除默认账号后确认剩余账号自动成为默认。

后端定向测试：

```bash
cd backend
go test ./internal/service -run 'Test(CreateAffiliateWithdrawal|AffiliateWithdrawal)' -count=1
go test -tags integration ./internal/repository -run 'TestAffiliateWithdrawal(Account|Repository)' -count=1
```

## 4. 合并冲突处理重点

遇到以下文件冲突时，必须人工合并，不能直接选择上游覆盖：

- `backend/internal/service/setting_parse.go`
- `backend/internal/service/setting_update.go`
- `backend/internal/service/setting_public.go`
- `backend/internal/service/settings_view.go`
- `backend/internal/service/account.go`
- `backend/internal/service/ratelimit_service.go`
- `backend/internal/service/account_highest_scheduling_rotation.go`
- `backend/internal/service/admin_account.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/service/account_openai_session_control.go`
- `backend/internal/service/openai_session_control.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- `backend/internal/service/session_limit_cache.go`
- `backend/internal/repository/session_limit_cache.go`
- `backend/internal/repository/scheduler_cache.go`
- `backend/internal/service/wire.go`
- `backend/internal/service/affiliate_service.go`
- `backend/internal/service/affiliate_withdrawal.go`
- `backend/internal/service/affiliate_withdrawal_account.go`
- `backend/internal/repository/affiliate_withdrawal_repo.go`
- `backend/internal/repository/affiliate_withdrawal_account_repo.go`
- `backend/cmd/server/wire_gen.go`
- `backend/internal/repository/account_repo.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/handler/dto/settings.go`
- `backend/internal/handler/admin/account_handler.go`
- `backend/internal/handler/admin/setting_handler_update.go`
- `backend/internal/handler/admin/affiliate_handler.go`
- `backend/internal/handler/user_handler.go`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/views/admin/AccountsView.vue`
- `frontend/src/views/user/ImagePlaygroundView.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/account/BulkEditAccountModal.vue`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/components/account/OpenAISessionControlFields.vue`
- `frontend/src/components/account/AccountCapacityCell.vue`
- `frontend/src/components/common/Toggle.vue`
- `frontend/src/router/index.ts`
- `frontend/src/utils/featureFlags.ts`
- `frontend/src/stores/app.ts`
- `frontend/src/types/index.ts`
- `frontend/src/api/imagePlayground.ts`
- `frontend/src/api/admin/accounts.ts`
- `frontend/src/views/user/AffiliateView.vue`
- `frontend/src/components/user/AffiliateWithdrawalAccountsDialog.vue`
- `frontend/src/views/admin/affiliates/AdminAffiliateWithdrawalsView.vue`
- `frontend/src/api/admin/affiliates.ts`

合并后至少 grep 确认这些关键字仍存在：

```text
image_playground_enabled
video_jobs_enabled
ImagePlaygroundEnabled
VideoJobsEnabled
FeatureFlags.imagePlayground
FeatureFlags.videoJobs
requiresImagePlayground
requiresVideoJobs
advancedEnabled
advancedRequestPath
advancedRequestBodyText
extractImageGenerationErrorMessage
IMAGE_GENERATION_TIMEOUT_MS
pool_mode
pool_mode_retry_count
pool_mode_retry_status_codes
highest_scheduling_rotation_config
highest_scheduling_mode
HighestSchedulingRotationConfig
HighestSchedulingRotationReconciler
InjectHighestSchedulingRotationReconciler
ShouldReconcileHighestSchedulingRotation
reconcileHighestSchedulingRotationForAccounts
highestSchedulingRotationCandidate
highestSchedulingRotationEnabled
saveHighestSchedulingRotation
handleToggleHighestScheduling
openai_session_control_enabled
openai_session_max_count
openai_session_idle_timeout_seconds
openai_session_slot_rotation_enabled
RegisterOpenAISessionID
GetOpenAIStagedSessionAccountID
registerOpenAISessionControl
affiliate_withdrawal_enabled
affiliate_withdrawal_min_amount
affiliate_withdrawal_fee_rate
AffiliateWithdrawal
AffiliateWithdrawalAccount
ProcessAffiliateWithdrawal
```

## 5. 本次已修复过的问题记录

### 5.1 设置保存无效

现象：

- 后台关闭“在线生图 / 视频任务”后保存无效，刷新后状态不对。

根因：

- 后端 `parseSettings` 没有把 `image_playground_enabled` / `video_jobs_enabled` 解析回管理端 `SystemSettings`。
- 导致管理端读取回显和后续保存链路可能被零值覆盖。

修复点：

- `backend/internal/service/setting_parse.go`
  - 补齐：
    - `ImagePlaygroundEnabled: !isFalseSettingValue(settings[SettingKeyImagePlaygroundEnabled])`
    - `VideoJobsEnabled: !isFalseSettingValue(settings[SettingKeyVideoJobsEnabled])`
- `backend/internal/service/setting_service_update_test.go`
  - 增加 `TestSettingService_ImagePlaygroundAndVideoJobsFeatureSwitches`

### 5.2 在线生图失败只显示通用错误

现象：

- 上游返回明确错误原因时，页面仍只显示“图片生成失败”。

修复点：

- `frontend/src/api/imagePlayground.ts`
  - 增加/保留 `extractImageGenerationErrorMessage`
- `frontend/src/views/user/ImagePlaygroundView.vue`
  - catch 分支优先展示上游错误消息。

### 5.3 在线生图 / 视频任务菜单合并后丢失

现象：

- 合并上游后菜单或开关链路丢失。

保护点：

- 后端 settings、public settings、HTML 注入、前端 feature flag、sidebar、router guard、系统设置页必须一起保留。
- 不能只恢复菜单 UI，不恢复后端设置和 public settings。

## 6. 最小验收标准

下次合并完成后，只有同时满足以下条件才算没有丢失本项目自定义功能：

- 系统设置中存在“在线生图”和“视频任务”两个开关。
- 两个开关保存后刷新仍保持正确状态。
- 关闭在线生图后 `/images` 菜单隐藏且直访被拦截。
- 关闭视频任务后 `/videos` 菜单隐藏且直访被拦截。
- `/api/v1/settings/public` 和 `window.__APP_CONFIG__` 都包含两个开关字段。
- 在线生图高级模式可编辑路径和 JSON 参数。
- 高级模式非法输入会提示并禁用生成。
- 在线生图失败会直接展示上游错误消息。
- OAuth 账号可配置并使用池模式重试，批量编辑可批量修改池模式字段。
- 最高调度轮转可按分组、账号类型和数量 `n` 配置并保存，保存后弹窗关闭。
- 最高调度轮转开启后，范围内账号禁止手动切换最高调度，且状态/调度变化导致符合条件最高调度数量低于 `n` 时会被动补位。
- 关闭最高调度轮转后，范围内最高调度状态会被清空。
- 最高调度列可点击手动切换，调度开关旁没有重复的最高调度按钮。
- API Key 多 Group 绑定保留 `group_ids` 配置顺序，并同步正确的 `group_id` legacy mirror。
- Group failover 先完成当前 Group 内账号处理，再按顺序切换；账号与 Group 切换共享预算且 Group-local 状态隔离。
- 每次 Group 切换重新执行 subscription、billing、mapping、并发和平台准入，成功 Group 的 sticky/usage 归属正确。
- 有效流式语义内容写出后不切换 Group，compact keepalive 不阻断 failover，invalid-request fallback 最多执行一次。
- OpenAI OAuth SessionID 控制可在创建、编辑和批量编辑中配置；默认模式满槽后已有 ID 可用、新 ID 自动切换账号，活跃过期后进入一个周期的暂存区。开启自动轮换后，被轮换的旧会话也必须进入同账号暂存区并在回访时优先原账号。受控会话命中并发已满的归属账号时会迁移到空闲受控账号；无显式 SessionID 时完全走普通调度且不读写槽位。
- OpenAI SessionID 槽位按下游 API Key 隔离，与 Anthropic 会话限制及 Codex 指纹收敛互不污染；Spark 影子账号与母账号共享槽位。
- 邀请返利提现按申请总额冻结和最终扣除，手续费仅从管理员实际转账金额中扣减；驳回整笔退回、终态不可重复处理、支付宝账号加密保存。
- 用户可管理并复用支付宝提现账号，默认账号和归属校验正确；账号变更或删除不影响历史提现快照，账号列表不泄露完整值或密文。
- 前端 typecheck 通过。
- 新增的 service 和 handler 回归测试通过。
