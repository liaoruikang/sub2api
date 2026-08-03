package service

import "sort"

const AccountExtraHighestSchedulingMode = "highest_scheduling_mode"

var deprecatedHighestSchedulingExtraKeys = [...]string{
	"highest_scheduling_recovery_minutes",
	"highest_scheduling_suppressed",
	"highest_scheduling_suppressed_until",
	"highest_scheduling_suppressed_at",
	"highest_scheduling_suppressed_reason",
}

// SanitizeAccountHighestSchedulingExtra enforces mode-only highest scheduling
// semantics. The mode is accepted only as a JSON boolean; deprecated runtime
// recovery and suppression keys are silently discarded. Other Extra keys are
// left untouched. The input map is never modified.
func SanitizeAccountHighestSchedulingExtra(extra map[string]any) map[string]any {
	if extra == nil {
		return nil
	}
	sanitized := make(map[string]any, len(extra))
	for key, value := range extra {
		sanitized[key] = value
	}
	if mode, exists := sanitized[AccountExtraHighestSchedulingMode]; exists {
		if _, ok := mode.(bool); !ok {
			delete(sanitized, AccountExtraHighestSchedulingMode)
		}
	}
	for _, key := range deprecatedHighestSchedulingExtraKeys {
		delete(sanitized, key)
	}
	return sanitized
}

func (a *Account) IsHighestSchedulingModeConfigured() bool {
	if a == nil || a.Extra == nil {
		return false
	}
	mode, ok := a.Extra[AccountExtraHighestSchedulingMode].(bool)
	return ok && mode
}

func accountHighestSchedulingEffective(account *Account) bool {
	return account != nil && account.IsHighestSchedulingModeConfigured()
}

func isBetterAccountByHighestSchedulingPriorityAndLastUsed(candidate, current *Account, preferOAuth bool) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateHighest := accountHighestSchedulingEffective(candidate)
	currentHighest := accountHighestSchedulingEffective(current)
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

func sortAccountsByHighestSchedulingPriorityAndLastUsed(accounts []*Account, preferOAuth bool) {
	sort.SliceStable(accounts, func(i, j int) bool {
		return isBetterAccountByHighestSchedulingPriorityAndLastUsed(accounts[i], accounts[j], preferOAuth)
	})
}

func highestSchedulingLoadTier(accounts []accountWithLoad) []accountWithLoad {
	for _, account := range accounts {
		if accountHighestSchedulingEffective(account.account) {
			out := make([]accountWithLoad, 0, len(accounts))
			for _, candidate := range accounts {
				if accountHighestSchedulingEffective(candidate.account) {
					out = append(out, candidate)
				}
			}
			return out
		}
	}
	return accounts
}

func sortByHighestSchedulingLoadCandidates(accounts []accountWithLoad, preferOAuth bool) {
	sort.SliceStable(accounts, func(i, j int) bool {
		return isBetterAccountWithLoadByHighestSchedulingPriorityLoadAndLastUsed(accounts[i], accounts[j], preferOAuth)
	})
}

func isBetterAccountWithLoadByHighestSchedulingPriorityLoadAndLastUsed(candidate, current accountWithLoad, preferOAuth bool) bool {
	candidateHighest := accountHighestSchedulingEffective(candidate.account)
	currentHighest := accountHighestSchedulingEffective(current.account)
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
