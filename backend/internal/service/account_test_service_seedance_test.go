package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
)

type seedanceAccountTestBody struct {
	io.Reader
	closed *bool
}

func (b *seedanceAccountTestBody) Close() error {
	*b.closed = true
	return nil
}

type seedanceAccountTestUpstream struct {
	calls            int
	createBody       []byte
	createBodyClosed bool
}

func (u *seedanceAccountTestUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return u.respond(req)
}

func (u *seedanceAccountTestUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.respond(req)
}

func (u *seedanceAccountTestUpstream) respond(req *http.Request) (*http.Response, error) {
	u.calls++
	switch u.calls {
	case 1:
		u.createBody, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: &seedanceAccountTestBody{
				Reader: strings.NewReader(`{"task":{"id":"task-1"}}`),
				closed: &u.createBodyClosed,
			},
		}, nil
	case 2:
		if !u.createBodyClosed {
			return nil, errors.New("create response body was not closed before polling")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"task":{"id":"task-1","status":"completed","outputs":["https://cdn.example.test/video.mp4"]}}`)),
		}, nil
	default:
		return nil, errors.New("unexpected Seedance account test request")
	}
}

func TestResolveSeedanceVideoPrompt(t *testing.T) {
	if got := resolveSeedanceVideoPrompt("   "); got != defaultSeedanceVideoTestPrompt {
		t.Fatalf("empty prompt resolved to %q", got)
	}
	if got := resolveSeedanceVideoPrompt("  custom camera move  "); got != "custom camera move" {
		t.Fatalf("custom prompt resolved to %q", got)
	}
}

func TestSeedanceVideoProbeTimeout(t *testing.T) {
	if seedanceVideoProbeTimeout != 5*time.Minute {
		t.Fatalf("seedanceVideoProbeTimeout = %s, want 5m", seedanceVideoProbeTimeout)
	}
}

func TestSeedanceAccountTestUsesCustomPromptAndReturnsStringOutputURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/1/test", nil)

	upstream := &seedanceAccountTestUpstream{}
	service := &AccountTestService{httpUpstream: upstream}
	account := &Account{
		ID: 1, Platform: PlatformSeedance, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": DefaultSeedanceBaseURL},
	}

	err := service.testSeedanceAccountConnection(ctx, account, "dreamina-seedance-2-0-hc", "  custom tracking shot  ", AccountTestModeGrokVideo)
	if err != nil {
		t.Fatalf("testSeedanceAccountConnection() error = %v", err)
	}
	if upstream.calls != 2 {
		t.Fatalf("upstream calls = %d, want 2", upstream.calls)
	}
	if !upstream.createBodyClosed {
		t.Fatal("create response body was not closed")
	}
	var payload map[string]any
	if err := json.Unmarshal(upstream.createBody, &payload); err != nil {
		t.Fatalf("decode create body: %v", err)
	}
	if payload["prompt"] != "custom tracking shot" {
		t.Fatalf("prompt = %#v, want custom prompt", payload["prompt"])
	}
	responseBody := recorder.Body.String()
	if !strings.Contains(responseBody, `"type":"video"`) || !strings.Contains(responseBody, "https://cdn.example.test/video.mp4") {
		t.Fatalf("SSE response did not include video output: %s", responseBody)
	}
}
