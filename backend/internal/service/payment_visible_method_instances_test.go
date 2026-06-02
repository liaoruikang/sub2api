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
