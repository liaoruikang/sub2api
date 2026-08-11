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

func TestAPIKeyService_RejectsV24AuthSnapshotWithoutOrderedGroups(t *testing.T) {
	svc := &APIKeyService{}
	groupID := int64(16)

	apiKey, ok, err := svc.applyAuthCacheEntry("k-v24-single-group", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  24,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
		},
	})

	if err != nil {
		t.Fatalf("expected v24 snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatal("expected v24 auth snapshot to be rejected after ordered groups were added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyService_AuthSnapshotLegacyGroupMaterializesGroupIDs(t *testing.T) {
	svc := &APIKeyService{}
	groupID := int64(9)
	apiKey := svc.snapshotToAPIKey("k-legacy-group", &APIKeyAuthSnapshot{
		Version:  apiKeyAuthSnapshotVersion,
		APIKeyID: 1,
		UserID:   2,
		GroupID:  &groupID,
		Status:   StatusActive,
		User: APIKeyAuthUserSnapshot{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Concurrency: 3,
		},
		Group: &APIKeyAuthGroupSnapshot{
			ID:               groupID,
			Platform:         PlatformOpenAI,
			Status:           StatusActive,
			SubscriptionType: SubscriptionTypeStandard,
		},
	})

	if apiKey == nil || apiKey.Group == nil {
		t.Fatalf("expected API key with legacy group, got %#v", apiKey)
	}
	if len(apiKey.GroupIDs) != 1 || apiKey.GroupIDs[0] != groupID {
		t.Fatalf("expected legacy group id to materialize, got %#v", apiKey.GroupIDs)
	}
	if len(apiKey.Groups) != 1 || apiKey.Groups[0].ID != groupID {
		t.Fatalf("expected legacy group to materialize in Groups, got %#v", apiKey.Groups)
	}
}

func TestAPIKeyService_AuthSnapshotPreservesOrderedGroups(t *testing.T) {
	svc := &APIKeyService{}
	groups := []*Group{
		{ID: 11, Name: "first", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
		{ID: 22, Name: "second", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
	}
	apiKey := &APIKey{
		ID:       1,
		UserID:   2,
		Key:      "k-ordered-groups",
		GroupID:  &groups[0].ID,
		GroupIDs: []int64{11, 22},
		Groups:   groups,
		Group:    groups[0],
		Status:   StatusActive,
		User: &User{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Concurrency: 3,
		},
	}

	snapshot := svc.snapshotFromAPIKey(nil, apiKey)
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if len(snapshot.GroupIDs) != 2 || snapshot.GroupIDs[0] != 11 || snapshot.GroupIDs[1] != 22 {
		t.Fatalf("expected ordered group ids, got %#v", snapshot.GroupIDs)
	}
	if len(snapshot.Groups) != 2 || snapshot.Groups[0].ID != 11 || snapshot.Groups[1].ID != 22 {
		t.Fatalf("expected ordered group snapshots, got %#v", snapshot.Groups)
	}

	roundTrip := svc.snapshotToAPIKey(apiKey.Key, snapshot)
	if roundTrip == nil || len(roundTrip.GroupIDs) != 2 || roundTrip.GroupIDs[0] != 11 || roundTrip.GroupIDs[1] != 22 {
		t.Fatalf("expected ordered group ids after round trip, got %#v", roundTrip)
	}
	if len(roundTrip.Groups) != 2 || roundTrip.Groups[0].ID != 11 || roundTrip.Groups[1].ID != 22 {
		t.Fatalf("expected ordered groups after round trip, got %#v", roundTrip.Groups)
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
