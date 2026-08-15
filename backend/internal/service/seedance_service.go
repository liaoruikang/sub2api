package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type SeedanceUpstreamResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type SeedanceService struct {
	repo              SeedanceRepository
	accountRepo       AccountRepository
	gateway           *GatewayService
	httpUpstream      HTTPUpstream
	tlsProfileService *TLSFingerprintProfileService
	usageService      *OpenAIGatewayService
}

func NewSeedanceService(
	repo SeedanceRepository,
	accountRepo AccountRepository,
	gateway *GatewayService,
	httpUpstream HTTPUpstream,
	tlsProfileService *TLSFingerprintProfileService,
	usageService *OpenAIGatewayService,
) *SeedanceService {
	return &SeedanceService{
		repo: repo, accountRepo: accountRepo, gateway: gateway,
		httpUpstream: httpUpstream, tlsProfileService: tlsProfileService,
		usageService: usageService,
	}
}

func (s *SeedanceService) Repository() SeedanceRepository { return s.repo }

func (s *SeedanceService) UsageService() *OpenAIGatewayService { return s.usageService }

func (s *SeedanceService) SelectAccount(
	ctx context.Context,
	groupID *int64,
	model string,
	excluded map[int64]struct{},
	userID int64,
) (*AccountSelectionResult, error) {
	if s == nil || s.gateway == nil {
		return nil, fmt.Errorf("seedance scheduler is unavailable")
	}
	return s.gateway.SelectAccountWithLoadAwareness(ctx, groupID, "", model, excluded, "", userID)
}

func (s *SeedanceService) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, fmt.Errorf("seedance account repository is unavailable")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !account.IsSeedance() || account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("bound account is not a Seedance API Key account")
	}
	return account, nil
}

func (s *SeedanceService) Forward(
	ctx context.Context,
	account *Account,
	method, path, rawQuery string,
	body []byte,
	inboundHeader http.Header,
) (*SeedanceUpstreamResponse, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, fmt.Errorf("seedance HTTP upstream is unavailable")
	}
	if account == nil || !account.IsSeedance() {
		return nil, fmt.Errorf("invalid Seedance account")
	}
	if err := ValidateSeedanceAccount(account.Platform, account.Type, account.Credentials); err != nil {
		return nil, err
	}
	base, err := url.Parse(account.GetSeedanceBaseURL())
	if err != nil {
		return nil, fmt.Errorf("parse Seedance Base URL: %w", err)
	}
	relative, err := url.Parse(path)
	if err != nil || !strings.HasPrefix(relative.Path, "/v1/") {
		return nil, fmt.Errorf("invalid Seedance upstream path")
	}
	base.Path = strings.TrimRight(base.Path, "/") + relative.Path
	base.RawQuery = rawQuery
	req, err := http.NewRequestWithContext(ctx, method, base.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(account.GetCredential("api_key")))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if accept := strings.TrimSpace(inboundHeader.Get("Accept")); accept != "" {
		req.Header.Set("Accept", accept)
	}
	if userAgent := strings.TrimSpace(inboundHeader.Get("User-Agent")); userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var profile *tlsfingerprint.Profile
	if s.tlsProfileService != nil {
		profile = s.tlsProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, profile)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Seedance response: %w", err)
	}
	return &SeedanceUpstreamResponse{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: responseBody}, nil
}
