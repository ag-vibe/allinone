package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

func TestTodoService_ListTodos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(1)

	tests := []struct {
		name    string
		setup   func(m *model.MockModelInterface)
		wantErr bool
	}{
		{
			name: "ensure user fails",
			setup: func(m *model.MockModelInterface) {
				m.EXPECT().EnsureUser(ctx, userID).Return(errors.New("ensure user error"))
			},
			wantErr: true,
		},
		{
			name: "no todos",
			setup: func(m *model.MockModelInterface) {
				m.EXPECT().EnsureUser(ctx, userID).Return(nil)
				m.EXPECT().NormalizeTodayToWeek(ctx, userID).Return(nil)
				m.EXPECT().NormalizeWeekToLater(ctx, userID).Return(nil)
				m.EXPECT().ListTodosByUser(ctx, userID).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "some todos",
			setup: func(m *model.MockModelInterface) {
				m.EXPECT().EnsureUser(ctx, userID).Return(nil)
				m.EXPECT().NormalizeTodayToWeek(ctx, userID).Return(nil)
				m.EXPECT().NormalizeWeekToLater(ctx, userID).Return(nil)
				m.EXPECT().ListTodosByUser(ctx, userID).Return([]*querier.TodoItem{
					{ID: uuid.New(), UserID: userID, Title: "a"},
					{ID: uuid.New(), UserID: userID, Title: "b"},
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := model.NewMockModelInterface(ctrl)
			if tt.setup != nil {
				tt.setup(m)
			}

			svc := NewTodoService(m)
			_, err := svc.ListTodos(ctx, userID)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTodoService_UpdateTodoPersistsDescriptionAndBucket(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(1)
	id := uuid.New()
	title := "new title"
	bucket := "week"
	description := "  trimmed description  "

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterface(ctrl)

	m.EXPECT().EnsureUser(ctx, userID).Return(nil)
	m.EXPECT().NormalizeTodayToWeek(ctx, userID).Return(nil)
	m.EXPECT().NormalizeWeekToLater(ctx, userID).Return(nil)
	m.EXPECT().ListTodosByUser(ctx, userID).Return([]*querier.TodoItem{
		{
			ID:          id,
			UserID:      userID,
			Title:       "old title",
			Description: "old description",
			Done:        false,
			Bucket:      "later",
			CreatedAt:   time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		},
	}, nil)
	m.EXPECT().UpdateTodo(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.UpdateTodoParams) (*querier.TodoItem, error) {
		if arg.ID != id {
			t.Fatalf("unexpected id: %v", arg.ID)
		}
		if arg.UserID != userID {
			t.Fatalf("unexpected user id: %d", arg.UserID)
		}
		if arg.Title != title {
			t.Fatalf("unexpected title: %q", arg.Title)
		}
		if arg.Description != "trimmed description" {
			t.Fatalf("unexpected description: %q", arg.Description)
		}
		if arg.Done {
			t.Fatalf("expected todo to remain not done")
		}
		if arg.DoneAt != nil {
			t.Fatalf("expected done_at to remain nil")
		}

		return &querier.TodoItem{
			ID:          id,
			UserID:      userID,
			Title:       arg.Title,
			Description: arg.Description,
			Done:        arg.Done,
			DoneAt:      arg.DoneAt,
			Bucket:      "later",
			CreatedAt:   time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		}, nil
	})
	m.EXPECT().UpdateTodoBucket(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.UpdateTodoBucketParams) error {
		if arg.ID != id {
			t.Fatalf("unexpected bucket update id: %v", arg.ID)
		}
		if arg.UserID != userID {
			t.Fatalf("unexpected bucket update user id: %d", arg.UserID)
		}
		if arg.Bucket != bucket {
			t.Fatalf("unexpected bucket: %q", arg.Bucket)
		}
		if arg.PlannedForDay.Valid {
			t.Fatalf("expected planned_for_day to be null")
		}
		if !arg.PlannedForWeek.Valid {
			t.Fatalf("expected planned_for_week to be set")
		}
		return nil
	})

	svc := NewTodoService(m)
	item, err := svc.UpdateTodo(ctx, userID, id, &title, nil, &bucket, &description)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Description != "trimmed description" {
		t.Fatalf("unexpected returned description: %q", item.Description)
	}
	if item.Bucket != bucket {
		t.Fatalf("unexpected returned bucket: %q", item.Bucket)
	}
	if !item.PlannedForWeek.Valid {
		t.Fatalf("expected returned planned_for_week to be set")
	}
	if item.PlannedForDay.Valid {
		t.Fatalf("expected returned planned_for_day to be null")
	}
}

func TestTodoService_UpdateTodoSetsDoneAtWhenCompleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := int32(1)
	id := uuid.New()
	done := true

	ctrl := gomock.NewController(t)
	m := model.NewMockModelInterface(ctrl)

	m.EXPECT().EnsureUser(ctx, userID).Return(nil)
	m.EXPECT().NormalizeTodayToWeek(ctx, userID).Return(nil)
	m.EXPECT().NormalizeWeekToLater(ctx, userID).Return(nil)
	m.EXPECT().ListTodosByUser(ctx, userID).Return([]*querier.TodoItem{
		{
			ID:          id,
			UserID:      userID,
			Title:       "old title",
			Description: "old description",
			Done:        false,
			DoneAt:      nil,
			Bucket:      "today",
			CreatedAt:   time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		},
	}, nil)
	m.EXPECT().UpdateTodo(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, arg querier.UpdateTodoParams) (*querier.TodoItem, error) {
		if !arg.Done {
			t.Fatalf("expected todo to be marked done")
		}
		if arg.DoneAt == nil {
			t.Fatalf("expected done_at to be set")
		}
		return &querier.TodoItem{
			ID:          id,
			UserID:      userID,
			Title:       arg.Title,
			Description: arg.Description,
			Done:        arg.Done,
			DoneAt:      arg.DoneAt,
			Bucket:      "today",
			CreatedAt:   time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		}, nil
	})
	m.EXPECT().UpdateTodoBucket(ctx, gomock.Any()).Return(nil)

	svc := NewTodoService(m)
	item, err := svc.UpdateTodo(ctx, userID, id, nil, &done, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !item.Done {
		t.Fatalf("expected returned todo to be done")
	}
	if item.DoneAt == nil {
		t.Fatalf("expected returned done_at to be set")
	}
}

func TestMapTodoToAPIIncludesDescription(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	doneAt := time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)
	item := &querier.TodoItem{
		ID:          uuid.New(),
		Title:       "todo title",
		Description: "todo description",
		Done:        true,
		DoneAt:      &doneAt,
		Bucket:      "today",
		CreatedAt:   now,
		PlannedForDay: pgtype.Date{
			Time:  now,
			Valid: true,
		},
	}

	got := mapTodoToAPI(item)
	if got.Description != item.Description {
		t.Fatalf("expected description %q, got %q", item.Description, got.Description)
	}
	if got.Title != item.Title {
		t.Fatalf("expected title %q, got %q", item.Title, got.Title)
	}
	if got.Bucket != "today" {
		t.Fatalf("expected bucket today, got %q", got.Bucket)
	}
	if got.DoneAt == nil || !got.DoneAt.Equal(doneAt) {
		t.Fatalf("expected doneAt %v, got %v", doneAt, got.DoneAt)
	}
}
