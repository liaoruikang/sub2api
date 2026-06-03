# 常见问题

## 余额计费和订阅套餐怎么选？

- 临时使用、测试、模型不固定：选余额计费。
- 固定分组、高频使用、想控制套餐周期：选订阅套餐。

具体价格、额度和有效期以站内显示为准。

## 为什么要选择分组？

分组决定 API Key 接入哪类能力，也影响客户端配置：

- Claude / Anthropic：Claude Code
- OpenAI：Codex CLI 或 OpenAI 兼容客户端
- Gemini：Gemini CLI
- Antigravity：按客户端选择 Claude Code 或 Gemini CLI

没有分组时，“使用密钥”和“导入到 CCS”可能无法生成正确配置。

## API Key 可以限制额度或 IP 吗？

可以。创建或编辑 API Key 时可按需设置：

- 总额度或时间窗口限额
- 有效期
- IP 白名单 / 黑名单

第一次接入建议先保持默认，确认可用后再加限制。

## 一键脚本支持哪些客户端？

<a href="/install.ps1" download>install.ps1</a> 支持：

- Codex CLI
- Claude Code
- VSCode Claude Code

也支持卸载单个客户端配置或全部配置。

## CC Switch 是什么？

CC Switch 是 Provider 管理工具，可统一管理 Claude Code、Codex、Gemini CLI 等客户端配置。

- 官网：[https://ccswitch.io](https://ccswitch.io)
- 仓库：[farion1231/cc-switch](https://github.com/farion1231/cc-switch)
- 下载：[GitHub Releases](https://github.com/farion1231/cc-switch/releases)

## 点击“导入到 CCS”没有反应怎么办？

通常是 CC Switch 未安装或 `ccswitch://` 协议未注册。

处理方式：

1. 安装并启动 CC Switch。
2. 回到 API 密钥页面重新点击“导入到 CCS”。
3. 仍失败时，改用一键脚本或手动配置。

## 接入后没有请求记录怎么办？

优先检查：

- API Key 是否复制完整。
- API Key 是否启用。
- 分组是否正确。
- 客户端是否重启并读取新配置。
- 请求是否发往 `https://api.ruikon.com`。

## API Key 泄露了怎么办？

立即禁用或删除旧密钥，重新创建 API Key，并更新客户端配置。

## 如何查看用量？

常用入口：

- 仪表盘
- 使用记录
- API Key 用量查询
- CC Switch 的 `/v1/usage` 状态显示

## 如何撤销客户端配置？

- 一键脚本：重新运行 <a href="/install.ps1" download>install.ps1</a>，选择“卸载”。
- CC Switch：禁用或删除对应 Provider。
- 手动配置：删除环境变量或配置文件，并重启客户端。