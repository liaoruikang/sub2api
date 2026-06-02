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
