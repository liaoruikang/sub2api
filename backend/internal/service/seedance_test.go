package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSeedanceAccount(t *testing.T) {
	valid := map[string]any{
		"api_key":  "sk-seedance-test",
		"base_url": DefaultSeedanceBaseURL,
	}
	require.NoError(t, ValidateSeedanceAccount(PlatformSeedance, AccountTypeAPIKey, valid))
	require.Error(t, ValidateSeedanceAccount(PlatformSeedance, AccountTypeOAuth, valid))
	require.Error(t, ValidateSeedanceAccount(PlatformSeedance, AccountTypeAPIKey, map[string]any{"api_key": ""}))
	require.Error(t, ValidateSeedanceAccount(PlatformSeedance, AccountTypeAPIKey, map[string]any{
		"api_key": "sk-test", "base_url": "https://example.test/api?token=leak",
	}))
	require.Error(t, ValidateSeedanceAccount(PlatformSeedance, AccountTypeAPIKey, map[string]any{
		"api_key": "sk-test", "base_url": "http://example.test",
	}))
	require.NoError(t, ValidateSeedanceAccount(PlatformOpenAI, AccountTypeOAuth, nil))
}

func TestSeedanceModelAndAssetChannelRules(t *testing.T) {
	require.True(t, IsSeedanceModel("dreamina-seedance-2-0-hc"))
	require.False(t, IsSeedanceModel("grok-imagine-video"))

	id, ok := SeedanceAssetIDFromURI("asset://asset-123")
	require.True(t, ok)
	require.Equal(t, "asset-123", id)
	_, ok = SeedanceAssetIDFromURI("https://example.test/asset-123")
	require.False(t, ok)
}
