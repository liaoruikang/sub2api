package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func highestSchedulingExtraRequest() map[string]any {
	return map[string]any{
		service.AccountExtraHighestSchedulingMode: false,
		"highest_scheduling_recovery_minutes":     15,
		"highest_scheduling_suppressed":           true,
		"highest_scheduling_suppressed_until":     "2026-06-09T12:15:00Z",
		"highest_scheduling_suppressed_at":        "2026-06-09T12:00:00Z",
		"highest_scheduling_suppressed_reason":    "boom",
		"unrelated":                               1,
	}
}

func TestAccountHandlerCreateDefersHighestSchedulingCleanupToService(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)
	body, err := json.Marshal(map[string]any{
		"name":        "mode-only-create",
		"platform":    "openai",
		"type":        "apikey",
		"credentials": map[string]any{"api_key": "sk-test"},
		"extra":       highestSchedulingExtraRequest(),
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.createdAccounts, 1)
	// The handler intentionally does not duplicate service-layer sanitization.
	require.Contains(t, adminSvc.createdAccounts[0].Extra, "highest_scheduling_recovery_minutes")
}

func TestAccountHandlerUpdateDefersHighestSchedulingCleanupToService(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)
	body, err := json.Marshal(map[string]any{"extra": highestSchedulingExtraRequest()})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
