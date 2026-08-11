package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewOrderedGroupFailoverStatePreservesOrderAndDeduplicates(t *testing.T) {
	state := NewOrderedGroupFailoverState([]int64{22, 11, 22, 0, -1, 33}, 3)

	require.Equal(t, []int64{22, 11, 33}, state.GroupIDs)
	require.Equal(t, int64(22), mustCurrentGroupID(t, state))
	require.Empty(t, state.FailedGroupIDs)
}

func TestOrderedGroupFailoverStateAdvanceUsesConfiguredOrder(t *testing.T) {
	state := NewOrderedGroupFailoverState([]int64{22, 11, 33}, 2)

	require.Equal(t, GroupFailoverContinue, state.Advance(context.Background(), nil, false))
	require.Equal(t, int64(11), mustCurrentGroupID(t, state))
	require.Equal(t, 1, state.SwitchCount)
	require.Contains(t, state.FailedGroupIDs, int64(22))

	require.Equal(t, GroupFailoverContinue, state.Advance(context.Background(), nil, false))
	require.Equal(t, int64(33), mustCurrentGroupID(t, state))
	require.Equal(t, 2, state.SwitchCount)

	require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), nil, false))
	require.Equal(t, 2, state.SwitchCount)
	require.Contains(t, state.FailedGroupIDs, int64(33))
}

func TestOrderedGroupFailoverStateAdvanceRejectsNonRetryableError(t *testing.T) {
	state := NewOrderedGroupFailoverState([]int64{11, 22}, 1)
	err := &service.UpstreamFailoverError{
		NextAccountAction: service.NextAccountStop,
	}

	require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), err, false))
	require.Equal(t, int64(11), mustCurrentGroupID(t, state))
	require.Zero(t, state.SwitchCount)
	require.Equal(t, err, state.LastFailoverErr)
}

func TestOrderedGroupFailoverStateAdvanceRejectsAfterResponseWrite(t *testing.T) {
	state := NewOrderedGroupFailoverState([]int64{11, 22}, 1)
	err := &service.UpstreamFailoverError{}

	require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), err, true))
	require.Equal(t, int64(11), mustCurrentGroupID(t, state))
	require.Zero(t, state.SwitchCount)
	require.Empty(t, state.FailedGroupIDs)
}

func TestOrderedGroupFailoverStateAdvanceStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := NewOrderedGroupFailoverState([]int64{11, 22}, 1)

	require.Equal(t, GroupFailoverCanceled, state.Advance(ctx, nil, false))
	require.Equal(t, int64(11), mustCurrentGroupID(t, state))
	require.Zero(t, state.SwitchCount)
}

func TestOrderedGroupFailoverStateAdvanceHonorsMaxSwitches(t *testing.T) {
	state := NewOrderedGroupFailoverState([]int64{11, 22, 33}, 1)

	require.Equal(t, GroupFailoverContinue, state.Advance(context.Background(), nil, false))
	require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), nil, false))
	require.Equal(t, int64(22), mustCurrentGroupID(t, state))
	require.Equal(t, 1, state.SwitchCount)
}

func mustCurrentGroupID(t *testing.T, state *OrderedGroupFailoverState) int64 {
	t.Helper()
	groupID, ok := state.CurrentGroupID()
	require.True(t, ok)
	return groupID
}
