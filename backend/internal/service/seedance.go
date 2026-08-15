package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrSeedanceResourceNotFound = errors.New("seedance resource not found")
	ErrSeedanceTaskNotFound     = errors.New("seedance task not found")
)

const (
	SeedanceResourceAssetGroup = "asset_group"
	SeedanceResourceAsset      = "asset"
	SeedanceResourceVideoTask  = "video_task"

	SeedanceChannelGroup  = "group"
	SeedanceChannelSD     = "sd"
	SeedanceChannelDoubao = "doubao"
)

var SeedanceModels = []string{
	"dreamina-seedance-2-0-260128",
	"dreamina-seedance-2-0-ep",
	"dreamina-seedance-2-0-fast-260128",
	"dreamina-seedance-2-0-fast-ep",
	"dreamina-seedance-2-0-mini-260615",
	"dreamina-seedance-2-0-mini-ep",
	"dreamina-seedance-2-5-ep",
	"dreamina-seedance-2-0-hc",
	"dreamina-seedance-2-0-fast-hc",
	"dreamina-seedance-2-0-mini-hc",
	"dreamina-seedance-2-5-hc",
	"doubao-seedance-2-0-260128-a",
}

var seedanceModelSet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(SeedanceModels))
	for _, model := range SeedanceModels {
		result[model] = struct{}{}
	}
	return result
}()

func IsSeedanceModel(model string) bool {
	_, ok := seedanceModelSet[strings.TrimSpace(model)]
	return ok
}

func ValidateSeedanceAccount(platform, accountType string, credentials map[string]any) error {
	if platform != PlatformSeedance {
		return nil
	}
	if accountType != AccountTypeAPIKey {
		return infraerrors.BadRequest("SEEDANCE_ACCOUNT_TYPE_INVALID", "Seedance accounts must use API Key authentication")
	}
	apiKey, _ := credentials["api_key"].(string)
	if strings.TrimSpace(apiKey) == "" {
		return infraerrors.BadRequest("SEEDANCE_API_KEY_REQUIRED", "Seedance API Key is required")
	}
	baseURL, _ := credentials["base_url"].(string)
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultSeedanceBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return infraerrors.BadRequest("SEEDANCE_BASE_URL_INVALID", "Seedance Base URL must be an HTTPS origin without credentials, query, or fragment")
	}
	return nil
}

type SeedanceResource struct {
	ID           int64
	ResourceID   string
	ResourceType string
	Channel      string
	UserID       int64
	APIKeyID     *int64
	GroupID      *int64
	AccountID    int64
	ParentID     string
	TaskID       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SeedanceVideoTask struct {
	ID               int64
	TaskID           string
	UserID           int64
	APIKeyID         *int64
	GroupID          *int64
	AccountID        int64
	Model            string
	Status           string
	DurationSeconds  float64
	Resolution       string
	RequestBody      json.RawMessage
	ResponseBody     json.RawMessage
	LastErrorCode    string
	LastErrorMessage string
	BilledAt         *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
}

type SeedanceVideoTaskList struct {
	Items []SeedanceVideoTask
	Total int64
}

type SeedanceRepository interface {
	CreateResource(ctx context.Context, resource *SeedanceResource) error
	GetResource(ctx context.Context, userID int64, apiKeyID *int64, resourceType, resourceID string) (*SeedanceResource, error)
	CreateTask(ctx context.Context, task *SeedanceVideoTask) error
	GetTask(ctx context.Context, userID int64, apiKeyID *int64, taskID string) (*SeedanceVideoTask, error)
	ListTasks(ctx context.Context, userID int64, apiKeyID *int64, page, pageSize int) (*SeedanceVideoTaskList, error)
	UpdateTask(ctx context.Context, task *SeedanceVideoTask) error
	ClaimTaskBilling(ctx context.Context, taskID string) (bool, error)
	ReleaseTaskBilling(ctx context.Context, taskID string) error
}

func SeedanceAssetIDFromURI(value string) (string, bool) {
	if !strings.HasPrefix(value, "asset://") {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimPrefix(value, "asset://"))
	return id, id != ""
}

func SeedanceTaskNotFound(taskID string) error {
	return fmt.Errorf("seedance task %q not found", taskID)
}
