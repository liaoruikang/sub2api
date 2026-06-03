# 瑞科 API 文档站 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前 VitePress 模板站点替换为“瑞科 API”正式文档站，覆盖中转站使用、一键接入、CC Switch 接入和常见问题。

**Architecture:** 使用 VitePress 默认主题，保留轻量 Markdown 文档结构；首页负责导流，`docs/guide/relay-station.md` 负责账号到密钥流程，`docs/guide/integration.md` 负责客户端接入流程，`docs/guide/faq.md` 负责问题兜底。配置集中在 `docs/.vitepress/config.mts`，不新增业务代码，不修改后端、前端应用和 `install.ps1`。

**Tech Stack:** VitePress 2.0.0-alpha.17、Markdown、TypeScript VitePress config、pnpm。

---

## File Structure

- Modify: `docs/index.md` — 正式首页，替换默认 VitePress hero 和示例入口。
- Create: `docs/guide/relay-station.md` — 中转站使用教程：注册、计费/订阅、创建密钥、分组、用量。
- Create: `docs/guide/integration.md` — 接入教程：一键脚本、CC Switch、手动配置、排查。
- Create: `docs/guide/faq.md` — 常见问题页。
- Modify: `docs/.vitepress/config.mts` — 站点标题、描述、导航、侧栏、本地搜索、页脚。
- Remove: `docs/markdown-examples.md` — 删除 VitePress 模板示例页。
- Remove: `docs/api-examples.md` — 删除 VitePress 模板示例页。

---

### Task 1: Create the guide directory

**Files:**
- Create directory: `docs/guide/`

- [ ] **Step 1: Confirm the guide directory is not required to exist before creation**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/guide'"
```

Expected: `False` if this is a fresh docs site, or `True` if another worker already created it.

- [ ] **Step 2: Create the guide directory**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "New-Item -ItemType Directory -Force -Path 'docs/guide' | Out-Null"
```

Expected: command exits successfully with no output.

- [ ] **Step 3: Verify the directory exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/guide'"
```

Expected: `True`.

---

### Task 2: Write the relay station usage guide

**Files:**
- Create: `docs/guide/relay-station.md`

- [ ] **Step 1: Create `docs/guide/relay-station.md` with complete content**

Write this exact file content:

```md
# 中转站使用

本页带你从创建账号开始，完成 API Key 创建、分组选择和用量查看。完成后，你就可以进入 [接入使用](./integration.md)，把瑞科 API 接入 Claude Code、Codex CLI、Gemini CLI 或 CC Switch。

## 使用流程总览

1. 注册或登录账号。
2. 了解余额计费与订阅套餐的区别。
3. 创建 API Key。
4. 为 API Key 选择分组。
5. 复制密钥并接入客户端。
6. 在仪表盘、使用记录或 API Key 用量查询中查看消耗。

::: info 截图占位
这里放“登录后仪表盘总览”的截图。
:::

## 1. 创建账号

打开瑞科 API 站点后，点击注册入口创建账号。

常见注册流程：

1. 输入邮箱地址。
2. 设置密码。
3. 如果站点开启邮箱验证，点击发送验证码，并输入邮箱收到的 6 位验证码。
4. 如果站点开启邀请码注册，填写邀请码。
5. 确认信息后创建账号。
6. 注册成功后进入用户仪表盘。

::: info 截图占位
这里放“注册页面”的截图。
:::

::: info 截图占位
这里放“邮箱验证码页面”的截图。
:::

如果你已经有账号，直接进入登录页，输入邮箱和密码登录即可。部分站点也可能开启 Linux.do、钉钉、OIDC、微信等第三方登录方式；具体入口以登录页实际展示为准。

## 2. 计费与订阅的区别

瑞科 API 通常有两种使用方式：余额计费和订阅套餐。

### 余额计费

余额计费适合按需使用。你充值后获得账户余额，调用 API 时按照实际请求消耗扣除余额。

适合场景：

- 偶尔使用不同模型。
- 想先小额测试接入效果。
- 希望用多少付多少。
- 不确定未来使用量。

你可以在仪表盘、个人设置、使用记录和 API Key 用量查询中查看余额和消耗。

### 订阅套餐

订阅套餐适合固定使用某个分组或高频使用。你购买某个分组对应的套餐后，在套餐有效期内按套餐规则使用。

适合场景：

- 长期使用固定模型或固定平台。
- 希望用套餐限制控制预算。
- 团队或个人有稳定调用频率。

订阅的有效期、额度、限制和可用分组以站内套餐页面实际展示为准。

### 分组是什么

分组决定 API Key 使用哪类上游能力。创建 API Key 时选择分组，后续接入客户端时会根据分组生成对应配置。

常见分组理解方式：

| 分组类型 | 适合接入 |
| --- | --- |
| Claude / Anthropic 类分组 | Claude Code、Anthropic 兼容客户端 |
| OpenAI 类分组 | Codex CLI、OpenAI SDK、OpenAI 兼容客户端 |
| Gemini 类分组 | Gemini CLI、Gemini 兼容客户端 |
| Antigravity 分组 | 按客户端选择 Claude Code 或 Gemini CLI |

价格、倍率、有效期、并发、RPM、额度和套餐权益均以站内显示为准。

## 3. 创建 API Key

登录后进入“API 密钥”页面。

创建步骤：

1. 点击“创建密钥”。
2. 填写密钥名称，例如“Claude Code 日常使用”或“Codex 测试”。
3. 选择分组。
4. 按需设置额外限制。
5. 点击保存。
6. 在密钥列表中复制生成的 API Key。

::: info 截图占位
这里放“API 密钥列表”的截图。
:::

::: info 截图占位
这里放“创建 API Key 弹窗”的截图。
:::

### 可选配置说明

创建或编辑 API Key 时，你可能会看到这些选项：

| 配置 | 说明 |
| --- | --- |
| 自定义密钥 | 使用你指定的密钥字符串，通常需要满足长度和字符规则。 |
| 额度限制 | 限制这个 API Key 最多可消费的金额，`0` 通常表示不限制。 |
| 速率限制 | 限制 5 小时、1 天、7 天等时间窗口内的消费额度。 |
| 密钥有效期 | 设置 API Key 的过期时间。 |
| IP 白名单 | 设置后只允许指定 IP 或 CIDR 使用此密钥。 |
| IP 黑名单 | 设置后禁止指定 IP 或 CIDR 使用此密钥。 |

如果你只是第一次接入，可以先只填写名称并选择分组，其他限制保持默认。

## 4. 选择分组

API Key 必须选择正确分组，接入配置才能匹配你的客户端。

选择建议：

- 使用 Claude Code：选择 Claude / Anthropic 类分组。
- 使用 Codex CLI：选择 OpenAI 类分组。
- 使用 Gemini CLI：选择 Gemini 类分组。
- 使用 Antigravity：根据你实际使用的客户端选择 Claude Code 或 Gemini CLI。

如果 API Key 没有分组，密钥列表中的“使用密钥”或“导入到 CCS”可能无法给出正确配置。此时回到 API 密钥列表，点击分组列，为密钥分配分组。

::: info 截图占位
这里放“在密钥列表中更换分组”的截图。
:::

## 5. 查看用量

接入后，你可以通过以下位置查看使用情况。

### 仪表盘

仪表盘展示账户概览，例如余额、API Key 数量、今日请求、今日消费、Token 使用趋势和分组使用分布。

::: info 截图占位
这里放“仪表盘用量概览”的截图。
:::

### 使用记录

使用记录页适合排查具体请求。你可以查看请求模型、端点、Token、费用、计费模式、请求耗时、缓存读写和导出数据。

### API Key 用量查询

API Key 用量查询适合快速确认某个密钥是否可用，以及当前余额、订阅、额度和有效期状态。

### CC Switch 用量查询

通过 CC Switch 导入 Provider 时，瑞科 API 会提供 `/v1/usage` 用量查询能力。CC Switch 可用它显示密钥是否有效、余额或额度剩余情况。

## 下一步

完成账号、密钥和分组后，继续阅读 [接入使用](./integration.md)。推荐优先使用“一键接入”或“CC Switch”。
```

- [ ] **Step 2: Verify the file exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/guide/relay-station.md'"
```

Expected: `True`.

---

### Task 3: Write the integration guide

**Files:**
- Create: `docs/guide/integration.md`

- [ ] **Step 1: Create `docs/guide/integration.md` with complete content**

Write this exact file content:

```md
# 接入使用

本页介绍如何把瑞科 API 接入常见 AI 编程客户端。推荐优先使用“一键接入”或“CC Switch”。

## 接入前准备

开始前请确认：

1. 已创建瑞科 API 账号。
2. 已创建 API Key。
3. API Key 已选择正确分组。
4. 已复制 API Key。
5. 你知道要接入的客户端：Claude Code、VSCode Claude Code、Codex CLI、Gemini CLI 或 CC Switch。

默认 API Base URL：

```txt
https://api.ruikon.com
```

如果你的站点管理员提供了其他 API Base URL，请以站内“API 端点”或管理员说明为准。

## 方式 1：一键接入（推荐）

一键接入适合 Windows 用户。脚本会通过菜单帮你写入 Claude Code、VSCode Claude Code 或 Codex CLI 的配置。

脚本来源：项目根目录的 `install.ps1`。

### 运行脚本

1. 将 `install.ps1` 保存到本地。
2. 打开 PowerShell。
3. 进入脚本所在目录。
4. 如当前 PowerShell 阻止脚本运行，先临时允许当前会话执行脚本：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

5. 运行脚本：

```powershell
.\install.ps1
```

::: info 截图占位
这里放“PowerShell 运行 install.ps1 菜单”的截图。
:::

### 安装流程

脚本第一步会让你选择操作：

- 安装
- 卸载

选择“安装”后，继续选择安装项目：

- Codex CLI
- Claude Code
- VSCode Claude Code

如果选择 Codex CLI，脚本还会询问是否开启 WebSocket：

- 不开启
- 开启

随后输入你在瑞科 API 创建的 API Key。输入时会以 `*` 隐藏显示，按 Enter 确认。

### Codex CLI 会写入什么

选择 Codex CLI 后，脚本会创建或更新：

```txt
%USERPROFILE%\.codex\config.toml
%USERPROFILE%\.codex\auth.json
```

`config.toml` 中写入瑞科 API 的 OpenAI 兼容 Provider：

```toml
model_provider = "OpenAI"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://api.ruikon.com"
wire_api = "responses"
supports_websockets = false
requires_openai_auth = true

[features]
responses_websockets_v2 = false
goals = true
```

如果你在脚本中选择开启 WebSocket，`supports_websockets` 和 `responses_websockets_v2` 会写为 `true`。

`auth.json` 中写入：

```json
{
  "OPENAI_API_KEY": "你的 API Key"
}
```

### Claude Code 会写入什么

选择 Claude Code 后，脚本会写入用户环境变量：

```txt
ANTHROPIC_BASE_URL=https://api.ruikon.com
ANTHROPIC_AUTH_TOKEN=你的 API Key
CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
```

写入后建议重新打开终端，再启动 Claude Code。

### VSCode Claude Code 会写入什么

选择 VSCode Claude Code 后，脚本会创建或覆盖：

```txt
%USERPROFILE%\.claude\settings.json
```

内容包含：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://api.ruikon.com",
    "ANTHROPIC_AUTH_TOKEN": "你的 API Key",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}
```

写入后建议重启 VSCode 或 Claude Code 扩展会话。

### 卸载配置

重新运行脚本：

```powershell
.\install.ps1
```

选择“卸载”，再选择要卸载的项目：

- Codex CLI
- Claude Code
- VSCode Claude Code
- 全部

卸载行为：

| 目标 | 卸载内容 |
| --- | --- |
| Codex CLI | 删除 `%USERPROFILE%\.codex\config.toml` 和 `%USERPROFILE%\.codex\auth.json` |
| Claude Code | 删除用户环境变量 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` |
| VSCode Claude Code | 删除 `%USERPROFILE%\.claude\settings.json` |
| 全部 | 依次执行以上三类卸载 |

## 方式 2：CC Switch（推荐）

CC Switch 是一个 Provider 管理工具，可统一管理多个 AI 编程客户端的 Provider 配置。

官方资料：

- 官方网站：[https://ccswitch.io](https://ccswitch.io)
- GitHub 仓库：[farion1231/cc-switch](https://github.com/farion1231/cc-switch)
- 下载入口：[GitHub Releases](https://github.com/farion1231/cc-switch/releases)

CC Switch 官方用户手册显示，它支持 Claude Code、Claude Desktop、Codex、Gemini CLI、OpenCode、OpenClaw 和 Hermes。

### 安装 CC Switch

前往 [GitHub Releases](https://github.com/farion1231/cc-switch/releases)，按系统选择安装包。

常见选择：

- Windows：下载 `.msi` 或 Portable `.zip`。
- macOS：可使用 Homebrew 安装，或下载 `.dmg` / `.zip`。
- Linux：下载 `.deb`、`.rpm` 或 `.AppImage`。

::: info 截图占位
这里放“CC Switch 下载页面”的截图。
:::

### 从瑞科 API 导入 Provider

1. 打开瑞科 API。
2. 进入“API 密钥”页面。
3. 找到要使用的 API Key。
4. 确认该 API Key 已选择分组。
5. 点击“导入到 CCS”。
6. 浏览器提示打开 CC Switch 时，确认打开。
7. 在 CC Switch 中检查导入的 Provider。
8. 启用该 Provider。
9. 必要时重启终端或对应 CLI。

::: info 截图占位
这里放“API 密钥列表中的导入到 CCS 按钮”截图。
:::

::: info 截图占位
这里放“CC Switch 导入 Provider 后的界面”截图。
:::

### 导入规则

瑞科 API 会生成 `ccswitch://v1/import?...` Deep Link，将 Provider 导入 CC Switch。

根据 API Key 的分组平台，导入目标会有所不同：

| 分组平台 | 导入目标 |
| --- | --- |
| OpenAI | Codex |
| Gemini | Gemini |
| Claude / Anthropic | Claude |
| Antigravity | 弹出选择，可导入 Claude Code 或 Gemini CLI |

导入时会带上 API Base URL、API Key、Provider 名称和用量查询脚本。用量查询脚本会请求：

```txt
GET https://api.ruikon.com/v1/usage
Authorization: Bearer 你的 API Key
```

CC Switch 可据此显示密钥是否有效、余额或额度剩余情况。

### CC Switch 未弹出怎么办

如果点击“导入到 CCS”后没有打开 CC Switch，通常是以下原因：

1. CC Switch 尚未安装。
2. 浏览器没有注册 `ccswitch://` 协议处理程序。
3. 操作系统拦截了应用打开请求。

处理方式：

1. 先启动一次 CC Switch。
2. 重新点击“导入到 CCS”。
3. 如果仍失败，使用“使用密钥”里的手动配置方式。

## 手动配置参考

当一键脚本或 CC Switch 不可用时，可以手动配置。

### Claude Code

在当前终端会话中临时使用：

```powershell
$env:ANTHROPIC_BASE_URL = "https://api.ruikon.com"
$env:ANTHROPIC_AUTH_TOKEN = "你的 API Key"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = "1"
```

如果要长期生效，建议使用一键脚本写入用户环境变量，或按你的系统习惯配置环境变量。

### Codex CLI

确保配置目录存在：

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.codex" | Out-Null
```

配置文件路径：

```txt
%USERPROFILE%\.codex\config.toml
```

认证文件路径：

```txt
%USERPROFILE%\.codex\auth.json
```

推荐优先使用一键脚本自动写入，避免手动复制时格式出错。

## 接入后验证

接入后可用以下方式检查是否成功：

1. 重启终端、VSCode 或对应 CLI。
2. 发起一次简单请求。
3. 回到瑞科 API 的“使用记录”查看是否出现新请求。
4. 使用“API Key 用量查询”确认密钥状态、余额或订阅信息。

如果没有记录，检查 API Key 是否复制完整、分组是否正确、客户端是否读取了最新配置。
```

- [ ] **Step 2: Verify the file exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/guide/integration.md'"
```

Expected: `True`.

---

### Task 4: Write the FAQ page

**Files:**
- Create: `docs/guide/faq.md`

- [ ] **Step 1: Create `docs/guide/faq.md` with complete content**

Write this exact file content:

```md
# 常见问题

## 余额计费和订阅套餐应该怎么选？

如果你只是测试、偶尔使用或模型使用不固定，优先选择余额计费。余额计费按实际 API 消耗扣除余额，适合按需使用。

如果你长期使用固定分组，或希望用套餐规则控制预算，可以选择订阅套餐。订阅的可用分组、有效期、额度和限制以站内套餐页面展示为准。

## 为什么创建 API Key 时要选择分组？

分组决定 API Key 使用哪类上游能力，也会影响客户端接入配置。

例如：

- Claude / Anthropic 类分组适合 Claude Code。
- OpenAI 类分组适合 Codex CLI 或 OpenAI 兼容客户端。
- Gemini 类分组适合 Gemini CLI。
- Antigravity 分组需要按实际客户端选择 Claude Code 或 Gemini CLI。

如果 API Key 没有分组，“使用密钥”和“导入到 CCS”可能无法生成正确配置。

## API Key 可以设置额度限制吗？

可以。创建或编辑 API Key 时，可以按站内表单设置额度限制、速率限制和有效期。

常见限制包括：

- 总额度限制。
- 5 小时限额。
- 日限额。
- 7 天限额。
- 密钥过期时间。

`0` 通常表示不限制，具体以页面说明为准。

## API Key 可以限制 IP 吗？

可以。你可以设置 IP 白名单或 IP 黑名单。

- 设置白名单后，只允许白名单中的 IP 或 CIDR 使用此密钥。
- 设置黑名单后，黑名单中的 IP 或 CIDR 会被禁止使用此密钥。

如果你不确定自己的出口 IP 是否固定，第一次接入时建议先不配置 IP 限制。

## 一键脚本支持哪些客户端？

项目根目录的 `install.ps1` 支持：

- Codex CLI
- Claude Code
- VSCode Claude Code

脚本也支持卸载这些配置，或一次性卸载全部配置。

## 一键脚本会修改哪些配置？

根据选择的客户端不同，脚本会修改：

| 客户端 | 写入内容 |
| --- | --- |
| Codex CLI | `%USERPROFILE%\.codex\config.toml` 和 `%USERPROFILE%\.codex\auth.json` |
| Claude Code | 用户环境变量 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` |
| VSCode Claude Code | `%USERPROFILE%\.claude\settings.json` |

## CC Switch 是什么？

CC Switch 是一个 Provider 管理工具，可统一管理 Claude Code、Claude Desktop、Codex、Gemini CLI、OpenCode、OpenClaw 和 Hermes 等客户端的 Provider 配置。

官方资料：

- 官方网站：[https://ccswitch.io](https://ccswitch.io)
- GitHub 仓库：[farion1231/cc-switch](https://github.com/farion1231/cc-switch)
- 下载入口：[GitHub Releases](https://github.com/farion1231/cc-switch/releases)

## 点击“导入到 CCS”没有反应怎么办？

通常是 CC Switch 未安装或 `ccswitch://` 协议处理程序未注册。

处理步骤：

1. 安装并启动 CC Switch。
2. 回到瑞科 API 的 API 密钥页面。
3. 再次点击“导入到 CCS”。
4. 如果仍无法打开，使用“使用密钥”中的手动配置，或改用一键脚本。

## 为什么接入后没有请求记录？

依次检查：

1. API Key 是否复制完整。
2. API Key 是否仍处于启用状态。
3. API Key 是否已选择正确分组。
4. 客户端是否读取了最新配置。
5. 终端、VSCode 或客户端是否需要重启。
6. 请求是否实际发送到了 `https://api.ruikon.com`。

## API Key 泄露了怎么办？

立即进入 API 密钥页面，删除或禁用泄露的密钥，然后创建新密钥。

如果你在客户端中保存了旧密钥，记得同步更新客户端配置。

## 如何查看用量？

可以通过这些入口查看：

- 仪表盘：查看余额、请求数、Token 和分组使用概览。
- 使用记录：查看每次请求的模型、Token、费用和耗时。
- API Key 用量查询：查看某个密钥的余额、订阅、额度和有效期状态。
- CC Switch：通过导入时配置的 `/v1/usage` 查询密钥状态和余额或额度。

## 如何撤销客户端配置？

如果使用一键脚本接入，重新运行 `install.ps1`，选择“卸载”，再选择对应客户端或“全部”。

如果使用 CC Switch 接入，在 CC Switch 中禁用或删除对应 Provider。

如果手动配置环境变量或配置文件，请删除对应环境变量或配置项，并重启终端、VSCode 或客户端。
```

- [ ] **Step 2: Verify the file exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/guide/faq.md'"
```

Expected: `True`.

---

### Task 5: Replace the home page

**Files:**
- Modify: `docs/index.md`

- [ ] **Step 1: Replace `docs/index.md` with complete content**

Write this exact file content:

```md
---
layout: home

hero:
  name: 瑞科 API
  text: 一个密钥，接入多种 AI 服务
  tagline: 从账号注册、创建 API Key、选择分组到 Claude Code / Codex CLI / CC Switch 接入，一份文档带你完成完整流程。
  actions:
    - theme: brand
      text: 开始使用
      link: /guide/relay-station
    - theme: alt
      text: 接入教程
      link: /guide/integration

features:
  - title: 统一 API Key
    details: 创建一个 API Key，按分组接入 Claude、OpenAI、Gemini 等不同能力。
  - title: 余额计费与订阅
    details: 支持按实际消耗使用余额，也支持按分组购买订阅套餐。
  - title: 一键脚本接入
    details: 使用 PowerShell 脚本快速配置 Claude Code、VSCode Claude Code 或 Codex CLI。
  - title: CC Switch 导入
    details: 在 API 密钥列表中一键导入 Provider，适合管理多个 AI 编程客户端。
  - title: 分组管理
    details: 通过分组决定 API Key 的上游平台和客户端接入方式。
  - title: 用量可追踪
    details: 在仪表盘、使用记录和 API Key 用量查询中查看余额、额度、Token 和费用。
---

# 快速开始

瑞科 API 是一个 AI API 中转站。你可以通过它统一管理 API Key、分组、余额、订阅和用量，再把同一个入口接入 Claude Code、Codex CLI、Gemini CLI 或 CC Switch。

## 新用户路径

1. 阅读 [中转站使用](./guide/relay-station.md)，完成注册、计费理解、API Key 创建和分组选择。
2. 阅读 [接入使用](./guide/integration.md)，选择一键脚本或 CC Switch 接入客户端。
3. 接入后回到站内查看仪表盘、使用记录和 API Key 用量。

## 推荐接入方式

| 方式 | 适合用户 |
| --- | --- |
| 一键接入 | Windows 用户，希望快速配置 Claude Code、VSCode Claude Code 或 Codex CLI。 |
| CC Switch | 需要统一管理 Claude Code、Codex、Gemini CLI 等多个客户端 Provider 的用户。 |
| 手动配置 | 脚本不可用、CC Switch 未安装，或需要自行控制配置文件的用户。 |

## 下一步

- 第一次使用：从 [中转站使用](./guide/relay-station.md) 开始。
- 已经有 API Key：直接看 [接入使用](./guide/integration.md)。
- 遇到问题：查看 [常见问题](./guide/faq.md)。
```

- [ ] **Step 2: Verify the home page exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/index.md'"
```

Expected: `True`.

---

### Task 6: Update VitePress configuration

**Files:**
- Modify: `docs/.vitepress/config.mts`

- [ ] **Step 1: Replace `docs/.vitepress/config.mts` with complete content**

Write this exact file content:

```ts
import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  lang: 'zh-CN',
  title: '瑞科 API',
  description: '一个密钥，接入多种 AI 服务',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '中转站使用', link: '/guide/relay-station' },
      { text: '接入使用', link: '/guide/integration' },
      { text: '常见问题', link: '/guide/faq' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '中转站使用',
          items: [
            { text: '完整流程', link: '/guide/relay-station' }
          ]
        },
        {
          text: '接入使用',
          items: [
            { text: '接入教程', link: '/guide/integration' },
            { text: '常见问题', link: '/guide/faq' }
          ]
        }
      ]
    },

    search: {
      provider: 'local'
    },

    footer: {
      message: '瑞科 API 文档',
      copyright: 'Copyright © 2026 瑞科 API'
    }
  }
})
```

- [ ] **Step 2: Verify the config file exists**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/.vitepress/config.mts'"
```

Expected: `True`.

---

### Task 7: Remove default VitePress example pages

**Files:**
- Remove: `docs/markdown-examples.md`
- Remove: `docs/api-examples.md`

- [ ] **Step 1: Inspect the example pages before deletion**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Get-Content 'docs/markdown-examples.md' -TotalCount 5; ''; Get-Content 'docs/api-examples.md' -TotalCount 5"
```

Expected: output shows VitePress template example content, not user-authored product documentation.

- [ ] **Step 2: Delete the template example pages**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Remove-Item 'docs/markdown-examples.md','docs/api-examples.md' -Force"
```

Expected: command exits successfully with no output.

- [ ] **Step 3: Verify the files are gone**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Test-Path 'docs/markdown-examples.md'; Test-Path 'docs/api-examples.md'"
```

Expected:

```txt
False
False
```

---

### Task 8: Validate the documentation site

**Files:**
- Validate: `docs/index.md`
- Validate: `docs/guide/relay-station.md`
- Validate: `docs/guide/integration.md`
- Validate: `docs/guide/faq.md`
- Validate: `docs/.vitepress/config.mts`

- [ ] **Step 1: Run the VitePress build**

Run from repository root:

```bash
pnpm --dir docs docs:build
```

Expected: VitePress build completes successfully and writes the static site output under `docs/.vitepress/dist`.

If the shell cannot find `pnpm`, run this Windows fallback from repository root:

```bash
powershell.exe -NoProfile -Command "Set-Location 'docs'; pnpm docs:build"
```

Expected: same successful VitePress build.

- [ ] **Step 2: Verify no default example pages are referenced in authored Markdown**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Select-String -Path 'docs/**/*.md','docs/.vitepress/config.mts' -Pattern 'markdown-examples|api-examples|My Awesome Project|VitePress Site' -CaseSensitive:$false"
```

Expected: no matches.

- [ ] **Step 3: Verify expected guide links exist in config and home page**

Run from repository root:

```bash
powershell.exe -NoProfile -Command "Select-String -Path 'docs/index.md','docs/.vitepress/config.mts' -Pattern '/guide/relay-station|/guide/integration|/guide/faq'"
```

Expected: matches for `/guide/relay-station`, `/guide/integration`, and `/guide/faq`.

- [ ] **Step 4: Authorization-gated git checkpoint**

Only run this step if the user explicitly asks for a commit.

```bash
git add docs/index.md docs/guide/relay-station.md docs/guide/integration.md docs/guide/faq.md docs/.vitepress/config.mts docs/markdown-examples.md docs/api-examples.md docs/superpowers/specs/2026-06-02-ruikon-api-docs-design.md docs/superpowers/plans/2026-06-02-ruikon-api-docs.md
git commit -m "docs: add ruikon api usage guide"
```

Expected: git creates a documentation commit.

---

## Self-Review

### Spec coverage

- Site title and formal replacement: covered by Task 5 and Task 6.
- Account creation flow: covered by Task 2.
- Billing versus subscription explanation: covered by Task 2 and Task 4.
- API Key creation: covered by Task 2.
- Group selection: covered by Task 2 and Task 4.
- One-click script access based on `install.ps1`: covered by Task 3.
- CC Switch official sources and project deep link behavior: covered by Task 3 and Task 4.
- Screenshot slots: covered by Task 2 and Task 3 using VitePress info containers.
- Remove default examples: covered by Task 7.
- Build and link validation: covered by Task 8.

### Placeholder scan

The plan includes screenshot slots because the user requested empty image positions. Implementation steps include exact file content, exact paths, exact commands, and expected outputs.

### Type and path consistency

All guide links use `/guide/relay-station`, `/guide/integration`, and `/guide/faq` in VitePress config, and `./guide/*.md` or `./*.md` relative links in Markdown. All file paths match the repository layout under `docs/`.
