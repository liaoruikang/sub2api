package service

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAISessionControlExtra(t *testing.T) {
	t.Run("nil extra defaults to disabled", func(t *testing.T) {
		extra, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeOAuth, nil)
		require.NoError(t, err)
		require.Equal(t, false, extra[OpenAISessionControlEnabledExtraKey])
	})

	t.Run("enabled defaults", func(t *testing.T) {
		extra, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
			OpenAISessionControlEnabledExtraKey: true,
		})
		require.NoError(t, err)
		require.Equal(t, DefaultOpenAISessionMaxCount, extra[OpenAISessionMaxCountExtraKey])
		require.Equal(t, DefaultOpenAISessionIdleTimeoutSeconds, extra[OpenAISessionIdleTimeoutSecondsExtraKey])
		require.Equal(t, false, extra[OpenAISessionSlotRotationEnabledExtraKey])
	})

	t.Run("disabled removes inactive values", func(t *testing.T) {
		extra, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
			OpenAISessionControlEnabledExtraKey:      false,
			OpenAISessionMaxCountExtraKey:            3,
			OpenAISessionIdleTimeoutSecondsExtraKey:  60,
			OpenAISessionSlotRotationEnabledExtraKey: true,
		})
		require.NoError(t, err)
		require.Equal(t, false, extra[OpenAISessionControlEnabledExtraKey])
		require.NotContains(t, extra, OpenAISessionMaxCountExtraKey)
		require.NotContains(t, extra, OpenAISessionIdleTimeoutSecondsExtraKey)
		require.NotContains(t, extra, OpenAISessionSlotRotationEnabledExtraKey)
	})

	t.Run("minimum count", func(t *testing.T) {
		_, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
			OpenAISessionControlEnabledExtraKey: true,
			OpenAISessionMaxCountExtraKey:       MinimumOpenAISessionMaxCount - 1,
		})
		require.ErrorContains(t, err, "greater than or equal")
	})

	t.Run("only OpenAI OAuth", func(t *testing.T) {
		_, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeAPIKey, map[string]any{
			OpenAISessionControlEnabledExtraKey: true,
		})
		require.ErrorContains(t, err, "only available")
	})

	t.Run("duration overflow", func(t *testing.T) {
		_, err := NormalizeOpenAISessionControlExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{
			OpenAISessionControlEnabledExtraKey:     true,
			OpenAISessionIdleTimeoutSecondsExtraKey: strconv.FormatInt(maximumOpenAISessionIdleTimeoutSeconds+1, 10),
		})
		require.ErrorContains(t, err, "supported duration range")
	})
}

func TestOpenAISessionControlAccountGetters(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			OpenAISessionControlEnabledExtraKey:      true,
			OpenAISessionMaxCountExtraKey:            float64(7),
			OpenAISessionIdleTimeoutSecondsExtraKey:  float64(90),
			OpenAISessionSlotRotationEnabledExtraKey: true,
		},
	}
	require.True(t, account.IsOpenAISessionControlEnabled())
	require.Equal(t, 7, account.GetOpenAISessionMaxCount())
	require.Equal(t, 90*time.Second, account.GetOpenAISessionIdleTimeout())
	require.True(t, account.IsOpenAISessionSlotRotationEnabled())
}

func TestMergeAndValidateOpenAISessionControlUpdate(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			OpenAISessionControlEnabledExtraKey:      true,
			OpenAISessionMaxCountExtraKey:            5,
			OpenAISessionIdleTimeoutSecondsExtraKey:  60,
			OpenAISessionSlotRotationEnabledExtraKey: false,
		},
	}
	require.Error(t, MergeAndValidateOpenAISessionControlUpdate(account, map[string]any{
		OpenAISessionMaxCountExtraKey: 2,
	}))
	require.NoError(t, MergeAndValidateOpenAISessionControlUpdate(account, map[string]any{
		OpenAISessionMaxCountExtraKey:            8,
		OpenAISessionSlotRotationEnabledExtraKey: true,
	}))
}
