package deviceauth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudcarver/anclax/core"
	anclax_service "github.com/cloudcarver/anclax/pkg/service"
	anclaxapigen "github.com/cloudcarver/anclax/pkg/zgen/apigen"
	"go.uber.org/mock/gomock"

	"github.com/wibus-wee/allinone/pkg/config"
	"github.com/wibus-wee/allinone/pkg/zcore/model"
	appgen "github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

type stubAuthService struct {
	signInFn func(ctx context.Context, userID int32) (*anclaxapigen.Credentials, error)
}

func (s *stubAuthService) SignIn(ctx context.Context, userID int32) (*anclaxapigen.Credentials, error) {
	if s.signInFn != nil {
		return s.signInFn(ctx, userID)
	}
	return nil, errors.New("unexpected SignIn call")
}

func (s *stubAuthService) CreateNewUser(ctx context.Context, username, password string) (*anclax_service.UserMeta, error) {
	return nil, errors.New("unexpected CreateNewUser call")
}

func (s *stubAuthService) CreateNewUserWithTx(ctx context.Context, tx core.Tx, username, password string) (*anclax_service.UserMeta, error) {
	return nil, errors.New("unexpected CreateNewUserWithTx call")
}

func (s *stubAuthService) GetUserByUserName(ctx context.Context, username string) (*anclax_service.UserMeta, error) {
	return nil, errors.New("unexpected GetUserByUserName call")
}

func (s *stubAuthService) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	return false, errors.New("unexpected IsUsernameExists call")
}

func (s *stubAuthService) DeleteUserByName(ctx context.Context, username string) error {
	return errors.New("unexpected DeleteUserByName call")
}

func (s *stubAuthService) RestoreUserByName(ctx context.Context, username string) error {
	return errors.New("unexpected RestoreUserByName call")
}

func (s *stubAuthService) CreateTestAccount(ctx context.Context, username, password string) (int32, error) {
	return 0, errors.New("unexpected CreateTestAccount call")
}

func (s *stubAuthService) SignInWithPassword(ctx context.Context, params anclaxapigen.SignInRequest) (*anclaxapigen.Credentials, error) {
	return nil, errors.New("unexpected SignInWithPassword call")
}

func (s *stubAuthService) RefreshToken(ctx context.Context, refreshToken string) (*anclaxapigen.Credentials, error) {
	return nil, errors.New("unexpected RefreshToken call")
}

func (s *stubAuthService) ListTasks(ctx context.Context) ([]anclaxapigen.Task, error) {
	return nil, errors.New("unexpected ListTasks call")
}

func (s *stubAuthService) GetTaskByID(ctx context.Context, id int32) (*anclaxapigen.Task, error) {
	return nil, errors.New("unexpected GetTaskByID call")
}

func (s *stubAuthService) ListEvents(ctx context.Context) ([]anclaxapigen.Event, error) {
	return nil, errors.New("unexpected ListEvents call")
}

func (s *stubAuthService) ListOrgs(ctx context.Context, userID int32) ([]anclaxapigen.Org, error) {
	return nil, errors.New("unexpected ListOrgs call")
}

func (s *stubAuthService) UpdateUserPassword(ctx context.Context, username, password string) (int32, error) {
	return 0, errors.New("unexpected UpdateUserPassword call")
}

func (s *stubAuthService) TryExecuteTask(ctx context.Context, taskID int32) error {
	return errors.New("unexpected TryExecuteTask call")
}

func TestAuthorizeCreatesDeviceCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	cfg := testConfig()
	authSvc := &stubAuthService{}
	svc, err := NewService(cfg, m, authSvc)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	fixedNow := time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	ctx := context.Background()
	scope := "read:todos"
	meta := RequestMeta{IP: strPtr("127.0.0.1"), UserAgent: strPtr("tests")}

	m.EXPECT().CreateDeviceCode(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, arg querier.CreateDeviceCodeParams) (*querier.DeviceCode, error) {
			if arg.ClientID != "todo" {
				t.Fatalf("unexpected clientId: %s", arg.ClientID)
			}
			if arg.Status != statusPending {
				t.Fatalf("unexpected status: %s", arg.Status)
			}
			if arg.Scope == nil || *arg.Scope != scope {
				t.Fatalf("unexpected scope: %v", arg.Scope)
			}
			if len(arg.DeviceCodeHash) == 0 || len(arg.UserCodeHash) == 0 {
				t.Fatalf("hashes should not be empty")
			}
			if !arg.ExpiresAt.Equal(fixedNow.Add(10 * time.Minute)) {
				t.Fatalf("unexpected expiresAt: %v", arg.ExpiresAt)
			}
			if arg.PollIntervalSec != 7 {
				t.Fatalf("unexpected interval: %d", arg.PollIntervalSec)
			}
			if arg.Ip == nil || *arg.Ip != "127.0.0.1" {
				t.Fatalf("unexpected ip: %v", arg.Ip)
			}
			if arg.UserAgent == nil || *arg.UserAgent != "tests" {
				t.Fatalf("unexpected ua: %v", arg.UserAgent)
			}
			return &querier.DeviceCode{ID: 1}, nil
		},
	)

	resp, err := svc.Authorize(ctx, appgen.DeviceAuthorizeRequest{
		ClientId: "todo",
		Scope:    &scope,
	}, meta)
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if resp.DeviceCode == "" || resp.UserCode == "" {
		t.Fatalf("expected device/user code")
	}
	if !strings.Contains(resp.UserCode, "-") {
		t.Fatalf("expected user code with dash, got %s", resp.UserCode)
	}
	if resp.VerificationUri != "/api/v1/auth/device/verify" {
		t.Fatalf("unexpected verification uri: %s", resp.VerificationUri)
	}
	if !strings.Contains(resp.VerificationUriComplete, resp.UserCode) {
		t.Fatalf("verification uri should contain user code")
	}
	if resp.ExpiresIn != int32(600) || resp.Interval != int32(7) {
		t.Fatalf("unexpected expires/interval: %d/%d", resp.ExpiresIn, resp.Interval)
	}
}

func TestApproveHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	svc, _ := NewService(testConfig(), m, &stubAuthService{})
	fixedNow := time.Date(2026, time.March, 31, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	userCode := "ABCD-EFGH"
	normalized := normalizeUserCode(userCode)
	hash := hashToken([]byte(testConfig().DeviceCode.Secret), normalized)

	m.EXPECT().GetDeviceCodeByUserHash(gomock.Any(), hash).Return(&querier.DeviceCode{
		ID:        7,
		Status:    statusPending,
		ExpiresAt: fixedNow.Add(2 * time.Minute),
	}, nil)

	uid := int32(42)
	m.EXPECT().ApproveDeviceCode(gomock.Any(), querier.ApproveDeviceCodeParams{
		ID:     7,
		UserID: &uid,
	}).Return(&querier.DeviceCode{ID: 7, Status: statusApproved}, nil)

	resp, err := svc.Approve(context.Background(), userCode, uid)
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if resp.Status != "approved" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
}

func TestApproveExpired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	svc, _ := NewService(testConfig(), m, &stubAuthService{})
	fixedNow := time.Date(2026, time.March, 31, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	userCode := "WXYZ-1234"
	hash := hashToken([]byte(testConfig().DeviceCode.Secret), normalizeUserCode(userCode))

	m.EXPECT().GetDeviceCodeByUserHash(gomock.Any(), hash).Return(&querier.DeviceCode{
		ID:        9,
		Status:    statusPending,
		ExpiresAt: fixedNow.Add(-1 * time.Minute),
	}, nil)
	m.EXPECT().UpdateDeviceCodeStatus(gomock.Any(), querier.UpdateDeviceCodeStatusParams{
		ID:     9,
		Status: statusExpired,
		UserID: nil,
	}).Return(&querier.DeviceCode{ID: 9, Status: statusExpired}, nil)

	if _, err := svc.Approve(context.Background(), userCode, 1); err == nil {
		t.Fatalf("expected error for expired code")
	}
}

func TestTokenPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	svc, _ := NewService(testConfig(), m, &stubAuthService{})
	fixedNow := time.Date(2026, time.March, 31, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	deviceCode := "device-code"
	hash := hashToken([]byte(testConfig().DeviceCode.Secret), deviceCode)

	m.EXPECT().GetDeviceCodeByDeviceHash(gomock.Any(), hash).Return(&querier.DeviceCode{
		ID:              11,
		Status:          statusPending,
		ExpiresAt:       fixedNow.Add(5 * time.Minute),
		PollIntervalSec: 5,
	}, nil)
	m.EXPECT().TouchDeviceCodePoll(gomock.Any(), int64(11)).Return(&querier.DeviceCode{ID: 11}, nil)

	resp, err := svc.Token(context.Background(), deviceCode)
	if err != nil {
		t.Fatalf("Token error: %v", err)
	}
	if resp.Error == nil || *resp.Error != "authorization_pending" {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestTokenSlowDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	svc, _ := NewService(testConfig(), m, &stubAuthService{})
	fixedNow := time.Date(2026, time.March, 31, 10, 0, 10, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	deviceCode := "device-code"
	hash := hashToken([]byte(testConfig().DeviceCode.Secret), deviceCode)
	lastPoll := fixedNow.Add(-2 * time.Second)

	m.EXPECT().GetDeviceCodeByDeviceHash(gomock.Any(), hash).Return(&querier.DeviceCode{
		ID:              12,
		Status:          statusPending,
		ExpiresAt:       fixedNow.Add(5 * time.Minute),
		LastPollAt:      &lastPoll,
		PollIntervalSec: 5,
	}, nil)
	m.EXPECT().TouchDeviceCodePoll(gomock.Any(), int64(12)).Return(&querier.DeviceCode{ID: 12}, nil)

	resp, err := svc.Token(context.Background(), deviceCode)
	if err != nil {
		t.Fatalf("Token error: %v", err)
	}
	if resp.Error == nil || *resp.Error != "slow_down" {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestTokenApprovedReturnsCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := model.NewMockModelInterface(ctrl)
	authSvc := &stubAuthService{}
	svc, _ := NewService(testConfig(), m, authSvc)
	fixedNow := time.Date(2026, time.March, 31, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	deviceCode := "device-code"
	hash := hashToken([]byte(testConfig().DeviceCode.Secret), deviceCode)
	uid := int32(99)

	m.EXPECT().GetDeviceCodeByDeviceHash(gomock.Any(), hash).Return(&querier.DeviceCode{
		ID:              13,
		Status:          statusApproved,
		UserID:          &uid,
		ExpiresAt:       fixedNow.Add(5 * time.Minute),
		PollIntervalSec: 5,
	}, nil)
	m.EXPECT().ConsumeDeviceCodeIfApproved(gomock.Any(), int64(13)).Return(&querier.DeviceCode{ID: 13, Status: statusConsumed}, nil)

	authSvc.signInFn = func(_ context.Context, userID int32) (*anclaxapigen.Credentials, error) {
		if userID != uid {
			return nil, errors.New("unexpected user id")
		}
		return &anclaxapigen.Credentials{
			AccessToken:  "access",
			RefreshToken: "refresh",
			TokenType:    anclaxapigen.Bearer,
		}, nil
	}

	resp, err := svc.Token(context.Background(), deviceCode)
	if err != nil {
		t.Fatalf("Token error: %v", err)
	}
	if resp.AccessToken == nil || *resp.AccessToken != "access" {
		t.Fatalf("unexpected access token: %v", resp.AccessToken)
	}
	if resp.RefreshToken == nil || *resp.RefreshToken != "refresh" {
		t.Fatalf("unexpected refresh token: %v", resp.RefreshToken)
	}
	if resp.TokenType == nil || *resp.TokenType != "Bearer" {
		t.Fatalf("unexpected token type: %v", resp.TokenType)
	}
}

func testConfig() *config.Config {
	exp := 10 * time.Minute
	interval := 7 * time.Second
	return &config.Config{
		DeviceCode: config.DeviceCodeConfig{
			Secret:       "test-secret",
			ExpiresIn:    &exp,
			PollInterval: &interval,
		},
	}
}
