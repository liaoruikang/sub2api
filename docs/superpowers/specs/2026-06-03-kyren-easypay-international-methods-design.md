# 启润 EasyPay 国际支付方式设计

日期：2026-06-03

## 目标

在现有 EasyPay/Epay 兼容支付接入基础上，支持启润支付商文档中的国际支付类型，让用户支付页可以独立展示并发起：

- Credit Card（`creditcard`）
- Crypto（`crypto`）
- PayNow（`paynow`）

这些方式与现有 `alipay`、`wxpay` 一样，通过 EasyPay 兼容接口提交订单、接收回调并完成订单履约。

## 已确认范围

采用“方案 A：扩展现有 EasyPay provider”。

本次不新增独立 `kyren` provider。后台仍配置 provider key 为 `easypay`，管理员将 EasyPay `apiBase` 指向启润 Epay 兼容接口地址，并在 EasyPay provider instance 中选择启用 `creditcard`、`crypto`、`paynow`。

用户支付页需要把这三种方式作为独立按钮展示，而不是合并成一个“国际支付”入口。

## 资料来源

启润 Epay 兼容迁移文档：

- `https://docs.kyren.top/zh/epay-migration/overview`
- `https://docs.kyren.top/zh/epay-migration/signature`
- `https://docs.kyren.top/zh/epay-migration/submit-php`
- `https://docs.kyren.top/zh/epay-migration/mapi-php`
- `https://docs.kyren.top/zh/epay-migration/api-php`
- `https://docs.kyren.top/zh/epay-migration/migration-checklist`

关键结论：

- 兼容接口使用原 Epay 的 `pid + sign` 鉴权，不使用 `x-api-key`。
- 支持的 `type` 包括 `alipay`、`wxpay`、`creditcard`、`crypto`、`paynow`。
- 浏览器跳转入口是 `/epay/submit.php`。
- 服务端直连创建支付入口是 `/epay/mapi.php`。
- 订单查询入口是 `/epay/api.php?act=order`。
- 兼容退款 `/epay/api.php?act=refund` 当前不支持，需要平台协助处理。
- notify 回调完成后商户应返回纯文本 `success`。

## 现有项目上下文

当前项目已有支付 provider 抽象、provider instance 配置、订单创建、回调验签、订单履约和前端支付方式展示流程。

相关文件：

- `backend/internal/payment/types.go`：后端支付类型常量和 provider interface。
- `backend/internal/payment/provider/easypay.go`：EasyPay provider 实现，包括 `submit.php`、`mapi.php`、`api.php`、webhook 验签和退款。
- `backend/internal/service/payment_visible_method_instances.go`：后端可见支付方式归一化与实例解析。
- `backend/internal/service/payment_config_limits.go`：支付方式限额/可用性聚合。
- `backend/internal/handler/payment_webhook_handler.go`：支付 webhook 入口和 ACK 行为。
- `frontend/src/types/payment.ts`：前端支付类型和 API 数据结构。
- `frontend/src/components/payment/providerConfig.ts`：后台 provider 支持类型、配置字段、webhook 路径和展示顺序。
- `frontend/src/components/payment/paymentFlow.ts`：前端可见支付方式归一化和支付启动决策。
- `frontend/src/components/payment/PaymentMethodSelector.vue`：用户侧支付方式展示。
- `frontend/src/views/user/PaymentView.vue`：用户支付页。

现状限制：

- 后端 EasyPay provider 的 `SupportedTypes()` 只包含 `alipay` 和 `wxpay`。
- 前端后台配置中 `easypay` 只暴露 `alipay` 和 `wxpay`。
- 用户侧可见支付方式当前不包含 `creditcard`、`crypto`、`paynow`。

## 后端设计

### 支付类型

在后端 payment type 常量中新增三个 EasyPay/Kyren 支付类型：

- `TypeCreditCard = "creditcard"`
- `TypeCrypto = "crypto"`
- `TypePayNow = "paynow"`

这些类型是一等可见支付方式，不是现有类型的 alias。

边界：

- `creditcard` 不映射到 Stripe 的 `card`。
- `crypto` 不新增链上钱包 provider。
- `paynow` 不新增 PayNow 专属 provider。
- 三者都由 `easypay` provider 处理。

### EasyPay provider 支持类型

扩展 EasyPay provider 的 `SupportedTypes()`，从：

```text
alipay, wxpay
```

扩展为：

```text
alipay, wxpay, creditcard, crypto, paynow
```

订单创建仍使用现有 EasyPay provider 方法，不新增 `kyren` provider 文件，不新增独立 webhook route。

### API 模式：`mapi.php`

默认或 `qrcode` 模式继续由后端 POST 到：

```text
/epay/mapi.php
```

请求使用 `application/x-www-form-urlencoded`，核心字段为：

- `pid`
- `type`
- `out_trade_no`
- `notify_url`
- `return_url`
- `name`
- `money`
- `clientip`
- `sign`
- `sign_type=MD5`

当用户选择 Credit Card、Crypto 或 PayNow 时，`type` 分别传：

- `creditcard`
- `crypto`
- `paynow`

响应处理继续优先使用现有字段：

- `trade_no`
- `payurl`
- `qrcode`

启润示例还可能返回 `img`。实现时可以把 `img` 作为 QR 图片候选字段接收，但不要求新增单独前端流程。前端仍依据 `pay_url` 和 `qr_code` 决定跳转或二维码展示。

### Popup / Redirect 模式：`submit.php`

`popup` 模式继续生成带签名的浏览器跳转 URL：

```text
/epay/submit.php?...signed params...
```

`creditcard`、`crypto`、`paynow` 与 `alipay`、`wxpay` 使用相同参数构造和签名流程。

### 可见方式与限额聚合

后端可见支付方式归一化需要新增：

- `creditcard -> creditcard`
- `crypto -> crypto`
- `paynow -> paynow`

这些不是别名，不能合并到 `stripe`、`easypay` 或其他 provider key。

限额/可用性接口应能返回新增 method，让前端只展示后台启用且后端判定可用的支付方式。

### Provider 选择

EasyPay provider instance 支持 `creditcard`、`crypto`、`paynow` 后，订单服务应能为这三个 `payment_type` 选择对应的 EasyPay provider instance。

如果多个 provider instance 支持同一 method，继续沿用现有实例选择、快照和负载均衡策略。

### Webhook / notify

Webhook 路径保持：

```text
/api/v1/payment/webhook/easypay
```

启润 notify 支持 GET 通知；现有 EasyPay webhook handler 已支持从 query string 或 raw body 读取回调内容，因此无需新增 route。

验签规则保持 Epay 兼容规则：

1. 排除 `sign` 和 `sign_type`。
2. 排除空值参数。
3. 参数名按原始 key 的 ASCII 升序排序。
4. 拼接为 `k=v&k2=v2`。
5. 直接追加商户密钥，不加 `&key=`。
6. 计算 MD5，输出小写 hex。

只有 `trade_status=TRADE_SUCCESS` 时才视为支付成功。订单履约继续依赖现有幂等 lifecycle，按 `out_trade_no` / 内部订单号定位订单。

### 查询与退款

订单查询继续使用：

```text
/epay/api.php?act=order
```

参数仍为 `pid`、`key`，并通过 `out_trade_no` 或 `trade_no` 查询。

退款不作为本次目标。启润文档说明兼容退款当前不支持，因此：

- 不新增启润自动退款能力。
- 不把退款请求失败伪装成成功。
- 保留现有 EasyPay refund 代码，避免破坏其他 EasyPay 商户。
- 如果启润返回退款不支持，系统应如实显示退款失败或不可自动退款。

## 前端设计

### 支付类型与可见方式

前端支付类型 union、可见方式归一化和支付 method map 需要新增：

- `creditcard`
- `crypto`
- `paynow`

它们作为独立 visible method 展示，不映射到 `stripe` 或 `airwallex`。

### 用户支付页展示

用户支付页新增三个独立按钮：

| type | 用户侧显示 | 后台显示 |
| --- | --- | --- |
| `creditcard` | Credit Card | Credit Card |
| `crypto` | Crypto | Crypto |
| `paynow` | PayNow | PayNow |

推荐排序：

1. Credit Card
2. Crypto
3. PayNow
4. Alipay
5. WeChat Pay
6. Stripe
7. Airwallex

点击按钮后，创建订单 payload 直接传对应 `payment_type`：

```json
{ "payment_type": "creditcard" }
```

```json
{ "payment_type": "crypto" }
```

```json
{ "payment_type": "paynow" }
```

### 支付启动行为

不为 `creditcard`、`crypto`、`paynow` 新增单独启动流程。继续复用现有通用判断：

- 后端返回 `pay_url`：走跳转或弹窗等待支付。
- 后端返回 `qr_code`：展示二维码并轮询订单状态。
- 后端返回 Stripe/Airwallex 的 `client_secret`、`intent_id`：仍只由对应 provider 流程使用。
- 没有可用支付入口：走现有未处理/错误提示路径。

### 后台 EasyPay provider 配置

后台 `easypay` provider 的可选 supported types 从：

```text
alipay, wxpay
```

扩展为：

```text
alipay, wxpay, creditcard, crypto, paynow
```

管理员可以在同一个 EasyPay provider instance 中启用任意组合，例如：

- 只启用 `creditcard`
- 启用 `crypto,paynow`
- 同时启用 `alipay,wxpay,creditcard,crypto,paynow`

不新增启润专属配置项。仍使用现有 EasyPay 字段：

- `pid`
- `pkey`
- `apiBase`
- `notifyUrl`
- `returnUrl`
- 可选 channel/cid 字段

如果启润不需要 `cidAlipay` 或 `cidWxpay`，管理员可以留空。

## 安全设计

本次涉及支付和第三方 API，必须保留以下安全边界：

- 所有 merchant secret / `pkey` 仍作为敏感配置处理，不在前端、日志或错误响应中泄露。
- EasyPay/Kyren webhook 必须验签，不能仅凭 `trade_status` 或订单号入账。
- 验签必须排除 `sign`、`sign_type` 和空值，按 ASCII 排序后直接拼接 merchant secret 做 MD5 小写 hex。
- 支付成功履约必须幂等，重复 notify 不能重复充值或重复开通订阅。
- 入账权益以本系统订单记录为准，不允许回调金额直接决定充值额度或订阅权益。
- 错误信息面向用户保持通用，不返回签名串、密钥、完整上游错误中的敏感内容。
- audit log 可以记录订单号、上游 trade_no、支付 type、状态、provider instance，但不能记录 `pkey`。

## 验收标准

### 后端

- EasyPay provider `SupportedTypes()` 包含 `alipay`、`wxpay`、`creditcard`、`crypto`、`paynow`。
- 创建订单时，`creditcard`、`crypto`、`paynow` 会被原样传给 EasyPay/Kyren 的 `type` 参数。
- EasyPay 签名逻辑保持兼容启润 Epay 文档。
- 新增 method 可以参与后端可见支付方式聚合和实例选择。
- EasyPay webhook 对新增 method 的回调可以验签并触发现有订单履约。
- Stripe 的 `card` 与 EasyPay/Kyren 的 `creditcard` 不互相混淆。
- 启润退款不支持时，系统不报告虚假的自动退款成功。

### 前端

- 后台 EasyPay provider 可选择 Credit Card、Crypto、PayNow。
- 用户支付页能显示 Credit Card、Crypto、PayNow 三个独立按钮。
- 点击三个按钮时，创建订单 payload 的 `payment_type` 分别是 `creditcard`、`crypto`、`paynow`。
- 收到 `pay_url` 时能正常跳转或弹窗等待支付。
- 收到 `qr_code` 时能正常展示二维码并等待支付。
- 现有 Alipay、WeChat Pay、Stripe、Airwallex 支付方式不回归。

## 测试计划

后端建议覆盖：

- EasyPay `SupportedTypes()` 单元测试。
- EasyPay `mapi.php` 创建订单参数与签名测试，覆盖 `creditcard`、`crypto`、`paynow`。
- EasyPay notify 验签与新增 type 回调解析测试。
- 可见支付方式聚合/限额测试，覆盖新增 method。
- provider instance 选择测试，确保新增 method 能路由到 EasyPay provider instance。

前端建议覆盖：

- `providerConfig` 中 EasyPay 支持类型包含新 method。
- `paymentFlow` 可见 method 归一化覆盖 `creditcard`、`crypto`、`paynow`。
- 用户支付页/selector 能展示新增按钮。
- 点击新增按钮时 create-order payload 正确。
- 支付启动决策对新增 method 的 `pay_url` 和 `qr_code` 响应保持通用行为。

验证偏好：前端按项目偏好执行 lint-only 验证，不强制运行 frontend build。

## 非目标

本次不做：

- 新增独立 `kyren` provider。
- 新增加密货币链上钱包验证、钱包签名或链上交易确认。
- 新增 PayNow 本地银行转账协议实现。
- 新增启润自动退款能力。
- 重构整个支付架构。
- 修改 Stripe Payment Element 的 `card` / `link` 逻辑。
- 改动用户订阅、余额充值、订单履约的核心权益计算规则。
