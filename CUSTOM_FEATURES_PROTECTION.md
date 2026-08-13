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
- 用户分组配置中，标签命中的专属 Group 显示“标签名 + 专属分组”，并且不可通过 checkbox 撤销。
- 删除标签、移除用户标签或移除 Group 标签后，派生授权自动撤销，但不能删除仍存在的手工授权。
- subscription Group 不能通过标签绕过 active subscription 检查。

关键文件：

- 后端：`backend/migrations/221_add_user_tag_group_authorization.sql`、`backend/internal/service/user_tag.go`、`backend/internal/repository/user_tag_repo.go`、`backend/internal/service/api_key_service.go`、`backend/internal/service/api_key_auth_cache_impl.go`
- 前端：`frontend/src/components/admin/user/UserTagManagementModal.vue`、`frontend/src/components/admin/user/UserTagMultiSelect.vue`、`frontend/src/components/admin/user/UserAllowedGroupsModal.vue`、`frontend/src/views/admin/GroupsView.vue`、`frontend/src/views/admin/UsersView.vue`

容易丢失点：

- 标签授权查询必须过滤已删除、非 active、非 standard 或非 exclusive Group，并对多标签 OR 查询去重。
- auth snapshot 需要包含标签派生 Group，缓存版本变化后旧快照必须回源。
- API Key 主 Group、有序次级 Group、Model Plaza 和 Available Channels 必须使用有效授权并集。

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

### 2.13 分组价格监测公告

功能要求：

- 公告设置中可以启用/停用分组价格监测。
- 监测范围支持全部分组或多个指定分组，监测周期单位为秒。
- 公告状态、通知方式和展示条件复用公告模型语义；监测有效期通过 `duration_days` 配置。
- 每个监测周期内所有倍率变化必须合并为一条标题为“分组价格调整通知”的 Markdown 公告。
- 每个周期仍只创建一条逻辑公告；倍率变化必须同时保存为带 `group_id` 的结构化明细，用户端不得依赖 Markdown 反解析权限。
- 公告内容必须逐项展示分组名称、涨价/降价方向和旧倍率到新倍率（如 `0.07 -> 0.08`），并突出方向。
- 用户查看价格公告时必须按当前分组权限动态过滤明细：公开标准分组所有用户可见；标准专属分组取手工授权与标签派生授权并集；订阅分组必须有有效订阅。用户后来获得权限时，可以看到仍在有效期内的历史公告对应明细；权限撤销后立即隐藏。
- 价格公告已读状态必须按 `(announcement_id, user_id, group_id)` 记录。用户读过公开明细后又获得专属分组权限时，该逻辑公告必须重新显示为未读；不能只使用公告级 `announcement_reads`。
- 服务启动后后台轮询；首次检查只建立基线，保存配置、停用或修改范围后必须重新建立基线，不能把期间变化误报。
- 管理端必须显示距离下一次实际监测的实时倒计时；时间基准由后端返回的 `next_check_at` 与 `server_time` 决定，配置保存后应立即重排周期，不能仅按页面打开时间自行估算。

关键 setting key：

- `group_price_monitor_config`

关键文件：

- 后端：`backend/internal/service/group_price_monitor.go`、`backend/internal/handler/admin/group_price_monitor_handler.go`、`backend/internal/server/routes/admin.go`、`backend/internal/service/wire.go`、`backend/cmd/server/wire.go`
- 公告权限与持久化：`backend/internal/service/announcement_service.go`、`backend/internal/repository/announcement_repo.go`、`backend/internal/repository/announcement_read_repo.go`、`backend/ent/schema/announcement_group_price_change.go`、`backend/ent/schema/announcement_group_price_read.go`
- 前端：`frontend/src/views/admin/AnnouncementsView.vue`、`frontend/src/api/admin/announcements.ts`、`frontend/src/types/index.ts`

界面约束：

- 价格监测配置必须通过公告列表工具栏入口打开独立弹窗，不能长期嵌在公告列表上方占用页面空间。
- 指定分组选择必须复用账号编辑弹窗使用的公共 `frontend/src/components/common/GroupSelector.vue`，不能在公告页复制一套分组选择器；“全部分组”作为独立总开关。
- 启用配置保存后必须立即建立当前倍率基线；服务启动读取已启用配置时也必须建立基线，不能吞掉第一次后续实际价格变化。
- 系统自动价格监测公告的 `created_by` 必须为空，管理员手工公告由服务写入管理员 ID；管理端公告列表必须区分“系统发布”和“管理员发布”。
- 用户公告 DTO 必须透传 `created_by`；公告铃列表、公告详情和弹窗都必须依据该字段标识“系统发布”或“管理员发布”，不能根据标题猜测来源。
- 价格变化公告内容必须使用 Markdown 表格，展示变动数量、分组、调整类型、调整前和调整后；方向（▲ 涨价/▼ 降价）及新倍率必须突出，分组名称保持纯文本，不能包裹 `**`。
- 分组价格监测启用开关必须通过状态绑定控制轨道颜色和滑块位移，并保留平滑过渡动画；不能依赖嵌套元素上的 `peer-checked` 选择器。
- 已启用时，公告列表工具栏和价格监测设置弹窗都要显示逐秒更新的下一次监测倒计时；到期后重新读取后端运行时计划，停用时不显示虚假的下一次时间。
- 分组价格监测不再让管理员填写开始/结束时间，前端只提交有效天数；监测配置本身不得固化 `starts_at` / `ends_at`，每条自动公告发布时以后端当前时间作为开始时间，并按 `duration_days` 日历天计算该公告的结束时间。
- 管理员修改分组倍率时必须记录每一次实际变更，监测周期结束时合并发布；不能只依赖固定间隔轮询，否则同一周期内的快速变更（如 `0.13 -> 0.12 -> 0.13`）会被最终值抵消。
- 用户公告接口必须根据当前权限重新渲染价格公告正文，禁止把包含无权限分组的完整 Markdown 返回给客户端；标记已读接口也必须重新鉴权，权限查询失败时不得退化为全部可见。
- 用户打开公告铃铛或页面从后台恢复时必须强制刷新公告，失去权限的公告应从列表和弹窗队列移除；同一公告因后来新增分组权限重新变为未读时，允许再次弹窗。

回归检查：

- 首次检查不发公告；同一周期多个分组变化合并为一条公告。
- 涨价和降价均显示方向、分组名称及精确倍率变化。
- 混合公开、手工专属、标签专属和订阅分组的同周期公告，对每个用户只显示其当前有权访问的明细。
- 先读公开明细再获得专属或标签权限时，历史公告重新未读；撤销权限后对应明细立即消失，重新获得权限时保留已有分组级已读记录。
- 后台服务清理时停止监测协程；不停止或重启主服务。
- 启用状态下倒计时从后端实际计划递减，刷新页面或客户端时钟有偏差时仍保持准确；修改周期后立即按新周期重排。
- 系统价格公告和管理员手工公告在管理端及用户端均显示正确的发布来源。

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
- `backend/internal/service/wire.go`
- `backend/cmd/server/wire_gen.go`
- `backend/internal/repository/account_repo.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/handler/dto/settings.go`
- `backend/internal/handler/admin/account_handler.go`
- `backend/internal/handler/admin/setting_handler_update.go`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/views/admin/AccountsView.vue`
- `frontend/src/views/user/ImagePlaygroundView.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/account/BulkEditAccountModal.vue`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/router/index.ts`
- `frontend/src/utils/featureFlags.ts`
- `frontend/src/stores/app.ts`
- `frontend/src/types/index.ts`
- `frontend/src/api/imagePlayground.ts`
- `frontend/src/api/admin/accounts.ts`

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
- 前端 typecheck 通过。
- 新增的 service 和 handler 回归测试通过。
