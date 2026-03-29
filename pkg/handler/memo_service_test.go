package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/mock/gomock"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

func TestMemoService_CreateMemo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(7)
	refID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterfaceWithTransaction(ctrl)
	var createdID uuid.UUID

	gomock.InOrder(
		m.EXPECT().EnsureUser(ctx, userID).Return(nil),
		m.EXPECT().GetMemoByID(ctx, querier.GetMemoByIDParams{ID: refID, UserID: userID}).Return(&querier.Memo{ID: refID, UserID: userID}, nil),
		m.EXPECT().CreateMemo(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.CreateMemoParams) (*querier.Memo, error) {
			createdID = arg.ID
			if arg.Content != "Hello #work [[memo:"+refID.String()+"]] #Go" {
				t.Fatalf("unexpected content: %q", arg.Content)
			}
			if arg.State != "active" {
				t.Fatalf("unexpected state: %q", arg.State)
			}
			return &querier.Memo{ID: arg.ID, UserID: arg.UserID, Content: arg.Content, Excerpt: arg.Excerpt, State: arg.State}, nil
		}),
	)
	gomock.InOrder(
		m.EXPECT().DeleteMemoTagsByMemo(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.DeleteMemoTagsByMemoParams) error {
			if arg.MemoID != createdID || arg.UserID != userID {
				t.Fatalf("unexpected delete tags params: %+v", arg)
			}
			return nil
		}),
		m.EXPECT().CreateMemoTag(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.CreateMemoTagParams) error {
			if arg.MemoID != createdID || arg.UserID != userID || arg.Tag != "go" {
				t.Fatalf("unexpected first tag params: %+v", arg)
			}
			return nil
		}),
		m.EXPECT().CreateMemoTag(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.CreateMemoTagParams) error {
			if arg.MemoID != createdID || arg.UserID != userID || arg.Tag != "work" {
				t.Fatalf("unexpected second tag params: %+v", arg)
			}
			return nil
		}),
		m.EXPECT().DeleteMemoRelationsBySource(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.DeleteMemoRelationsBySourceParams) error {
			if arg.SourceMemoID != createdID || arg.UserID != userID {
				t.Fatalf("unexpected delete relations params: %+v", arg)
			}
			return nil
		}),
		m.EXPECT().CreateMemoRelation(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.CreateMemoRelationParams) error {
			if arg.SourceMemoID != createdID || arg.TargetMemoID != refID || arg.UserID != userID {
				t.Fatalf("unexpected relation params: %+v", arg)
			}
			return nil
		}),
	)

	svc := NewMemoService(m)
	item, err := svc.CreateMemo(ctx, userID, "  Hello #work [[memo:"+refID.String()+"]] #Go  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "go" || item.Tags[1] != "work" {
		t.Fatalf("unexpected tags: %#v", item.Tags)
	}
	if len(item.References) != 1 || item.References[0] != refID {
		t.Fatalf("unexpected references: %#v", item.References)
	}
}

func TestMemoService_CreateMemo_InvalidReference(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(7)
	refID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterfaceWithTransaction(ctrl)
	m.EXPECT().EnsureUser(ctx, userID).Return(nil)
	m.EXPECT().GetMemoByID(ctx, querier.GetMemoByIDParams{ID: refID, UserID: userID}).Return(nil, pgx.ErrNoRows)

	svc := NewMemoService(m)
	_, err := svc.CreateMemo(ctx, userID, "hello [[memo:"+refID.String()+"]]")
	if !errors.Is(err, ErrInvalidMemoReference) {
		t.Fatalf("expected invalid reference error, got %v", err)
	}
}

func TestMemoService_UpdateMemo_Archive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(9)
	memoID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterfaceWithTransaction(ctrl)
	m.EXPECT().EnsureUser(ctx, userID).Return(nil)
	m.EXPECT().GetMemoByID(ctx, querier.GetMemoByIDParams{ID: memoID, UserID: userID}).Return(&querier.Memo{ID: memoID, UserID: userID, Content: "hello", State: "active"}, nil)
	m.EXPECT().UpdateMemo(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.UpdateMemoParams) (*querier.Memo, error) {
		if arg.State != "archived" {
			t.Fatalf("unexpected state: %q", arg.State)
		}
		if arg.ArchivedAt == nil || arg.ArchivedAt.IsZero() {
			t.Fatal("expected archived_at")
		}
		return &querier.Memo{ID: memoID, UserID: userID, Content: arg.Content, Excerpt: arg.Excerpt, State: arg.State, ArchivedAt: arg.ArchivedAt, UpdatedAt: time.Now().UTC()}, nil
	})
	m.EXPECT().DeleteMemoTagsByMemo(ctx, querier.DeleteMemoTagsByMemoParams{MemoID: memoID, UserID: userID}).Return(nil)
	m.EXPECT().DeleteMemoRelationsBySource(ctx, querier.DeleteMemoRelationsBySourceParams{SourceMemoID: memoID, UserID: userID}).Return(nil)

	state := "archived"
	svc := NewMemoService(m)
	item, err := svc.UpdateMemo(ctx, userID, memoID, nil, &state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Item.ArchivedAt == nil {
		t.Fatal("expected archived memo")
	}
}

func TestMemoService_DeleteMemo_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(5)
	memoID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterfaceWithTransaction(ctrl)
	m.EXPECT().EnsureUser(ctx, userID).Return(nil)
	m.EXPECT().DeleteAttachmentLinksByResource(ctx, querier.DeleteAttachmentLinksByResourceParams{UserID: userID, ResourceType: memoResourceType, ResourceID: memoID}).Return(nil)
	m.EXPECT().DeleteMemo(ctx, querier.DeleteMemoParams{ID: memoID, UserID: userID}).Return(uuid.Nil, pgx.ErrNoRows)

	svc := NewMemoService(m)
	err := svc.DeleteMemo(ctx, userID, memoID)
	if !errors.Is(err, ErrMemoNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
