package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GroupFailoverAction 表示有序分组 failover 的下一步动作。
type GroupFailoverAction int

const (
	GroupFailoverContinue GroupFailoverAction = iota
	GroupFailoverExhausted
	GroupFailoverCanceled
)

// OrderedGroupFailoverState 管理 API Key 分组的配置顺序和请求级切换预算。
// 账号级排除集合和重试状态仍由 FailoverState 独立维护。
type OrderedGroupFailoverState struct {
	GroupIDs        []int64
	CurrentIndex    int
	FailedGroupIDs  map[int64]struct{}
	SwitchCount     int
	MaxSwitches     int
	LastFailoverErr *service.UpstreamFailoverError
	requestBudget   *RequestFailoverBudget
}

// NewOrderedGroupFailoverState 创建有序分组 failover 状态。
func NewOrderedGroupFailoverState(groupIDs []int64, maxSwitches int) *OrderedGroupFailoverState {
	return NewOrderedGroupFailoverStateWithBudget(groupIDs, maxSwitches, nil)
}

func NewOrderedGroupFailoverStateWithBudget(groupIDs []int64, maxSwitches int, budget *RequestFailoverBudget) *OrderedGroupFailoverState {
	ordered := make([]int64, 0, len(groupIDs))
	seen := make(map[int64]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		ordered = append(ordered, groupID)
	}
	return &OrderedGroupFailoverState{
		GroupIDs:       ordered,
		FailedGroupIDs: make(map[int64]struct{}),
		MaxSwitches:    maxSwitches,
		requestBudget:  budget,
	}
}

// CurrentGroupID 返回当前候选分组。
func (s *OrderedGroupFailoverState) CurrentGroupID() (int64, bool) {
	if s == nil || s.CurrentIndex < 0 || s.CurrentIndex >= len(s.GroupIDs) {
		return 0, false
	}
	return s.GroupIDs[s.CurrentIndex], true
}

// Advance 在当前分组安全耗尽时推进到下一个配置分组。
// failoverErr 为 nil 时表示当前分组没有可用账号；非 nil 时必须明确允许 failover。
// 一旦客户端已经收到语义响应，禁止切换分组。
func (s *OrderedGroupFailoverState) Advance(
	ctx context.Context,
	failoverErr *service.UpstreamFailoverError,
	responseWritten bool,
) GroupFailoverAction {
	if ctx != nil && ctx.Err() != nil {
		return GroupFailoverCanceled
	}
	if responseWritten {
		return GroupFailoverExhausted
	}
	if s == nil {
		return GroupFailoverExhausted
	}
	if failoverErr != nil {
		s.LastFailoverErr = failoverErr
		if !failoverErr.ShouldRetryNextAccount() {
			return GroupFailoverExhausted
		}
	}
	currentGroupID, ok := s.CurrentGroupID()
	if !ok {
		return GroupFailoverExhausted
	}
	s.FailedGroupIDs[currentGroupID] = struct{}{}
	if s.SwitchCount >= s.MaxSwitches {
		return GroupFailoverExhausted
	}
	for next := s.CurrentIndex + 1; next < len(s.GroupIDs); next++ {
		if _, failed := s.FailedGroupIDs[s.GroupIDs[next]]; failed {
			continue
		}
		if s.requestBudget != nil && !s.requestBudget.TryConsume() {
			return GroupFailoverExhausted
		}
		s.CurrentIndex = next
		s.SwitchCount++
		return GroupFailoverContinue
	}
	s.CurrentIndex = len(s.GroupIDs)
	return GroupFailoverExhausted
}
