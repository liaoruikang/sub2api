//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type availableGroupsUserRepoStub struct {
	UserRepository
	user *User
}

func (s *availableGroupsUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type availableGroupsGroupRepoStub struct {
	GroupRepository
	groups []Group
}

func (s *availableGroupsGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return s.groups, nil
}

type availableGroupsSubscriptionRepoStub struct {
	UserSubscriptionRepository
}

func (availableGroupsSubscriptionRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return []UserSubscription{}, nil
}

type availableGroupsTagRepoStub struct {
	UserTagRepository
	groupIDs  []int64
	groupTags map[int64][]UserTag
	err       error
	tagsErr   error
}

func (s availableGroupsTagRepoStub) GetGroupIDsByUserID(context.Context, int64) ([]int64, error) {
	return s.groupIDs, s.err
}

func (s availableGroupsTagRepoStub) GetByGroupID(_ context.Context, groupID int64) ([]UserTag, error) {
	if s.tagsErr != nil {
		return nil, s.tagsErr
	}
	return s.groupTags[groupID], nil
}

func TestAPIKeyServiceGetAvailableGroupsIncludesAuthorizedGroupsAndSurvivesTagLookupFailure(t *testing.T) {
	tests := []struct {
		name         string
		tagRepo      UserTagRepository
		expected     []int64
		expectedTags []UserTag
	}{
		{
			name: "includes tag-derived exclusive groups",
			tagRepo: availableGroupsTagRepoStub{
				groupIDs:  []int64{33},
				groupTags: map[int64][]UserTag{33: {{ID: 5, Name: "Downstream"}}},
			},
			expected:     []int64{22, 33, 11},
			expectedTags: []UserTag{{ID: 5, Name: "Downstream"}},
		},
		{
			name:     "keeps public and manual groups when tag lookup fails",
			tagRepo:  availableGroupsTagRepoStub{err: errors.New("tag tables unavailable")},
			expected: []int64{22, 11},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAPIKeyService(
				nil,
				&availableGroupsUserRepoStub{user: &User{ID: 7, AllowedGroups: []int64{22}}},
				nil,
				nil,
				nil,
				nil,
				nil,
			)
			service.groupRepo = &availableGroupsGroupRepoStub{groups: []Group{
				{ID: 11, Name: "Public", IsExclusive: false, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
				{ID: 22, Name: "Manual", IsExclusive: true, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
				{ID: 33, Name: "Tag-derived", IsExclusive: true, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
				{ID: 44, Name: "Other exclusive", IsExclusive: true, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
			}}
			service.userSubRepo = availableGroupsSubscriptionRepoStub{}
			service.SetUserTagRepository(tt.tagRepo)

			groups, err := service.GetAvailableGroups(context.Background(), 7)

			require.NoError(t, err)
			ids := make([]int64, 0, len(groups))
			for _, group := range groups {
				ids = append(ids, group.ID)
			}
			require.Equal(t, tt.expected, ids)
			if tt.expectedTags != nil {
				require.Equal(t, tt.expectedTags, groups[1].Tags)
				require.Equal(t, []int64{5}, groups[1].TagIDs)
			}
		})
	}
}
