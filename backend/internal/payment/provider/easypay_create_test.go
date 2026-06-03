package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEasyPayCreateAPIPaymentUsesKyrenInternationalTypes(t *testing.T) {
	t.Parallel()

	for _, method := range []payment.PaymentType{payment.TypeCreditCard, payment.TypeCrypto, payment.TypePayNow} {
		method := method
		t.Run(method, func(t *testing.T) {
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
				PaymentType: method,
				Subject:     "AI credits",
				ClientIP:    "203.0.113.10",
			})
			if err != nil {
				t.Fatalf("CreatePayment returned error: %v", err)
			}
			if gotPath != "/mapi.php" {
				t.Fatalf("create path = %q, want /mapi.php", gotPath)
			}
			if got := gotForm.Get("type"); got != method {
				t.Fatalf("form[type] = %q, want %q (form=%v)", got, method, gotForm)
			}
			if got := gotForm.Get("pid"); got != "pid-1" {
				t.Fatalf("form[pid] = %q, want pid-1", got)
			}
			if got := gotForm.Get("out_trade_no"); got != "sub2_img" {
				t.Fatalf("form[out_trade_no] = %q, want sub2_img", got)
			}
			if got := gotForm.Get("money"); got != "9.99" {
				t.Fatalf("form[money] = %q, want 9.99", got)
			}
			if got := gotForm.Get("clientip"); got != "203.0.113.10" {
				t.Fatalf("form[clientip] = %q, want 203.0.113.10", got)
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
		})
	}
}

func TestEasyPayCreatePopupPaymentUsesKyrenInternationalTypes(t *testing.T) {
	t.Parallel()

	for _, method := range []payment.PaymentType{payment.TypeCreditCard, payment.TypeCrypto, payment.TypePayNow} {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			provider := newTestEasyPay(t, "https://api.kyren.top/epay")
			provider.config["paymentMode"] = paymentModePopup
			resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
				OrderID:     "sub2_popup",
				Amount:      "12.34",
				PaymentType: method,
				Subject:     "AI credits",
				IsMobile:    true,
			})
			if err != nil {
				t.Fatalf("CreatePayment returned error: %v", err)
			}
			parsed, err := url.Parse(resp.PayURL)
			if err != nil {
				t.Fatalf("parse PayURL: %v", err)
			}
			if parsed.Path != "/epay/submit.php" {
				t.Fatalf("PayURL path = %q, want /epay/submit.php", parsed.Path)
			}
			query := parsed.Query()
			if got := query.Get("type"); got != method {
				t.Fatalf("query[type] = %q, want %q", got, method)
			}
			if got := query.Get("out_trade_no"); got != "sub2_popup" {
				t.Fatalf("query[out_trade_no] = %q, want sub2_popup", got)
			}
			if got := query.Get("money"); got != "12.34" {
				t.Fatalf("query[money] = %q, want 12.34", got)
			}
			if got := query.Get("device"); got != deviceMobile {
				t.Fatalf("query[device] = %q, want %q", got, deviceMobile)
			}
			if got := query.Get("sign_type"); got != signTypeMD5 {
				t.Fatalf("query[sign_type] = %q, want %q", got, signTypeMD5)
			}
			if got := query.Get("sign"); got == "" {
				t.Fatalf("query[sign] is empty (url=%s)", resp.PayURL)
			}
		})
	}
}
