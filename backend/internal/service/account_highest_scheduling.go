package service

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	AccountExtraHighestSchedulingMode             = "highest_scheduling_mode"
	AccountExtraHighestSchedulingRecoveryMinutes  = "highest_scheduling_recovery_minutes"
	AccountExtraHighestSchedulingSuppressed       = "highest_scheduling_suppressed"
	AccountExtraHighestSchedulingSuppressedUntil  = "highest_scheduling_suppressed_until"
	AccountExtraHighestSchedulingSuppressedAt     = "highest_scheduling_suppressed_at"
	AccountExtraHighestSchedulingSuppressedReason = "highest_scheduling_suppressed_reason"
)

func (a *Account) IsHighestSchedulingModeConfigured() bool {
	if a == nil || a.Extra == nil {
		return false
	}
	return highestSchedulingBool(a.Extra[AccountExtraHighestSchedulingMode])
}

func (a *Account) GetHighestSchedulingRecoveryMinutes() int {
	if a == nil || a.Extra == nil {
		return 0
	}
	minutes := highestSchedulingInt(a.Extra[AccountExtraHighestSchedulingRecoveryMinutes])
	if minutes < 0 {
		return 0
	}
	return minutes
}

func (a *Account) GetHighestSchedulingSuppressedUntil() *time.Time {
	if a == nil || a.Extra == nil {
		return nil
	}
	return highestSchedulingTime(a.Extra[AccountExtraHighestSchedulingSuppressedUntil])
}

func (a *Account) IsHighestSchedulingModeSuppressed(now time.Time) bool {
	if a == nil || a.Extra == nil {
		return false
	}
	if highestSchedulingBool(a.Extra[AccountExtraHighestSchedulingSuppressed]) {
		return true
	}
	until := a.GetHighestSchedulingSuppressedUntil()
	return until != nil && now.Before(*until)
}

func (a *Account) IsHighestSchedulingModeEffective(now time.Time) bool {
	if !a.IsHighestSchedulingModeConfigured() {
		return false
	}
	return !a.IsHighestSchedulingModeSuppressed(now)
}

func accountHighestSchedulingEffective(account *Account, now time.Time) bool {
	return account != nil && account.IsHighestSchedulingModeEffective(now)
}

func isBetterAccountByHighestSchedulingPriorityAndLastUsed(candidate, current *Account, preferOAuth bool, now time.Time) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateHighest := accountHighestSchedulingEffective(candidate, now)
	currentHighest := accountHighestSchedulingEffective(current, now)
	if candidateHighest != currentHighest {
		return candidateHighest
	}
	if candidate.Priority != current.Priority {
		return candidate.Priority < current.Priority
	}
	switch {
	case candidate.LastUsedAt == nil && current.LastUsedAt != nil:
		return true
	case candidate.LastUsedAt != nil && current.LastUsedAt == nil:
		return false
	case candidate.LastUsedAt == nil && current.LastUsedAt == nil:
		if preferOAuth && candidate.Type != current.Type {
			return candidate.Type == AccountTypeOAuth
		}
		return false
	default:
		return candidate.LastUsedAt.Before(*current.LastUsedAt)
	}
}

func sortAccountsByHighestSchedulingPriorityAndLastUsed(accounts []*Account, preferOAuth bool, now time.Time) {
	sort.SliceStable(accounts, func(i, j int) bool {
		return isBetterAccountByHighestSchedulingPriorityAndLastUsed(accounts[i], accounts[j], preferOAuth, now)
	})
}

func filterByHighestSchedulingLoadCandidates(accounts []accountWithLoad, now time.Time) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	hasHighest := false
	for _, account := range accounts {
		if accountHighestSchedulingEffective(account.account, now) {
			hasHighest = true
			break
		}
	}
	if !hasHighest {
		return accounts
	}
	out := make([]accountWithLoad, 0, len(accounts))
	for _, account := range accounts {
		if accountHighestSchedulingEffective(account.account, now) {
			out = append(out, account)
		}
	}
	return out
}

func isBetterAccountWithLoadByHighestSchedulingPriorityLoadAndLastUsed(candidate, current accountWithLoad, preferOAuth bool, now time.Time) bool {
	candidateHighest := accountHighestSchedulingEffective(candidate.account, now)
	currentHighest := accountHighestSchedulingEffective(current.account, now)
	if candidateHighest != currentHighest {
		return candidateHighest
	}
	if candidate.account.Priority != current.account.Priority {
		return candidate.account.Priority < current.account.Priority
	}
	if candidate.loadInfo.LoadRate != current.loadInfo.LoadRate {
		return candidate.loadInfo.LoadRate < current.loadInfo.LoadRate
	}
	if candidate.loadInfo.WaitingCount != current.loadInfo.WaitingCount {
		return candidate.loadInfo.WaitingCount < current.loadInfo.WaitingCount
	}
	switch {
	case candidate.account.LastUsedAt == nil && current.account.LastUsedAt != nil:
		return true
	case candidate.account.LastUsedAt != nil && current.account.LastUsedAt == nil:
		return false
	case candidate.account.LastUsedAt == nil && current.account.LastUsedAt == nil:
		if preferOAuth && candidate.account.Type != current.account.Type {
			return candidate.account.Type == AccountTypeOAuth
		}
		return false
	default:
		return candidate.account.LastUsedAt.Before(*current.account.LastUsedAt)
	}
}

func sortAccountsByHighestSchedulingPriorityOnly(accounts []*Account, preferOAuth bool, now time.Time) {
	sort.SliceStable(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		aHighest := accountHighestSchedulingEffective(a, now)
		bHighest := accountHighestSchedulingEffective(b, now)
		if aHighest != bHighest {
			return aHighest
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if preferOAuth && a.Type != b.Type {
			return a.Type == AccountTypeOAuth
		}
		return false
	})
}

func highestSchedulingBool(value any) bool {
	v, ok := value.(bool)
	return ok && v
}

func highestSchedulingInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return 0
}

func highestSchedulingTime(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return nil
		}
		if t, err := time.Parse(time.RFC3339, text); err == nil {
			return &t
		}
	case json.Number:
		return highestSchedulingUnixTime(v.String())
	case int:
		return highestSchedulingUnixTime(strconv.Itoa(v))
	case int64:
		return highestSchedulingUnixTime(strconv.FormatInt(v, 10))
	case float64:
		return highestSchedulingUnixTime(strconv.FormatInt(int64(v), 10))
	}
	return nil
}

func highestSchedulingUnixTime(value string) *time.Time {
	ts, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || ts <= 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}
