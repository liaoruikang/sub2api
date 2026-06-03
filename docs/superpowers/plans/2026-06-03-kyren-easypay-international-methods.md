# Kyren EasyPay International Methods Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing EasyPay provider so Kyren/Epay-compatible `creditcard`, `crypto`, and `paynow` payment types are configurable in admin and visible as independent user payment buttons.

**Architecture:** Keep the existing `easypay` provider as the only Kyren integration surface. Add canonical payment type constants, expand EasyPay supported types, route the new methods explicitly to EasyPay to avoid Stripe/official-provider confusion, and reuse the existing EasyPay `mapi.php` / `submit.php` / webhook flows. Frontend changes are limited to payment type lists, visible-method normalization, i18n labels, and selector display; no new checkout flow is introduced.

**Tech Stack:** Go payment/service layers with Ent-backed provider instances, Vue 3 + TypeScript frontend, Vitest component/unit tests, Go `testing` package, existing EasyPay MD5 signing helpers.

---

## File Structure

- Modify: `backend/internal/payment/types.go` — add `creditcard`, `crypto`, and `paynow` payment type constants.
- Modify: `backend/internal/payment/provider/easypay.go` — include the new types in `SupportedTypes()` and use Kyren `img` as a QR fallback when `qrcode` is absent.
- Modify: `backend/internal/payment/provider/easypay_sign_test.go` — add a provider capability test for EasyPay supported types.
- Create: `backend/internal/payment/provider/easypay_create_test.go` — test Kyren `mapi.php` create response handling for an international method.
- Modify: `backend/internal/service/payment_visible_method_instances.go` — treat `creditcard`, `crypto`, and `paynow` as EasyPay-visible methods.
- Modify: `backend/internal/service/payment_resume_service.go` — route the new visible methods to `easypay` before falling through to the generic load balancer.
- Create: `backend/internal/service/payment_visible_method_instances_test.go` — test visible method enumeration and EasyPay-only routing.
- Modify: `backend/internal/service/payment_config_limits_test.go` — verify available method limits expose the new EasyPay methods independently.
- Modify: `frontend/src/types/payment.ts` — widen the frontend `PaymentType` union.
- Modify: `frontend/src/components/payment/providerConfig.ts` — add EasyPay supported types and user-facing method order.
- Modify: `frontend/src/components/payment/paymentFlow.ts` — normalize the new payment methods as first-class visible methods.
- Modify: `frontend/src/components/payment/PaymentMethodSelector.vue` — use the EasyPay icon for the new methods instead of falling back to Alipay.
- Create: `frontend/src/components/payment/__tests__/PaymentMethodSelector.spec.ts` — verify independent button labels.
- Modify: `frontend/src/components/payment/__tests__/providerConfig.spec.ts` — verify EasyPay admin supported types and ordering.
- Modify: `frontend/src/components/payment/__tests__/paymentFlow.spec.ts` — verify visible method normalization, payloads, and generic launch behavior.
- Modify: `frontend/src/i18n/locales/en.ts` — add English labels for Credit Card, Crypto, and PayNow.
- Modify: `frontend/src/i18n/locales/zh.ts` — add Chinese labels for the new methods.

---

### Task 1: Add backend payment constants and EasyPay capability

**Files:**
- Modify: `backend/internal/payment/provider/easypay_sign_test.go`
- Modify: `backend/internal/payment/types.go`
- Modify: `backend/internal/payment/provider/easypay.go`

- [ ] **Step 1: Write the failing EasyPay supported-types test**

In `backend/internal/payment/provider/easypay_sign_test.go`, replace the import block with:

```go
import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)
```

Append this test and helper to the same file:

```go
func TestEasyPaySupportedTypesIncludesKyrenInternationalMethods(t *testing.T) {
	t.Parallel()

	provider := &EasyPay{}
	got := provider.SupportedTypes()
	want := []payment.PaymentType{
		payment.TypeAlipay,
		payment.TypeWxpay,
		payment.TypeCreditCard,
		payment.TypeCrypto,
		payment.TypePayNow,
	}

	assertPaymentTypesEqual(t, got, want)
}

func assertPaymentTypesEqual(t *testing.T, got, want []payment.PaymentType) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("SupportedTypes len = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SupportedTypes[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}
```

- [ ] **Step 2: Run the backend provider test and verify RED**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run TestEasyPaySupportedTypesIncludesKyrenInternationalMethods -count=1
```

Expected: FAIL. Before implementation, the package does not define `payment.TypeCreditCard`, `payment.TypeCrypto`, or `payment.TypePayNow`, or EasyPay does not return those types.

- [ ] **Step 3: Add canonical backend payment type constants**

In `backend/internal/payment/types.go`, replace the supported payment type constants block with this block:

```go
// Supported payment type constants.
const (
	TypeAlipay       PaymentType = "alipay"
	TypeWxpay        PaymentType = "wxpay"
	TypeAlipayDirect PaymentType = "alipay_direct"
	TypeWxpayDirect  PaymentType = "wxpay_direct"
	TypeStripe       PaymentType = "stripe"
	TypeCard         PaymentType = "card"
	TypeLink         PaymentType = "link"
	TypeEasyPay      PaymentType = "easypay"
	TypeCreditCard   PaymentType = "creditcard"
	TypeCrypto       PaymentType = "crypto"
	TypePayNow       PaymentType = "paynow"
	TypeAirwallex    PaymentType = "airwallex"
)
```

Do not map `TypeCreditCard` to Stripe `TypeCard`; they are separate payment methods.

- [ ] **Step 4: Expand EasyPay provider supported types**

In `backend/internal/payment/provider/easypay.go`, replace `SupportedTypes()` with:

```go
func (e *EasyPay) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{
		payment.TypeAlipay,
		payment.TypeWxpay,
		payment.TypeCreditCard,
		payment.TypeCrypto,
		payment.TypePayNow,
	}
}
```

- [ ] **Step 5: Run the provider test and verify GREEN**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run TestEasyPaySupportedTypesIncludesKyrenInternationalMethods -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

```bash
git add backend/internal/payment/types.go backend/internal/payment/provider/easypay.go backend/internal/payment/provider/easypay_sign_test.go
git commit -m "feat(payment): add EasyPay international payment types

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Preserve Kyren `mapi.php` response compatibility

**Files:**
- Create: `backend/internal/payment/provider/easypay_create_test.go`
- Modify: `backend/internal/payment/provider/easypay.go`

- [ ] **Step 1: Write the failing Kyren `img` QR fallback test**

Create `backend/internal/payment/provider/easypay_create_test.go` with this content:

```go
package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEasyPayCreateAPIPaymentUsesImgFallbackForKyrenQRCode(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"trade_no":"K202606030001","out_trade_no":"sub2_img","payurl":"https://api.kyren.top/epay/redirect/order_abc","img":"https://api.kyren.top/epay/qr/order_abc.png"}`))
	}))
	defer server.Close()

	provider := newTestEasyPay(t, server.URL)
	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "sub2_img",
		Amount:      "9.99",
		PaymentType: payment.TypeCreditCard,
		Subject:     "AI credits",
		ClientIP:    "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("CreatePayment returned error: %v", err)
	}
	if gotPath != "/mapi.php" {
		t.Fatalf("create path = %q, want /mapi.php", gotPath)
	}
	if got := gotForm.Get("type"); got != payment.TypeCreditCard {
		t.Fatalf("form[type] = %q, want %q (form=%v)", got, payment.TypeCreditCard, gotForm)
	}
	if got := gotForm.Get("sign_type"); got != signTypeMD5 {
		t.Fatalf("form[sign_type] = %q, want %q", got, signTypeMD5)
	}
	if got := gotForm.Get("sign"); got == "" {
		t.Fatalf("form[sign] is empty (form=%v)", gotForm)
	}
	if resp.TradeNo != "K202606030001" {
		t.Fatalf("TradeNo = %q, want K202606030001", resp.TradeNo)
	}
	if resp.QRCode != "https://api.kyren.top/epay/qr/order_abc.png" {
		t.Fatalf("QRCode = %q, want Kyren img fallback", resp.QRCode)
	}
}
```

- [ ] **Step 2: Run the create-payment test and verify RED**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run TestEasyPayCreateAPIPaymentUsesImgFallbackForKyrenQRCode -count=1
```

Expected: FAIL with `QRCode = "", want Kyren img fallback` because the current response parser reads `qrcode` but not Kyren's `img` field.

- [ ] **Step 3: Implement `img` as a QR fallback**

In `backend/internal/payment/provider/easypay.go`, update the response struct inside `createAPIPayment()` to include `Img`:

```go
	var resp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		TradeNo string `json:"trade_no"`
		PayURL  string `json:"payurl"`
		PayURL2 string `json:"payurl2"` // H5 mobile payment URL
		QRCode  string `json:"qrcode"`
		Img     string `json:"img"`
	}
```

Then replace the return section at the end of `createAPIPayment()` with:

```go
	payURL := resp.PayURL
	if req.IsMobile && resp.PayURL2 != "" {
		payURL = resp.PayURL2
	}
	qrCode := resp.QRCode
	if qrCode == "" {
		qrCode = resp.Img
	}
	return &payment.CreatePaymentResponse{TradeNo: resp.TradeNo, PayURL: payURL, QRCode: qrCode}, nil
```

- [ ] **Step 4: Run the create-payment test and verify GREEN**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run TestEasyPayCreateAPIPaymentUsesImgFallbackForKyrenQRCode -count=1
```

Expected: PASS.

- [ ] **Step 5: Run all EasyPay provider tests**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run 'TestEasyPay|TestNormalizeEasyPayAPIBase' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 2**

```bash
git add backend/internal/payment/provider/easypay.go backend/internal/payment/provider/easypay_create_test.go
git commit -m "fix(payment): accept Kyren EasyPay QR image responses

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Route international methods explicitly to EasyPay

**Files:**
- Create: `backend/internal/service/payment_visible_method_instances_test.go`
- Modify: `backend/internal/service/payment_visible_method_instances.go`
- Modify: `backend/internal/service/payment_resume_service.go`

- [ ] **Step 1: Write failing visible-method and load-balancer routing tests**

Create `backend/internal/service/payment_visible_method_instances_test.go` with this content:

```go
package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEnabledVisibleMethodsForProviderIncludesKyrenInternationalMethods(t *testing.T) {
	t.Parallel()

	got := enabledVisibleMethodsForProvider(
		payment.TypeEasyPay,
		"alipay,wxpay,creditcard,crypto,paynow",
	)
	want := []string{
		payment.TypeCreditCard,
		payment.TypeCrypto,
		payment.TypePayNow,
		payment.TypeAlipay,
		payment.TypeWxpay,
	}

	if len(got) != len(want) {
		t.Fatalf("enabled methods len = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("enabled methods[%d] = %q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestVisibleMethodLoadBalancerRoutesKyrenMethodsToEasyPay(t *testing.T) {
	t.Parallel()

	for _, method := range []payment.PaymentType{payment.TypeCreditCard, payment.TypeCrypto, payment.TypePayNow} {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingLoadBalancer{}
			lb := &visibleMethodLoadBalancer{
				inner:         recorder,
				configService: &PaymentConfigService{},
			}

			_, err := lb.SelectInstance(
				context.Background(),
				"",
				method,
				payment.Strategy(payment.DefaultLoadBalanceStrategy),
				9.99,
			)
			if err != nil {
				t.Fatalf("SelectInstance returned error: %v", err)
			}
			if recorder.providerKey != payment.TypeEasyPay {
				t.Fatalf("providerKey = %q, want %q", recorder.providerKey, payment.TypeEasyPay)
			}
			if recorder.paymentType != method {
				t.Fatalf("paymentType = %q, want %q", recorder.paymentType, method)
			}
		})
	}
}

type recordingLoadBalancer struct {
	providerKey string
	paymentType payment.PaymentType
}

func (r *recordingLoadBalancer) GetInstanceConfig(_ context.Context, _ int64) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *recordingLoadBalancer) SelectInstance(
	_ context.Context,
	providerKey string,
	paymentType payment.PaymentType,
	_ payment.Strategy,
	_ float64,
) (*payment.InstanceSelection, error) {
	r.providerKey = providerKey
	r.paymentType = paymentType
	return &payment.InstanceSelection{
		InstanceID:     "test-easypay-instance",
		ProviderKey:    providerKey,
		SupportedTypes: string(paymentType),
		Config:         map[string]string{},
	}, nil
}
```

- [ ] **Step 2: Run the routing tests and verify RED**

Run from repository root:

```bash
cd backend && go test ./internal/service -run 'TestEnabledVisibleMethodsForProviderIncludesKyrenInternationalMethods|TestVisibleMethodLoadBalancerRoutesKyrenMethodsToEasyPay' -count=1
```

Expected: FAIL. The current code only treats `alipay` and `wxpay` as visible EasyPay methods, and the load balancer forwards `creditcard` / `crypto` / `paynow` with an empty provider key.

- [ ] **Step 3: Add an EasyPay-international method helper and expand visible methods**

In `backend/internal/service/payment_visible_method_instances.go`, add this helper near `enabledVisibleMethodsForProvider`:

```go
func isEasyPayInternationalMethod(method string) bool {
	switch NormalizeVisibleMethod(method) {
	case payment.TypeCreditCard, payment.TypeCrypto, payment.TypePayNow:
		return true
	default:
		return false
	}
}
```

Then replace `enabledVisibleMethodsForProvider` with:

```go
func enabledVisibleMethodsForProvider(providerKey, supportedTypes string) []string {
	methodSet := make(map[string]struct{}, 5)
	addMethod := func(method string) {
		method = NormalizeVisibleMethod(method)
		switch method {
		case payment.TypeAlipay, payment.TypeWxpay, payment.TypeCreditCard, payment.TypeCrypto, payment.TypePayNow:
			methodSet[method] = struct{}{}
		}
	}

	switch strings.TrimSpace(providerKey) {
	case payment.TypeAlipay:
		if strings.TrimSpace(supportedTypes) == "" {
			addMethod(payment.TypeAlipay)
			break
		}
		for _, supportedType := range splitTypes(supportedTypes) {
			if NormalizeVisibleMethod(supportedType) == payment.TypeAlipay {
				addMethod(payment.TypeAlipay)
				break
			}
		}
	case payment.TypeWxpay:
		if strings.TrimSpace(supportedTypes) == "" {
			addMethod(payment.TypeWxpay)
			break
		}
		for _, supportedType := range splitTypes(supportedTypes) {
			if NormalizeVisibleMethod(supportedType) == payment.TypeWxpay {
				addMethod(payment.TypeWxpay)
				break
			}
		}
	case payment.TypeEasyPay:
		for _, supportedType := range splitTypes(supportedTypes) {
			addMethod(supportedType)
		}
	}

	methods := make([]string, 0, len(methodSet))
	for _, method := range []string{
		payment.TypeCreditCard,
		payment.TypeCrypto,
		payment.TypePayNow,
		payment.TypeAlipay,
		payment.TypeWxpay,
	} {
		if _, ok := methodSet[method]; ok {
			methods = append(methods, method)
		}
	}
	return methods
}
```

- [ ] **Step 4: Route international methods to EasyPay in the visible-method load balancer**

In `backend/internal/service/payment_resume_service.go`, replace `visibleMethodLoadBalancer.SelectInstance` with:

```go
func (lb *visibleMethodLoadBalancer) SelectInstance(ctx context.Context, providerKey string, paymentType payment.PaymentType, strategy payment.Strategy, orderAmount float64) (*payment.InstanceSelection, error) {
	visibleMethod := NormalizeVisibleMethod(paymentType)
	if providerKey != "" {
		return lb.inner.SelectInstance(ctx, providerKey, paymentType, strategy, orderAmount)
	}
	if isEasyPayInternationalMethod(visibleMethod) {
		return lb.inner.SelectInstance(ctx, payment.TypeEasyPay, paymentType, strategy, orderAmount)
	}
	if visibleMethod != payment.TypeAlipay && visibleMethod != payment.TypeWxpay {
		return lb.inner.SelectInstance(ctx, providerKey, paymentType, strategy, orderAmount)
	}

	inst, err := lb.configService.resolveEnabledVisibleMethodInstance(ctx, visibleMethod)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, fmt.Errorf("visible payment method %s has no enabled provider instance", visibleMethod)
	}
	return lb.inner.SelectInstance(ctx, inst.ProviderKey, paymentType, strategy, orderAmount)
}
```

This keeps the existing source-selection behavior for Alipay and WeChat Pay, while ensuring the new Kyren methods never fall through to Stripe or official providers with broad `supported_types` settings.

- [ ] **Step 5: Allow `resolveEnabledVisibleMethodInstance` to recognize the new methods if called directly**

In `backend/internal/service/payment_visible_method_instances.go`, replace this guard inside `resolveEnabledVisibleMethodInstance`:

```go
	if method != payment.TypeAlipay && method != payment.TypeWxpay {
		return nil, nil
	}
```

with:

```go
	if method != payment.TypeAlipay && method != payment.TypeWxpay && !isEasyPayInternationalMethod(method) {
		return nil, nil
	}
```

- [ ] **Step 6: Run the routing tests and verify GREEN**

Run from repository root:

```bash
cd backend && go test ./internal/service -run 'TestEnabledVisibleMethodsForProviderIncludesKyrenInternationalMethods|TestVisibleMethodLoadBalancerRoutesKyrenMethodsToEasyPay' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 3**

```bash
git add backend/internal/service/payment_visible_method_instances.go backend/internal/service/payment_resume_service.go backend/internal/service/payment_visible_method_instances_test.go
git commit -m "feat(payment): route Kyren EasyPay methods explicitly

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Expose EasyPay international methods in backend availability and limits

**Files:**
- Modify: `backend/internal/service/payment_config_limits_test.go`

- [ ] **Step 1: Write the backend availability test**

Append this test to `backend/internal/service/payment_config_limits_test.go`:

```go
func TestGetAvailableMethodLimitsExposesEasyPayInternationalMethods(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	_, err := client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeEasyPay).
		SetName("Kyren EasyPay International").
		SetConfig("{}").
		SetSupportedTypes("creditcard,crypto,paynow").
		SetLimits(`{"creditcard":{"singleMin":5,"singleMax":500},"crypto":{"singleMin":10,"singleMax":1000},"paynow":{"singleMin":3,"singleMax":300}}`).
		SetEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentProviderInstance.Create().
		SetProviderKey(payment.TypeStripe).
		SetName("Stripe Card").
		SetConfig(`{"currency":"CNY"}`).
		SetSupportedTypes("card,link").
		SetLimits(`{"stripe":{"singleMin":20,"singleMax":2000}}`).
		SetEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentConfigService{entClient: client}
	resp, err := svc.GetAvailableMethodLimits(ctx)
	require.NoError(t, err)

	creditCard, ok := resp.Methods[payment.TypeCreditCard]
	require.True(t, ok, "creditcard should be visible: %#v", resp.Methods)
	require.Equal(t, float64(5), creditCard.SingleMin)
	require.Equal(t, float64(500), creditCard.SingleMax)

	crypto, ok := resp.Methods[payment.TypeCrypto]
	require.True(t, ok, "crypto should be visible: %#v", resp.Methods)
	require.Equal(t, float64(10), crypto.SingleMin)
	require.Equal(t, float64(1000), crypto.SingleMax)

	payNow, ok := resp.Methods[payment.TypePayNow]
	require.True(t, ok, "paynow should be visible: %#v", resp.Methods)
	require.Equal(t, float64(3), payNow.SingleMin)
	require.Equal(t, float64(300), payNow.SingleMax)

	_, hasStripeCardAlias := resp.Methods[payment.TypeCard]
	require.False(t, hasStripeCardAlias, "Stripe card must stay aggregated under stripe")
	_, hasStripe := resp.Methods[payment.TypeStripe]
	require.True(t, hasStripe, "Stripe provider should remain a separate visible method")
}
```

- [ ] **Step 2: Run the availability test**

Run from repository root:

```bash
cd backend && go test ./internal/service -run TestGetAvailableMethodLimitsExposesEasyPayInternationalMethods -count=1
```

Expected: PASS after Tasks 1 and 3 are complete. If it fails, the failure indicates the available-method grouping does not keep Kyren `creditcard` independent from Stripe `card`.

- [ ] **Step 3: Run focused payment config tests**

Run from repository root:

```bash
cd backend && go test ./internal/service -run 'TestGetAvailableMethodLimits|TestPcGroupByPaymentType|TestPcAggregateMethodLimits|TestPcInstanceTypeLimits' -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit Task 4**

```bash
git add backend/internal/service/payment_config_limits_test.go
git commit -m "test(payment): cover EasyPay international method availability

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Add frontend provider config, labels, and ordering

**Files:**
- Modify: `frontend/src/components/payment/__tests__/providerConfig.spec.ts`
- Modify: `frontend/src/components/payment/providerConfig.ts`
- Modify: `frontend/src/types/payment.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`

- [ ] **Step 1: Write failing provider config tests**

In `frontend/src/components/payment/__tests__/providerConfig.spec.ts`, update the import to include `METHOD_ORDER` and `PROVIDER_SUPPORTED_TYPES`:

```ts
import { METHOD_ORDER, PAYMENT_CURRENCY_OPTIONS, PROVIDER_CONFIG_FIELDS, PROVIDER_SUPPORTED_TYPES } from '@/components/payment/providerConfig'
```

Append these tests:

```ts
describe('PROVIDER_SUPPORTED_TYPES.easypay', () => {
  it('includes Kyren EasyPay international payment types after domestic methods', () => {
    expect(PROVIDER_SUPPORTED_TYPES.easypay).toEqual([
      'alipay',
      'wxpay',
      'creditcard',
      'crypto',
      'paynow',
    ])
  })
})

describe('METHOD_ORDER', () => {
  it('places international EasyPay methods before domestic and provider-level methods', () => {
    expect([...METHOD_ORDER]).toEqual([
      'creditcard',
      'crypto',
      'paynow',
      'alipay',
      'alipay_direct',
      'wxpay',
      'wxpay_direct',
      'stripe',
      'airwallex',
    ])
  })
})
```

- [ ] **Step 2: Run provider config tests and verify RED**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/providerConfig.spec.ts
```

Expected: FAIL because EasyPay does not yet list `creditcard`, `crypto`, or `paynow`, and the method order does not start with those methods.

- [ ] **Step 3: Expand the frontend payment type union**

In `frontend/src/types/payment.ts`, replace the `PaymentType` union with:

```ts
export type PaymentType = 'alipay' | 'wxpay' | 'alipay_direct' | 'wxpay_direct' | 'stripe' | 'easypay' | 'creditcard' | 'crypto' | 'paynow' | 'airwallex'
```

- [ ] **Step 4: Update EasyPay supported types and method order**

In `frontend/src/components/payment/providerConfig.ts`, replace `PROVIDER_SUPPORTED_TYPES` with:

```ts
/** Maps provider key → available payment types. */
export const PROVIDER_SUPPORTED_TYPES: Record<string, string[]> = {
  easypay: ['alipay', 'wxpay', 'creditcard', 'crypto', 'paynow'],
  alipay: ['alipay'],
  wxpay: ['wxpay'],
  stripe: ['card', 'alipay', 'wxpay', 'link'],
  airwallex: ['airwallex'],
}
```

Replace `METHOD_ORDER` with:

```ts
/** Fixed display order for user-facing payment methods */
export const METHOD_ORDER = ['creditcard', 'crypto', 'paynow', 'alipay', 'alipay_direct', 'wxpay', 'wxpay_direct', 'stripe', 'airwallex'] as const
```

- [ ] **Step 5: Add English labels**

In `frontend/src/i18n/locales/en.ts`, find the `payment.methods` object and replace it with this object:

```ts
    methods: {
      easypay: 'EasyPay',
      alipay: 'Alipay',
      wxpay: 'WeChat Pay',
      stripe: 'Stripe',
      airwallex: 'Airwallex',
      creditcard: 'Credit Card',
      crypto: 'Crypto',
      paynow: 'PayNow',
      card: 'Card',
      link: 'Link',
      alipay_direct: 'Alipay (Direct)',
      wxpay_direct: 'WeChat Pay (Direct)',
    },
```

- [ ] **Step 6: Add Chinese labels**

In `frontend/src/i18n/locales/zh.ts`, find the `payment.methods` object and replace it with this object:

```ts
    methods: {
      easypay: '易支付',
      alipay: '支付宝',
      wxpay: '微信支付',
      stripe: 'Stripe',
      airwallex: 'Airwallex',
      creditcard: 'Credit Card',
      crypto: 'Crypto',
      paynow: 'PayNow',
      card: '银行卡',
      link: 'Link',
      alipay_direct: '支付宝（直连）',
      wxpay_direct: '微信支付（直连）',
    },
```

- [ ] **Step 7: Run provider config tests and verify GREEN**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/providerConfig.spec.ts
```

Expected: PASS.

- [ ] **Step 8: Commit Task 5**

```bash
git add frontend/src/types/payment.ts frontend/src/components/payment/providerConfig.ts frontend/src/components/payment/__tests__/providerConfig.spec.ts frontend/src/i18n/locales/en.ts frontend/src/i18n/locales/zh.ts
git commit -m "feat(payment): expose Kyren EasyPay methods in admin config

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Add frontend payment flow behavior for new visible methods

**Files:**
- Modify: `frontend/src/components/payment/__tests__/paymentFlow.spec.ts`
- Modify: `frontend/src/components/payment/paymentFlow.ts`

- [ ] **Step 1: Write failing payment flow tests**

In `frontend/src/components/payment/__tests__/paymentFlow.spec.ts`, inside `describe('getVisibleMethods', () => { ... })`, append:

```ts
  it('keeps Kyren EasyPay international methods as first-class visible methods', () => {
    const visible = getVisibleMethods({
      creditcard: methodLimit({ single_min: 5 }),
      crypto: methodLimit({ single_min: 10 }),
      paynow: methodLimit({ single_min: 3 }),
      card: methodLimit({ single_min: 99 }),
    })

    expect(visible).toEqual({
      creditcard: methodLimit({ single_min: 5 }),
      crypto: methodLimit({ single_min: 10 }),
      paynow: methodLimit({ single_min: 3 }),
    })
  })
```

Inside `describe('decidePaymentLaunch', () => { ... })`, append:

```ts
  it('uses the generic redirect flow for Credit Card pay_url responses', () => {
    const decision = decidePaymentLaunch(createOrderResult({
      pay_url: 'https://api.kyren.top/epay/redirect/card-order',
      payment_mode: 'popup',
      out_trade_no: 'sub2_card',
    }), {
      visibleMethod: 'creditcard',
      orderType: 'balance',
      isMobile: false,
    })

    expect(decision.kind).toBe('redirect_waiting')
    expect(decision.paymentState.paymentType).toBe('creditcard')
    expect(decision.paymentState.payUrl).toBe('https://api.kyren.top/epay/redirect/card-order')
    expect(decision.paymentState.outTradeNo).toBe('sub2_card')
  })

  it('uses the generic QR flow for Crypto qrcode responses', () => {
    const decision = decidePaymentLaunch(createOrderResult({
      qr_code: 'https://api.kyren.top/epay/qr/crypto-order.png',
      payment_mode: 'qrcode',
      out_trade_no: 'sub2_crypto',
    }), {
      visibleMethod: 'crypto',
      orderType: 'subscription',
      isMobile: false,
    })

    expect(decision.kind).toBe('qr_waiting')
    expect(decision.paymentState.paymentType).toBe('crypto')
    expect(decision.paymentState.qrCode).toBe('https://api.kyren.top/epay/qr/crypto-order.png')
    expect(decision.paymentState.orderType).toBe('subscription')
  })
```

Inside `describe('buildCreateOrderPayload', () => { ... })`, append:

```ts
  it('preserves Kyren EasyPay international payment types in create-order payloads', () => {
    expect(buildCreateOrderPayload({
      amount: 66,
      paymentType: 'paynow',
      orderType: 'balance',
      origin: 'https://app.example.com',
      isMobile: true,
      isWechatBrowser: true,
    })).toEqual({
      amount: 66,
      payment_type: 'paynow',
      order_type: 'balance',
      return_url: 'https://app.example.com/payment/result',
      is_mobile: true,
      payment_source: 'hosted_redirect',
    })
  })
```

- [ ] **Step 2: Run payment flow tests and verify RED**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/paymentFlow.spec.ts
```

Expected: FAIL because the new methods are not normalized as visible methods yet.

- [ ] **Step 3: Add visible method aliases and widen the visible method type**

In `frontend/src/components/payment/paymentFlow.ts`, replace `VISIBLE_METHOD_ALIASES` and `VisiblePaymentMethod` with:

```ts
const VISIBLE_METHOD_ALIASES = {
  alipay: 'alipay',
  alipay_direct: 'alipay',
  wxpay: 'wxpay',
  wxpay_direct: 'wxpay',
  creditcard: 'creditcard',
  crypto: 'crypto',
  paynow: 'paynow',
  stripe: 'stripe',
  airwallex: 'airwallex',
} as const

export type VisiblePaymentMethod = 'alipay' | 'wxpay' | 'creditcard' | 'crypto' | 'paynow' | 'stripe' | 'airwallex'
```

Do not add `card` to `VISIBLE_METHOD_ALIASES`; Stripe `card` remains a Stripe sub-method and must not become a standalone user-facing method through this EasyPay change.

- [ ] **Step 4: Run payment flow tests and verify GREEN**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/paymentFlow.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit Task 6**

```bash
git add frontend/src/components/payment/paymentFlow.ts frontend/src/components/payment/__tests__/paymentFlow.spec.ts
git commit -m "feat(payment): normalize Kyren EasyPay checkout methods

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Render independent payment selector buttons accessibly

**Files:**
- Create: `frontend/src/components/payment/__tests__/PaymentMethodSelector.spec.ts`
- Modify: `frontend/src/components/payment/PaymentMethodSelector.vue`

- [ ] **Step 1: Write failing selector rendering tests**

Create `frontend/src/components/payment/__tests__/PaymentMethodSelector.spec.ts` with this content:

```ts
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'

const labels: Record<string, string> = {
  'payment.paymentMethod': 'Payment Method',
  'payment.methods.creditcard': 'Credit Card',
  'payment.methods.crypto': 'Crypto',
  'payment.methods.paynow': 'PayNow',
  'payment.fee': 'Fee',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => labels[key] ?? key,
  }),
}))

describe('PaymentMethodSelector Kyren EasyPay methods', () => {
  it('renders Credit Card, Crypto, and PayNow as independent buttons in method order', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'creditcard',
        methods: [
          { type: 'paynow', fee_rate: 0, available: true },
          { type: 'creditcard', fee_rate: 0, available: true },
          { type: 'crypto', fee_rate: 0, available: true },
        ],
      },
    })

    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(3)
    expect(buttons[0].text()).toContain('Credit Card')
    expect(buttons[1].text()).toContain('Crypto')
    expect(buttons[2].text()).toContain('PayNow')
  })

  it('emits the exact Kyren payment type when a method is selected', async () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'creditcard',
        methods: [
          { type: 'crypto', fee_rate: 0, available: true },
        ],
      },
    })

    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('select')?.[0]).toEqual(['crypto'])
  })
})
```

- [ ] **Step 2: Run selector tests and verify RED**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/PaymentMethodSelector.spec.ts
```

Expected: FAIL until `METHOD_ORDER` and i18n labels from earlier tasks are present, or until icon fallback no longer defaults the new methods to Alipay. If Tasks 5 and 6 are already complete, the first test may pass; the implementation step still removes the misleading Alipay icon fallback.

- [ ] **Step 3: Map the new methods to the EasyPay icon**

In `frontend/src/components/payment/PaymentMethodSelector.vue`, add the EasyPay icon import after the existing icon imports:

```ts
import easypayIcon from '@/assets/icons/easypay.svg'
```

Replace `METHOD_ICONS` with:

```ts
const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  creditcard: easypayIcon,
  crypto: easypayIcon,
  paynow: easypayIcon,
}
```

Keep the existing `<button>` element, text label, `disabled` state, and 60px height. These already satisfy the accessibility requirements for semantic role, reachable keyboard control, visible text, and target size.

- [ ] **Step 4: Run selector tests and verify GREEN**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/PaymentMethodSelector.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit Task 7**

```bash
git add frontend/src/components/payment/PaymentMethodSelector.vue frontend/src/components/payment/__tests__/PaymentMethodSelector.spec.ts
git commit -m "feat(payment): render Kyren EasyPay method buttons

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Run focused verification and lint-only frontend validation

**Files:**
- No source changes unless a verification failure identifies a specific regression.

- [ ] **Step 1: Run focused backend provider tests**

Run from repository root:

```bash
cd backend && go test ./internal/payment/provider -run 'TestEasyPay|TestNormalizeEasyPayAPIBase' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run focused backend service tests**

Run from repository root:

```bash
cd backend && go test ./internal/service -run 'TestEnabledVisibleMethodsForProviderIncludesKyrenInternationalMethods|TestVisibleMethodLoadBalancerRoutesKyrenMethodsToEasyPay|TestGetAvailableMethodLimitsExposesEasyPayInternationalMethods|TestGetAvailableMethodLimits|TestPcGroupByPaymentType|TestPcAggregateMethodLimits|TestPcInstanceTypeLimits' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run focused frontend tests**

Run from repository root:

```bash
cd frontend && pnpm vitest run src/components/payment/__tests__/providerConfig.spec.ts src/components/payment/__tests__/paymentFlow.spec.ts src/components/payment/__tests__/PaymentMethodSelector.spec.ts
```

Expected: PASS.

- [ ] **Step 4: Run frontend lint check only**

Run from repository root:

```bash
cd frontend && pnpm lint:check
```

Expected: PASS. Do not run `pnpm build` for this task; the project preference is frontend lint-only validation.

- [ ] **Step 5: Run backend unit suite if focused tests are green**

Run from repository root:

```bash
cd backend && go test -tags=unit ./...
```

Expected: PASS. If unrelated pre-existing tests fail, record the exact failing package and output before continuing.

- [ ] **Step 6: Commit verification-only fixes if any were required**

If Steps 1-5 required source fixes, stage only the files touched by those fixes and commit:

```bash
git add <exact-files-fixed-during-verification>
git commit -m "fix(payment): stabilize Kyren EasyPay method tests

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

If no source fixes were required, do not create a verification-only commit.

---

## Self-Review Checklist

- Spec coverage: Tasks 1-4 cover backend constants, EasyPay capability, Kyren `mapi.php` compatibility, visible method aggregation, explicit EasyPay routing, and Stripe `card` separation. Tasks 5-7 cover admin supported types, user-facing labels, ordering, payload preservation, generic redirect/QR launch behavior, and independent buttons. Task 8 covers backend tests and frontend lint-only validation.
- Security coverage: Existing EasyPay MD5 signing and webhook verification remain unchanged; the plan does not expose `pkey`, does not bypass signature verification, and keeps refund support unchanged because Kyren documents compatible refunds as unsupported.
- Type consistency: Backend constants are `TypeCreditCard`, `TypeCrypto`, `TypePayNow`; frontend string values are exactly `creditcard`, `crypto`, `paynow`; tests and UI labels use the same values.
- Scope control: The plan does not add a `kyren` provider, does not alter Stripe `card` / `link`, does not add blockchain wallet verification, and does not add automatic refund behavior.
