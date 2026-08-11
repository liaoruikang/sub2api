package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRequestFailoverBudgetSharedAcrossAccountAndGroupSwitches(t *testing.T) {
	budget := NewRequestFailoverBudget(2)
	mock := &mockTempUnscheduler{}
	accountState := NewFailoverStateWithBudget(10, false, budget)
	groupState := NewOrderedGroupFailoverStateWithBudget([]int64{11, 22, 33}, 2, budget)

	err := newTestFailoverErr(500, false, false)
	require.Equal(t, FailoverContinue, accountState.HandleFailoverError(
		context.Background(), mock, 101, service.PlatformOpenAI, maxSameAccountRetries, err,
	))
	require.Equal(t, 1, budget.SwitchCount)

	require.Equal(t, GroupFailoverContinue, groupState.Advance(context.Background(), nil, false))
	require.Equal(t, 2, budget.SwitchCount)
	require.Equal(t, int64(22), mustCurrentGroupID(t, groupState))

	require.Equal(t, FailoverExhausted, accountState.HandleFailoverError(
		context.Background(), mock, 102, service.PlatformOpenAI, maxSameAccountRetries, err,
	))
	require.Equal(t, 1, accountState.SwitchCount)
	require.Equal(t, 2, budget.SwitchCount)
	require.Contains(t, accountState.FailedAccountIDs, int64(102))

	require.Equal(t, GroupFailoverExhausted, groupState.Advance(context.Background(), nil, false))
	require.Equal(t, int64(22), mustCurrentGroupID(t, groupState))
	require.Equal(t, 1, groupState.SwitchCount)
}

func TestRequestFailoverBudgetNonSwitchActionsDoNotConsume(t *testing.T) {
	t.Run("same account retry", func(t *testing.T) {
		budget := NewRequestFailoverBudget(1)
		state := NewFailoverStateWithBudget(3, false, budget)
		mock := &mockTempUnscheduler{}

		require.Equal(t, FailoverContinue, state.HandleFailoverError(
			context.Background(), mock, 101, service.PlatformOpenAI, maxSameAccountRetries,
			newTestFailoverErr(429, true, false),
		))
		require.Zero(t, budget.SwitchCount)
		require.Zero(t, state.SwitchCount)
		require.Equal(t, 1, state.SameAccountRetryCount[101])
		require.Empty(t, state.FailedAccountIDs)
	})

	t.Run("profit veto", func(t *testing.T) {
		budget := NewRequestFailoverBudget(1)
		state := NewFailoverStateWithBudget(3, false, budget)

		require.Equal(t, FailoverContinue, state.RecordProfitVeto(101))
		require.Zero(t, budget.SwitchCount)
		require.Zero(t, state.SwitchCount)
		require.Contains(t, state.FailedAccountIDs, int64(101))
	})

	t.Run("no next group", func(t *testing.T) {
		budget := NewRequestFailoverBudget(1)
		state := NewOrderedGroupFailoverStateWithBudget([]int64{11}, 1, budget)

		require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), nil, false))
		require.Zero(t, budget.SwitchCount)
		require.Zero(t, state.SwitchCount)
		require.Equal(t, len(state.GroupIDs), state.CurrentIndex)
	})
}

func TestRequestFailoverBudgetGroupAdvanceGuardsStateOnExhaustion(t *testing.T) {
	budget := NewRequestFailoverBudget(0)
	state := NewOrderedGroupFailoverStateWithBudget([]int64{11, 22}, 1, budget)

	require.Equal(t, GroupFailoverExhausted, state.Advance(context.Background(), nil, false))
	require.Zero(t, budget.SwitchCount)
	require.Zero(t, state.SwitchCount)
	require.Equal(t, int64(11), mustCurrentGroupID(t, state))
	require.Contains(t, state.FailedGroupIDs, int64(11))
}

func TestRequestFailoverBudgetRejectsCanceledWrittenAndNonRetryableGroupAdvance(t *testing.T) {
	tests := []struct {
		name string
		call func(*OrderedGroupFailoverState) GroupFailoverAction
	}{
		{
			name: "canceled",
			call: func(state *OrderedGroupFailoverState) GroupFailoverAction {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return state.Advance(ctx, nil, false)
			},
		},
		{
			name: "response written",
			call: func(state *OrderedGroupFailoverState) GroupFailoverAction {
				return state.Advance(context.Background(), nil, true)
			},
		},
		{
			name: "non-retryable error",
			call: func(state *OrderedGroupFailoverState) GroupFailoverAction {
				return state.Advance(context.Background(), &service.UpstreamFailoverError{
					NextAccountAction: service.NextAccountStop,
				}, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewRequestFailoverBudget(1)
			state := NewOrderedGroupFailoverStateWithBudget([]int64{11, 22}, 1, budget)

			require.NotEqual(t, GroupFailoverContinue, tt.call(state))
			require.Zero(t, budget.SwitchCount)
			require.Zero(t, state.SwitchCount)
			require.Equal(t, int64(11), mustCurrentGroupID(t, state))
		})
	}
}

func TestRequestFailoverBudgetKeepsGroupLocalAccountFailuresIsolated(t *testing.T) {
	budget := NewRequestFailoverBudget(2)
	mock := &mockTempUnscheduler{}
	first := NewFailoverStateWithBudget(3, false, budget)
	second := NewFailoverStateWithBudget(3, false, budget)

	require.Equal(t, FailoverContinue, first.HandleFailoverError(
		context.Background(), mock, 101, service.PlatformOpenAI, maxSameAccountRetries,
		newTestFailoverErr(500, false, false),
	))
	require.Contains(t, first.FailedAccountIDs, int64(101))
	require.NotContains(t, second.FailedAccountIDs, int64(101))

	require.Equal(t, FailoverContinue, second.RecordProfitVeto(202))
	require.Contains(t, second.FailedAccountIDs, int64(202))
	require.NotContains(t, first.FailedAccountIDs, int64(202))
	require.Equal(t, 1, budget.SwitchCount)
}
