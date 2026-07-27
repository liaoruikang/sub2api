package service

import "testing"

func TestAPIKeyService_RejectsV12AuthSnapshotWithoutSelectedLimitedTimeMultiplierFields(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-limited-multiplier", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  12,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v12 auth snapshot to be rejected after selected limited-time multiplier fields were added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyService_AuthSnapshotRoundTripsLimitedTimeRPMLimit(t *testing.T) {
	svc := &APIKeyService{}
	apiKey := svc.snapshotToAPIKey("k-limited-time-rpm", &APIKeyAuthSnapshot{
		Version:  apiKeyAuthSnapshotVersion,
		APIKeyID: 1,
		UserID:   2,
		Status:   StatusActive,
		User: APIKeyAuthUserSnapshot{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Balance:     10,
			Concurrency: 3,
		},
		Group: &APIKeyAuthGroupSnapshot{
			ID:                  9,
			Name:                "openai",
			Platform:            PlatformOpenAI,
			Status:              StatusActive,
			SubscriptionType:    SubscriptionTypeStandard,
			RateMultiplier:      1,
			LimitedTimeRPMLimit: 123,
		},
	})

	if apiKey == nil || apiKey.Group == nil {
		t.Fatalf("expected API key with group from snapshot, got %#v", apiKey)
	}
	if apiKey.Group.LimitedTimeRPMLimit != 123 {
		t.Fatalf("expected limited-time RPM limit to round-trip, got %d", apiKey.Group.LimitedTimeRPMLimit)
	}
}

func TestAPIKeyService_RejectsV15AuthSnapshotWithoutReasoningEffortPolicy(t *testing.T) {
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-reasoning-mappings", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{Version: 15},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatal("expected v15 auth snapshot to be rejected after reasoning effort policy was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}
