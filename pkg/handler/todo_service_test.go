package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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
				m.EXPECT().ListTodosByUser(ctx, userID).Return(nil, nil)
			},
			wantErr: false,
		},
		{
			name: "some todos",
			setup: func(m *model.MockModelInterface) {
				m.EXPECT().EnsureUser(ctx, userID).Return(nil)
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
