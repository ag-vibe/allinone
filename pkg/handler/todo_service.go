package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

// TodoService defines the operations supported by the TODO domain.
type TodoService interface {
	ListTodos(ctx context.Context, userID int32) ([]*querier.TodoItem, error)
	CreateTodo(ctx context.Context, userID int32, title string) (*querier.TodoItem, error)
	UpdateTodo(ctx context.Context, userID int32, id uuid.UUID, title *string, done *bool, bucket *string, description *string) (*querier.TodoItem, error)
	DeleteTodo(ctx context.Context, userID int32, id uuid.UUID) error
}

// ErrTodoNotFound indicates that the requested TODO item does not exist.
var ErrTodoNotFound = errors.New("todo not found")

// todoService is the concrete implementation of TodoService.
type todoService struct {
	model model.ModelInterface
}

// NewTodoService creates a new TodoService.
func NewTodoService(m model.ModelInterface) TodoService {
	return &todoService{model: m}
}

func (s *todoService) ensureUser(ctx context.Context, userID int32) error {
	return s.model.EnsureUser(ctx, userID)
}

func (s *todoService) normalizeBuckets(ctx context.Context, userID int32) error {
	if err := s.model.NormalizeTodayToWeek(ctx, userID); err != nil {
		return err
	}
	if err := s.model.NormalizeWeekToLater(ctx, userID); err != nil {
		return err
	}
	return nil
}

func (s *todoService) ListTodos(ctx context.Context, userID int32) ([]*querier.TodoItem, error) {
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.normalizeBuckets(ctx, userID); err != nil {
		return nil, err
	}
	return s.model.ListTodosByUser(ctx, userID)
}

func (s *todoService) CreateTodo(ctx context.Context, userID int32, title string) (*querier.TodoItem, error) {
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}

	id := uuid.New()
	params := querier.CreateTodoParams{
		ID:     id,
		UserID: userID,
		Title:  title,
		Bucket: "later",
	}

	return s.model.CreateTodo(ctx, params)
}

func (s *todoService) UpdateTodo(ctx context.Context, userID int32, id uuid.UUID, title *string, done *bool, bucket *string, description *string) (*querier.TodoItem, error) {
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}

	// Normalize before loading and updating so we work on the latest state.
	if err := s.normalizeBuckets(ctx, userID); err != nil {
		return nil, err
	}

	items, err := s.model.ListTodosByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var current *querier.TodoItem
	for _, item := range items {
		if item.ID == id {
			current = item
			break
		}
	}
	if current == nil {
		return nil, ErrTodoNotFound
	}

	newTitle := current.Title
	if title != nil {
		newTitle = *title
	}

	newDone := current.Done
	if done != nil {
		newDone = *done
	}

	newDescription := current.Description
	if description != nil {
		newDescription = strings.TrimSpace(*description)
	}

	newBucket := current.Bucket
	if bucket != nil && *bucket != "" {
		newBucket = *bucket
	}

	// Update main fields first.
	updateParams := querier.UpdateTodoParams{
		ID:          id,
		UserID:      userID,
		Title:       newTitle,
		Done:        newDone,
		Description: newDescription,
	}

	updated, err := s.model.UpdateTodo(ctx, updateParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTodoNotFound
		}
		return nil, err
	}

	// Now compute planned_for_day/week based on the new bucket and done state.
	var plannedDay pgtype.Date
	var plannedWeek pgtype.Date

	// Helper to mark a pgtype.Date as null.
	setNull := func(d *pgtype.Date) {
		*d = pgtype.Date{Valid: false}
	}

	setNull(&plannedDay)
	setNull(&plannedWeek)

	if !newDone {
		now := time.Now().UTC()
		weekStart := dateToPgType(startOfWeek(now))

		switch newBucket {
		case "today":
			plannedDay = dateToPgType(truncateToDate(now))
			plannedWeek = weekStart
		case "week":
			plannedWeek = weekStart
		case "later":
			// keep both null
		}
	}

	bucketParams := querier.UpdateTodoBucketParams{
		ID:             id,
		UserID:         userID,
		Bucket:         newBucket,
		PlannedForDay:  plannedDay,
		PlannedForWeek: plannedWeek,
	}

	if err := s.model.UpdateTodoBucket(ctx, bucketParams); err != nil {
		return nil, err
	}

	// Reflect the bucket/planned fields on the returned item.
	updated.Bucket = newBucket
	updated.Description = newDescription
	updated.PlannedForDay = plannedDay
	updated.PlannedForWeek = plannedWeek

	return updated, nil
}

func (s *todoService) DeleteTodo(ctx context.Context, userID int32, id uuid.UUID) error {
	if err := s.ensureUser(ctx, userID); err != nil {
		return err
	}
	_, err := s.model.DeleteTodo(ctx, querier.DeleteTodoParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTodoNotFound
	}
	return err
}

// truncateToDate returns a time at midnight UTC for the given time.
func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// startOfWeek returns the Monday of the week of t (UTC).
func startOfWeek(t time.Time) time.Time {
	// Go's Weekday: Sunday=0, Monday=1, ..., Saturday=6.
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	// We want Monday.
	delta := wd - 1
	return truncateToDate(t.AddDate(0, 0, -delta))
}

func dateToPgType(t time.Time) pgtype.Date {
	var d pgtype.Date
	_ = d.Scan(t)
	return d
}
