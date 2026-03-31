package deviceauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/cloudcarver/anclax/pkg/logger"
	anclax_service "github.com/cloudcarver/anclax/pkg/service"
	"github.com/jackc/pgx/v5"

	"github.com/wibus-wee/allinone/pkg/config"
	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

var log = logger.NewLogAgent("device-auth")

const (
	defaultExpiresIn    = 10 * time.Minute
	defaultPollInterval = 5 * time.Second
	deviceCodeBytes     = 32
	userCodeChars       = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	userCodeLen         = 8
)

const (
	statusPending  = "pending"
	statusApproved = "approved"
	statusDenied   = "denied"
	statusExpired  = "expired"
	statusConsumed = "consumed"
)

type RequestMeta struct {
	IP        *string
	UserAgent *string
}

type Service struct {
	model        model.ModelInterface
	authService  anclax_service.ServiceInterface
	now          func() time.Time
	secret       []byte
	expiresIn    time.Duration
	pollInterval time.Duration
	verifyPath   string
}

func NewService(cfg *config.Config, m model.ModelInterface, authSvc anclax_service.ServiceInterface) (*Service, error) {
	expiresIn := defaultExpiresIn
	if cfg.DeviceCode.ExpiresIn != nil {
		expiresIn = *cfg.DeviceCode.ExpiresIn
	}
	pollInterval := defaultPollInterval
	if cfg.DeviceCode.PollInterval != nil {
		pollInterval = *cfg.DeviceCode.PollInterval
	}

	secret := []byte(cfg.DeviceCode.Secret)
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		log.Warn("device code secret not set; generated ephemeral secret")
	}

	return &Service{
		model:        m,
		authService:  authSvc,
		now:          time.Now,
		secret:       secret,
		expiresIn:    expiresIn,
		pollInterval: pollInterval,
		verifyPath:   "/api/v1/auth/device/verify",
	}, nil
}

func (s *Service) Authorize(ctx context.Context, req apigen.DeviceAuthorizeRequest, meta RequestMeta) (*apigen.DeviceAuthorizeResult, error) {
	if strings.TrimSpace(req.ClientId) == "" {
		return nil, fmt.Errorf("clientId is required")
	}

	deviceCode, err := generateDeviceCode()
	if err != nil {
		return nil, err
	}
	userCode, err := generateUserCode()
	if err != nil {
		return nil, err
	}

	normalizedUserCode := normalizeUserCode(userCode)
	deviceHash := hashToken(s.secret, deviceCode)
	userHash := hashToken(s.secret, normalizedUserCode)

	expiresAt := s.now().Add(s.expiresIn)
	intervalSec := int32(s.pollInterval.Seconds())
	if intervalSec <= 0 {
		intervalSec = int32(defaultPollInterval.Seconds())
	}

	_, err = s.model.CreateDeviceCode(ctx, querier.CreateDeviceCodeParams{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		ClientID:        req.ClientId,
		Scope:           req.Scope,
		Status:          statusPending,
		UserID:          nil,
		ExpiresAt:       expiresAt,
		PollIntervalSec: int32(intervalSec),
		Ip:              meta.IP,
		UserAgent:       meta.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	verificationURI := s.verifyPath
	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", s.verifyPath, userCode)

	return &apigen.DeviceAuthorizeResult{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationUri:         verificationURI,
		VerificationUriComplete: verificationURIComplete,
		ExpiresIn:               int32(s.expiresIn.Seconds()),
		Interval:                intervalSec,
	}, nil
}

func (s *Service) Approve(ctx context.Context, userCode string, userID int32) (*apigen.DeviceApproveResult, error) {
	normalized := normalizeUserCode(userCode)
	if normalized == "" {
		return nil, fmt.Errorf("userCode is required")
	}

	hash := hashToken(s.secret, normalized)
	record, err := s.model.GetDeviceCodeByUserHash(ctx, hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device code not found")
		}
		return nil, err
	}

	if s.now().After(record.ExpiresAt) {
		_, _ = s.model.UpdateDeviceCodeStatus(ctx, querier.UpdateDeviceCodeStatusParams{
			ID:     record.ID,
			Status: statusExpired,
			UserID: nil,
		})
		return nil, fmt.Errorf("device code expired")
	}

	switch record.Status {
	case statusDenied:
		return nil, fmt.Errorf("device code denied")
	case statusConsumed:
		return nil, fmt.Errorf("device code already used")
	case statusApproved:
		return &apigen.DeviceApproveResult{Status: "approved"}, nil
	}

	_, err = s.model.ApproveDeviceCode(ctx, querier.ApproveDeviceCodeParams{
		ID:     record.ID,
		UserID: &userID,
	})
	if err != nil {
		return nil, err
	}

	return &apigen.DeviceApproveResult{Status: "approved"}, nil
}

func (s *Service) Token(ctx context.Context, deviceCode string) (*apigen.DeviceTokenResult, error) {
	if strings.TrimSpace(deviceCode) == "" {
		return &apigen.DeviceTokenResult{Error: strPtr("invalid_request"), ErrorDescription: strPtr("deviceCode is required")}, nil
	}

	hash := hashToken(s.secret, deviceCode)
	record, err := s.model.GetDeviceCodeByDeviceHash(ctx, hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &apigen.DeviceTokenResult{Error: strPtr("expired_token"), ErrorDescription: strPtr("device code not found")}, nil
		}
		return nil, err
	}

	if s.now().After(record.ExpiresAt) {
		_, _ = s.model.UpdateDeviceCodeStatus(ctx, querier.UpdateDeviceCodeStatusParams{
			ID:     record.ID,
			Status: statusExpired,
			UserID: nil,
		})
		return &apigen.DeviceTokenResult{Error: strPtr("expired_token"), ErrorDescription: strPtr("device code expired")}, nil
	}

	if record.LastPollAt != nil {
		minInterval := time.Duration(record.PollIntervalSec) * time.Second
		if minInterval <= 0 {
			minInterval = s.pollInterval
		}
		if s.now().Sub(*record.LastPollAt) < minInterval {
			_, _ = s.model.TouchDeviceCodePoll(ctx, record.ID)
			return &apigen.DeviceTokenResult{Error: strPtr("slow_down"), ErrorDescription: strPtr("polling too quickly")}, nil
		}
	}

	switch record.Status {
	case statusPending:
		_, _ = s.model.TouchDeviceCodePoll(ctx, record.ID)
		return &apigen.DeviceTokenResult{Error: strPtr("authorization_pending")}, nil
	case statusDenied:
		return &apigen.DeviceTokenResult{Error: strPtr("access_denied")}, nil
	case statusConsumed:
		return &apigen.DeviceTokenResult{Error: strPtr("expired_token")}, nil
	case statusApproved:
		if record.UserID == nil {
			return &apigen.DeviceTokenResult{Error: strPtr("authorization_pending")}, nil
		}

		_, err := s.model.ConsumeDeviceCodeIfApproved(ctx, record.ID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return &apigen.DeviceTokenResult{Error: strPtr("expired_token")}, nil
			}
			return nil, err
		}

		credentials, err := s.authService.SignIn(ctx, *record.UserID)
		if err != nil {
			_, _ = s.model.UpdateDeviceCodeStatus(ctx, querier.UpdateDeviceCodeStatusParams{
				ID:     record.ID,
				Status: statusApproved,
				UserID: record.UserID,
			})
			return nil, err
		}

		return &apigen.DeviceTokenResult{
			AccessToken:  strPtr(credentials.AccessToken),
			RefreshToken: strPtr(credentials.RefreshToken),
			TokenType:    strPtr(string(credentials.TokenType)),
		}, nil
	default:
		return &apigen.DeviceTokenResult{Error: strPtr("invalid_request")}, nil
	}
}

func generateDeviceCode() (string, error) {
	buf := make([]byte, deviceCodeBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateUserCode() (string, error) {
	raw := make([]byte, userCodeLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 0; i < userCodeLen; i++ {
		idx := int(raw[i]) % len(userCodeChars)
		b.WriteByte(userCodeChars[idx])
		if i == 3 {
			b.WriteByte('-')
		}
	}
	return b.String(), nil
}

func normalizeUserCode(code string) string {
	var b strings.Builder
	for _, r := range code {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func hashToken(secret []byte, value string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(value))
	return h.Sum(nil)
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
