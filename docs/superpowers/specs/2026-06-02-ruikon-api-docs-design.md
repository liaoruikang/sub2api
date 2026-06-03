# 瑞科 API 文档站设计

日期：2026-06-02

## 目标

将当前 VitePress 模板站点直接替换为正式的“瑞科 API”文档站，面向第一次使用中转站的用户，提供从创建账号到接入客户端的完整路径。

文档要覆盖：

1. 中转站使用：创建账号、计费与订阅区别、创建 API Key、选择分组、查看用量。
2. 接入使用：一键脚本接入、CC Switch 接入、必要的手动配置兜底。
3. 图片位置预留：使用明确截图占位，后续由站点维护者自行填补截图。

## 已确认范围

采用“方案 A：单页快速上手 + 两个专题页”。

站点将直接替换默认模板内容，不再在导航或侧栏展示 VitePress 示例页。

## 站点信息

- 站点名称：瑞科 API
- VitePress 标题：瑞科 API
- 描述方向：一个密钥，接入多种 AI 服务；统一中转、统一密钥、按量或订阅使用。
- 首页 CTA：
  - 开始使用 → `/guide/relay-station`
  - 接入教程 → `/guide/integration`

## 页面结构

### `docs/index.md`

正式首页，替换默认 VitePress 首页。

内容：

- Hero：瑞科 API / 一个密钥，接入多种 AI 服务。
- 功能卡片：统一 API Key、按量计费、订阅套餐、CC Switch 导入、一键脚本接入、用量查询。
- 快速路径：
  1. 注册账号
  2. 创建 API Key
  3. 选择分组
  4. 接入 Claude Code / Codex CLI / Gemini CLI / CC Switch

### `docs/guide/relay-station.md`

中转站使用流程页。

章节：

1. 创建账号
   - 打开站点。
   - 注册邮箱和密码。
   - 如站点开启邮箱验证，输入 6 位验证码。
   - 如站点开启邀请码，填写邀请码。
   - 注册成功后进入仪表盘。
   - 截图占位：注册页、邮箱验证页、仪表盘。

2. 计费与订阅的区别
   - 余额/计费：按实际 API 消耗扣除余额，适合临时或按需使用。
   - 订阅：购买某个分组对应的套餐，在有效期内按套餐规则使用，适合固定模型或高频使用。
   - 分组：决定 API Key 走哪类上游能力，例如 Claude / OpenAI / Gemini / Antigravity。
   - 价格、倍率、有效期、限制以站内显示为准。

3. 创建 API Key
   - 进入 API 密钥页面。
   - 点击创建密钥。
   - 填写名称。
   - 选择分组。
   - 可选：自定义密钥、额度限制、速率限制、有效期、IP 白名单/黑名单。
   - 保存并复制密钥。
   - 截图占位：密钥列表、创建弹窗、复制按钮。

4. 选择分组
   - Claude / Anthropic 分组适合 Claude Code。
   - OpenAI 分组适合 Codex CLI 或 OpenAI SDK。
   - Gemini 分组适合 Gemini CLI。
   - Antigravity 分组按客户端选择 Claude Code 或 Gemini CLI。
   - 如果密钥没有分组，先分配分组，否则使用配置和导入功能可能不可用。

5. 查看用量
   - 仪表盘查看概览。
   - 使用记录查看请求、模型、Token、费用、计费模式。
   - API Key 用量查询查看余额、订阅、额度状态。
   - CC Switch 可通过 `/v1/usage` 查询余额或额度状态。

### `docs/guide/integration.md`

接入使用教程页。

章节：

1. 接入前准备
   - 已注册账号。
   - 已创建 API Key。
   - API Key 已选择正确分组。
   - 确认站点 API Base URL：`https://api.ruikon.com`。

2. 方式 1：一键接入（推荐）
   - 基于项目根目录 `install.ps1` 的真实行为编写 Windows PowerShell 教程。
   - 支持安装和卸载。
   - 安装目标：Codex CLI、Claude Code、VSCode Claude Code。
   - Codex CLI 可选是否开启 WebSocket。
   - 输入 API Key 后自动写配置。

   配置影响：

   - Codex CLI：
     - `%USERPROFILE%\.codex\config.toml`
     - `%USERPROFILE%\.codex\auth.json`
     - `base_url = "https://api.ruikon.com"`
     - `model = "gpt-5.5"`
   - Claude Code：
     - 用户环境变量 `ANTHROPIC_BASE_URL=https://api.ruikon.com`
     - 用户环境变量 `ANTHROPIC_AUTH_TOKEN=<你的 API Key>`
     - 用户环境变量 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
   - VSCode Claude Code：
     - `%USERPROFILE%\.claude\settings.json`
     - 写入 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`、`CLAUDE_CODE_ATTRIBUTION_HEADER`。

3. 方式 2：CC Switch（推荐）
   - CC Switch 是 Provider 管理工具。
   - 官方站点：`https://ccswitch.io`
   - 官方仓库：`https://github.com/farion1231/cc-switch`
   - 下载入口：`https://github.com/farion1231/cc-switch/releases`
   - 支持 Claude Code、Claude Desktop、Codex、Gemini CLI、OpenCode、OpenClaw、Hermes。
   - 在瑞科 API 的 API 密钥列表中点击“导入到 CCS”。
   - 浏览器提示打开 CC Switch 时确认。
   - 项目会生成 `ccswitch://v1/import?...` Deep Link 导入 Provider。
   - 根据分组平台导入到不同客户端：
     - OpenAI 分组 → Codex。
     - Gemini 分组 → Gemini。
     - Anthropic / Claude 分组 → Claude。
     - Antigravity 分组 → 用户选择 Claude Code 或 Gemini CLI。
   - 导入后在 CC Switch 中启用 Provider；必要时重启终端或对应 CLI。

4. 手动配置参考
   - 作为脚本失败或 CC Switch 未安装时的兜底。
   - 简要提供 Claude Code、Codex CLI 的关键变量或配置文件位置。
   - 不展开过多内容，避免和一键脚本重复。

5. 常见问题
   - 找不到 CC Switch：确认安装并注册协议处理程序。
   - 没有分组：先回 API 密钥页面选择分组。
   - 配置不生效：重启终端、VSCode 或对应 CLI。
   - API Key 泄露：立即删除旧密钥并重新创建。

### `docs/guide/faq.md`

常见问题页。

内容：

- 余额和订阅如何选择？
- API Key 能否设置额度和 IP 限制？
- 为什么创建密钥时必须选择分组？
- 一键脚本支持哪些客户端？
- CC Switch 导入失败怎么办？
- 如何查看用量？
- 如何撤销或重置接入配置？

## VitePress 配置

修改 `docs/.vitepress/config.mts`：

- `title` 改为 `瑞科 API`。
- `description` 改为中文产品描述。
- `nav` 改为：首页、中转站使用、接入使用、常见问题。
- `sidebar` 改为两组：中转站使用、接入使用。
- 移除默认示例页入口。
- `socialLinks` 如无确定链接，先移除默认 Vue/VitePress GitHub 链接，避免误导。

## 默认示例页处理

删除默认模板示例页：

- `docs/markdown-examples.md`
- `docs/api-examples.md`

原因：正式站点不应暴露 VitePress 模板示例内容，且用户已确认直接替换成正式站点。

## 资料来源和约束

- `install.ps1`：一键接入教程必须以脚本真实行为为准。
- `frontend/src/utils/ccswitchImport.ts` 和 `frontend/src/views/user/KeysView.vue`：CC Switch 导入教程必须以项目深链导入逻辑为准。
- `backend/internal/handler/gateway_handler.go`：`/v1/usage` 可用于 CC Switch 用量查询说明。
- CC Switch 官方资料：
  - 官方站点：`https://ccswitch.io`
  - 官方仓库：`https://github.com/farion1231/cc-switch`
  - 用户手册页面成功抓取，确认支持客户端、Provider 管理、Deep Link 导入和下载入口。
  - README 直接抓取返回 403，Provider 文档页面请求超时；文档中不编造未确认字段。

## 验证方式

实施后执行：

- 优先检查文档结构和链接。
- 可运行 `pnpm docs:build` 验证 VitePress 构建。
- 如果只需要快速检查，可至少确认：
  - 首页可进入两个专题页。
  - 侧栏不再出现默认示例页。
  - 所有内部链接指向存在页面。

## 非目标

本次不修改后端、前端业务功能或 `install.ps1` 脚本行为。

本次不新增真实截图，只放截图占位。

本次不承诺具体价格、倍率、套餐权益或模型可用性；这些内容以站内配置和实际展示为准。
