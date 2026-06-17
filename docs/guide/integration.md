# 接入使用

推荐优先使用“一键接入”或“CC Switch”。手动配置只作为兜底方案。

## 接入前准备 {#prepare}

开始前确认：

- 已注册账号并创建 API Key。
- API Key 已选择正确分组。
- 已复制 API Key。

默认 API Base URL：

```txt
https://api.ruikon.com
```

如站内显示了其他 API 端点，以站内为准。

## 方式 1：一键接入（推荐） {#one-click}

适合 Windows 用户。下载 <a href="/install.ps1" download>install.ps1</a> 后运行即可。

运行方式：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\install.ps1
```

按菜单选择：

1. 选择“安装”。
2. 选择客户端：`Codex CLI`、`Claude Code` 或 `VSCode Claude Code`。
3. 如果选择 Codex CLI，可选择是否开启 WebSocket。
4. 输入瑞科 API Key。
5. 按提示完成。

::: info 截图占位
这里放“install.ps1 菜单”的截图。
:::

脚本会自动写入：

- **Codex CLI**：`%USERPROFILE%\.codex\config.toml`、`%USERPROFILE%\.codex\auth.json`
- **Claude Code**：用户环境变量 `ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`
- **VSCode Claude Code**：`%USERPROFILE%\.claude\settings.json`

卸载时重新运行脚本，选择“卸载”，再选择对应客户端或“全部”。

## 方式 2：CC Switch（推荐） {#cc-switch}

CC Switch 用来统一管理 Claude Code、Codex、Gemini CLI 等客户端的 Provider。

官方入口：

- 官网：[https://ccswitch.io](https://ccswitch.io)
- 仓库：[farion1231/cc-switch](https://github.com/farion1231/cc-switch)
- 下载：[GitHub Releases](https://github.com/farion1231/cc-switch/releases)

使用方式：

1. 安装并启动 CC Switch。
2. 打开瑞科 API 的“API 密钥”页面。
3. 确认 API Key 已选择分组。
4. 点击“导入到 CCS”。
5. 浏览器提示打开 CC Switch 时确认。
6. 在 CC Switch 中启用导入的 Provider。

::: info 官方截图
以下截图来自 CC Switch 官方仓库，用于说明 Provider 管理界面。
:::

![CC Switch 主界面](/images/ccswitch-main.png)

![CC Switch 添加 Provider](/images/ccswitch-add-provider.png)

导入规则：

- OpenAI 分组 → Codex
- Gemini 分组 → Gemini
- Claude / Anthropic 分组 → Claude
- Antigravity 分组 → 选择 Claude Code 或 Gemini CLI

导入后，CC Switch 可通过 `/v1/usage` 查询密钥状态、余额或额度。

## 手动配置参考 {#manual-config}

一键脚本或 CC Switch 不可用时，可以手动配置。

Claude Code 临时环境变量：

```powershell
$env:ANTHROPIC_BASE_URL = "https://api.ruikon.com"
$env:ANTHROPIC_AUTH_TOKEN = "你的 API Key"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = "1"
```

Codex CLI 推荐优先使用一键脚本自动写入，避免手动配置出错。

## 接入后验证 {#verify}

1. 重启终端、VSCode 或对应客户端。
2. 发起一次简单请求。
3. 回到瑞科 API 的“使用记录”查看是否有新记录。
4. 用“API Key 用量查询”确认密钥状态。

如果没有记录，优先检查 API Key 是否完整、分组是否正确、客户端是否读取了最新配置。