package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	OpenAISessionControlEnabledExtraKey      = "openai_session_control_enabled"
	OpenAISessionMaxCountExtraKey            = "openai_session_max_count"
	OpenAISessionIdleTimeoutSecondsExtraKey  = "openai_session_idle_timeout_seconds"
	OpenAISessionSlotRotationEnabledExtraKey = "openai_session_slot_rotation_enabled"

	DefaultOpenAISessionMaxCount           = 35
	MinimumOpenAISessionMaxCount           = 3
	DefaultOpenAISessionIdleTimeoutSeconds = int((24 * time.Hour) / time.Second)
	maximumOpenAISessionIdleTimeoutSeconds = int64(math.MaxInt64 / int64(time.Second))
)

func (a *Account) IsOpenAISessionControlEnabled() bool {
	if a == nil || !a.IsOpenAIOAuth() || a.Extra == nil {
		return false
	}
	enabled, ok := a.Extra[OpenAISessionControlEnabledExtraKey].(bool)
	return ok && enabled
}

func (a *Account) GetOpenAISessionMaxCount() int {
	if !a.IsOpenAISessionControlEnabled() {
		return 0
	}
	value := parseExtraInt(a.Extra[OpenAISessionMaxCountExtraKey])
	if value < MinimumOpenAISessionMaxCount {
		return DefaultOpenAISessionMaxCount
	}
	return value
}

func (a *Account) GetOpenAISessionIdleTimeout() time.Duration {
	if !a.IsOpenAISessionControlEnabled() {
		return 0
	}
	seconds := parseExtraInt(a.Extra[OpenAISessionIdleTimeoutSecondsExtraKey])
	if seconds <= 0 || int64(seconds) > maximumOpenAISessionIdleTimeoutSeconds {
		seconds = DefaultOpenAISessionIdleTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (a *Account) IsOpenAISessionSlotRotationEnabled() bool {
	if !a.IsOpenAISessionControlEnabled() {
		return false
	}
	enabled, ok := a.Extra[OpenAISessionSlotRotationEnabledExtraKey].(bool)
	return ok && enabled
}

func HasOpenAISessionControlUpdate(extra map[string]any) bool {
	if extra == nil {
		return false
	}
	for _, key := range []string{
		OpenAISessionControlEnabledExtraKey,
		OpenAISessionMaxCountExtraKey,
		OpenAISessionIdleTimeoutSecondsExtraKey,
		OpenAISessionSlotRotationEnabledExtraKey,
	} {
		if _, ok := extra[key]; ok {
			return true
		}
	}
	return false
}

func NormalizeOpenAISessionControlExtra(platform, accountType string, extra map[string]any) (map[string]any, error) {
	normalized := cloneAccountExtraMap(extra)
	enabled, hasEnabled, err := strictBoolExtra(normalized, OpenAISessionControlEnabledExtraKey)
	if err != nil {
		return nil, err
	}
	rotationEnabled, hasRotationEnabled, err := strictBoolExtra(normalized, OpenAISessionSlotRotationEnabledExtraKey)
	if err != nil {
		return nil, err
	}

	eligible := platform == PlatformOpenAI && accountType == AccountTypeOAuth
	if !eligible {
		if (hasEnabled && enabled) || (hasRotationEnabled && rotationEnabled) {
			return nil, fmt.Errorf("OpenAI SessionID control is only available for OpenAI OAuth accounts")
		}
		deleteOpenAISessionControlExtra(normalized)
		return normalized, nil
	}
	if normalized == nil {
		normalized = make(map[string]any)
	}

	if !hasEnabled {
		enabled = false
		normalized[OpenAISessionControlEnabledExtraKey] = false
	}
	if !enabled {
		delete(normalized, OpenAISessionMaxCountExtraKey)
		delete(normalized, OpenAISessionIdleTimeoutSecondsExtraKey)
		delete(normalized, OpenAISessionSlotRotationEnabledExtraKey)
		return normalized, nil
	}
	if !hasRotationEnabled {
		rotationEnabled = false
	}

	maxCount := DefaultOpenAISessionMaxCount
	if raw, ok := normalized[OpenAISessionMaxCountExtraKey]; ok {
		maxCount, err = strictPositiveIntExtra(raw)
		if err != nil || maxCount < MinimumOpenAISessionMaxCount {
			return nil, fmt.Errorf("%s must be an integer greater than or equal to %d", OpenAISessionMaxCountExtraKey, MinimumOpenAISessionMaxCount)
		}
	}
	timeoutSeconds := DefaultOpenAISessionIdleTimeoutSeconds
	if raw, ok := normalized[OpenAISessionIdleTimeoutSecondsExtraKey]; ok {
		timeoutSeconds, err = strictPositiveIntExtra(raw)
		if err != nil || int64(timeoutSeconds) > maximumOpenAISessionIdleTimeoutSeconds {
			return nil, fmt.Errorf("%s must be a positive integer within the supported duration range", OpenAISessionIdleTimeoutSecondsExtraKey)
		}
	}

	normalized[OpenAISessionMaxCountExtraKey] = maxCount
	normalized[OpenAISessionIdleTimeoutSecondsExtraKey] = timeoutSeconds
	normalized[OpenAISessionSlotRotationEnabledExtraKey] = rotationEnabled
	return normalized, nil
}

func MergeAndValidateOpenAISessionControlUpdate(account *Account, updates map[string]any) error {
	if !HasOpenAISessionControlUpdate(updates) {
		return nil
	}
	if account == nil || !account.IsOpenAIOAuth() || account.IsCredentialShadow() {
		return fmt.Errorf("OpenAI SessionID control bulk editing requires OpenAI OAuth accounts")
	}
	merged := cloneAccountExtraMap(account.Extra)
	if merged == nil {
		merged = make(map[string]any)
	}
	for key, value := range updates {
		merged[key] = value
	}
	_, err := NormalizeOpenAISessionControlExtra(account.Platform, account.Type, merged)
	return err
}

func deleteOpenAISessionControlExtra(extra map[string]any) {
	delete(extra, OpenAISessionControlEnabledExtraKey)
	delete(extra, OpenAISessionMaxCountExtraKey)
	delete(extra, OpenAISessionIdleTimeoutSecondsExtraKey)
	delete(extra, OpenAISessionSlotRotationEnabledExtraKey)
}

func cloneAccountExtraMap(extra map[string]any) map[string]any {
	if extra == nil {
		return nil
	}
	cloned := make(map[string]any, len(extra))
	for key, value := range extra {
		cloned[key] = value
	}
	return cloned
}

func strictBoolExtra(extra map[string]any, key string) (bool, bool, error) {
	raw, ok := extra[key]
	if !ok {
		return false, false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, true, fmt.Errorf("%s must be a boolean", key)
	}
	return value, true, nil
}

func strictPositiveIntExtra(raw any) (int, error) {
	var value int64
	switch typed := raw.(type) {
	case int:
		value = int64(typed)
	case int32:
		value = int64(typed)
	case int64:
		value = typed
	case float64:
		if math.Trunc(typed) != typed || typed > float64(math.MaxInt) {
			return 0, fmt.Errorf("not an integer")
		}
		value = int64(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, err
		}
		value = parsed
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, err
		}
		value = parsed
	default:
		return 0, fmt.Errorf("not an integer")
	}
	if value <= 0 || value > int64(math.MaxInt) {
		return 0, fmt.Errorf("out of range")
	}
	return int(value), nil
}
