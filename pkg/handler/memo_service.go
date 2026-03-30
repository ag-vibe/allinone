package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

const memoResourceType = "memo"

type MemoService interface {
	ListMemos(ctx context.Context, userID int32, filter MemoListFilter) ([]*HydratedMemoSummary, error)
	GetMemo(ctx context.Context, userID int32, id uuid.UUID) (*HydratedMemo, error)
	CreateMemo(ctx context.Context, userID int32, content json.RawMessage, plainText string, excerpt string, tags []string, references []uuid.UUID) (*HydratedMemo, error)
	UpdateMemo(ctx context.Context, userID int32, id uuid.UUID, content json.RawMessage, plainText string, excerpt string, tags []string, references []uuid.UUID, state *string) (*HydratedMemo, error)
	DeleteMemo(ctx context.Context, userID int32, id uuid.UUID) error
	ListMemoBacklinks(ctx context.Context, userID int32, id uuid.UUID) ([]*HydratedMemoSummary, error)
	ListTags(ctx context.Context, userID int32, filter TagListFilter) ([]*querier.ListTagsRow, error)
}

type MemoListFilter struct {
	Query  string
	Tag    string
	State  string
	Limit  int32
	Offset int32
}

type TagListFilter struct {
	Query  string
	Limit  int32
	Offset int32
}

type HydratedMemo struct {
	Item       *memoItem
	Tags       []string
	References []uuid.UUID
}

type HydratedMemoSummary struct {
	Item *memoItem
	Tags []string
}

var (
	ErrMemoNotFound         = errors.New("memo not found")
	ErrInvalidMemoContent   = errors.New("memo content is required")
	ErrInvalidMemoState     = errors.New("invalid memo state")
	ErrInvalidMemoReference = errors.New("invalid memo reference")
)

type memoService struct {
	model model.ModelInterface
}

type memoItem struct {
	ID         uuid.UUID
	UserID     int32
	Content    json.RawMessage
	PlainText  string
	Excerpt    string
	State      string
	ArchivedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewMemoService(m model.ModelInterface) MemoService {
	return &memoService{model: m}
}

func (s *memoService) ListMemos(ctx context.Context, userID int32, filter MemoListFilter) ([]*HydratedMemoSummary, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}
	if err := validateMemoState(filter.State, true); err != nil {
		return nil, err
	}

	items, err := s.model.ListMemos(ctx, querier.ListMemosParams{
		UserID:  userID,
		Column2: filter.State,
		Column3: strings.TrimSpace(filter.Query),
		Column4: canonicalizeTag(filter.Tag),
		Limit:   clampLimit(filter.Limit),
		Offset:  clampOffset(filter.Offset),
	})
	if err != nil {
		return nil, err
	}
	memoItems := make([]*memoItem, 0, len(items))
	for _, item := range items {
		memoItems = append(memoItems, memoItemFromList(item))
	}
	return s.hydrateMemoSummaries(ctx, s.model, userID, memoItems)
}

func (s *memoService) GetMemo(ctx context.Context, userID int32, id uuid.UUID) (*HydratedMemo, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}

	item, err := s.getMemoByID(ctx, s.model, userID, id)
	if err != nil {
		return nil, err
	}
	return s.hydrateMemo(ctx, s.model, userID, item)
}

func (s *memoService) CreateMemo(ctx context.Context, userID int32, content json.RawMessage, plainText string, excerpt string, tags []string, references []uuid.UUID) (*HydratedMemo, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}

	if err := validateMemoContent(content); err != nil {
		return nil, err
	}
	normalizedPlainText := normalizeMemoText(plainText)
	normalizedExcerpt := normalizeMemoExcerpt(excerpt)
	normalizedTags := normalizeMemoTags(tags)
	normalizedReferences := normalizeMemoReferences(references)

	var created *querier.CreateMemoRow
	if err := s.model.RunTransaction(ctx, func(txModel model.ModelInterface) error {
		if err := s.validateMemoReferences(ctx, txModel, userID, normalizedReferences); err != nil {
			return err
		}

		item, err := txModel.CreateMemo(ctx, querier.CreateMemoParams{
			ID:         uuid.New(),
			UserID:     userID,
			Content:    content,
			PlainText:  normalizedPlainText,
			Excerpt:    normalizedExcerpt,
			State:      "active",
			ArchivedAt: nil,
		})
		if err != nil {
			return err
		}
		created = item
		return s.syncMemoDerivedData(ctx, txModel, userID, item.ID, normalizedTags, normalizedReferences)
	}); err != nil {
		return nil, err
	}

	return &HydratedMemo{Item: memoItemFromCreate(created), Tags: normalizedTags, References: normalizedReferences}, nil
}

func (s *memoService) UpdateMemo(ctx context.Context, userID int32, id uuid.UUID, content json.RawMessage, plainText string, excerpt string, tags []string, references []uuid.UUID, state *string) (*HydratedMemo, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}

	var updated *querier.UpdateMemoRow

	if err := s.model.RunTransaction(ctx, func(txModel model.ModelInterface) error {
		current, err := s.getMemoByID(ctx, txModel, userID, id)
		if err != nil {
			return err
		}

		if err := validateMemoContent(content); err != nil {
			return err
		}

		newState := current.State
		if state != nil {
			newState = strings.TrimSpace(*state)
		}
		if err := validateMemoState(newState, false); err != nil {
			return err
		}

		normalizedPlainText := normalizeMemoText(plainText)
		normalizedExcerpt := normalizeMemoExcerpt(excerpt)
		normalizedTags := normalizeMemoTags(tags)
		normalizedReferences := normalizeMemoReferences(references)
		if err := s.validateMemoReferences(ctx, txModel, userID, normalizedReferences); err != nil {
			return err
		}

		archivedAt := nextArchivedAt(current.ArchivedAt, newState)
		item, err := txModel.UpdateMemo(ctx, querier.UpdateMemoParams{
			ID:         id,
			UserID:     userID,
			Content:    content,
			PlainText:  normalizedPlainText,
			Excerpt:    normalizedExcerpt,
			State:      newState,
			ArchivedAt: archivedAt,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMemoNotFound
			}
			return err
		}
		updated = item
		return s.syncMemoDerivedData(ctx, txModel, userID, id, normalizedTags, normalizedReferences)
	}); err != nil {
		return nil, err
	}

	normalizedTags := normalizeMemoTags(tags)
	normalizedReferences := normalizeMemoReferences(references)
	return &HydratedMemo{Item: memoItemFromUpdate(updated), Tags: normalizedTags, References: normalizedReferences}, nil
}

func (s *memoService) DeleteMemo(ctx context.Context, userID int32, id uuid.UUID) error {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return err
	}

	return s.model.RunTransaction(ctx, func(txModel model.ModelInterface) error {
		if err := txModel.DeleteAttachmentLinksByResource(ctx, querier.DeleteAttachmentLinksByResourceParams{
			UserID:       userID,
			ResourceType: memoResourceType,
			ResourceID:   id,
		}); err != nil {
			return err
		}
		_, err := txModel.DeleteMemo(ctx, querier.DeleteMemoParams{ID: id, UserID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemoNotFound
		}
		return err
	})
}

func (s *memoService) ListMemoBacklinks(ctx context.Context, userID int32, id uuid.UUID) ([]*HydratedMemoSummary, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := s.getMemoByID(ctx, s.model, userID, id); err != nil {
		return nil, err
	}

	items, err := s.model.ListMemoBacklinks(ctx, querier.ListMemoBacklinksParams{UserID: userID, TargetMemoID: id})
	if err != nil {
		return nil, err
	}
	result := make([]*HydratedMemoSummary, 0, len(items))
	for _, item := range items {
		tags, err := s.model.ListMemoTagsByMemo(ctx, querier.ListMemoTagsByMemoParams{MemoID: item.ID, UserID: userID})
		if err != nil {
			return nil, err
		}
		result = append(result, &HydratedMemoSummary{Item: memoItemFromBacklink(item), Tags: tags})
	}
	return result, nil
}

func (s *memoService) ListTags(ctx context.Context, userID int32, filter TagListFilter) ([]*querier.ListTagsRow, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.model.ListTags(ctx, querier.ListTagsParams{
		UserID:  userID,
		Column2: strings.TrimSpace(filter.Query),
		Limit:   clampLimit(filter.Limit),
		Offset:  clampOffset(filter.Offset),
	})
}

func (s *memoService) getMemoByID(ctx context.Context, m model.ModelInterface, userID int32, id uuid.UUID) (*memoItem, error) {
	item, err := m.GetMemoByID(ctx, querier.GetMemoByIDParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMemoNotFound
	}
	if err != nil {
		return nil, err
	}
	return memoItemFromGet(item), nil
}

func (s *memoService) hydrateMemo(ctx context.Context, m model.ModelInterface, userID int32, item *memoItem) (*HydratedMemo, error) {
	tags, err := m.ListMemoTagsByMemo(ctx, querier.ListMemoTagsByMemoParams{MemoID: item.ID, UserID: userID})
	if err != nil {
		return nil, err
	}
	references, err := m.ListMemoReferenceIDsBySource(ctx, querier.ListMemoReferenceIDsBySourceParams{SourceMemoID: item.ID, UserID: userID})
	if err != nil {
		return nil, err
	}
	return &HydratedMemo{Item: item, Tags: tags, References: references}, nil
}

func (s *memoService) hydrateMemoSummaries(ctx context.Context, m model.ModelInterface, userID int32, items []*memoItem) ([]*HydratedMemoSummary, error) {
	result := make([]*HydratedMemoSummary, 0, len(items))
	for _, item := range items {
		tags, err := m.ListMemoTagsByMemo(ctx, querier.ListMemoTagsByMemoParams{MemoID: item.ID, UserID: userID})
		if err != nil {
			return nil, err
		}
		result = append(result, &HydratedMemoSummary{Item: item, Tags: tags})
	}
	return result, nil
}

func memoItemFromCreate(row *querier.CreateMemoRow) *memoItem {
	return &memoItem{
		ID:         row.ID,
		UserID:     row.UserID,
		Content:    row.Content,
		PlainText:  row.PlainText,
		Excerpt:    row.Excerpt,
		State:      row.State,
		ArchivedAt: row.ArchivedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func memoItemFromUpdate(row *querier.UpdateMemoRow) *memoItem {
	return &memoItem{
		ID:         row.ID,
		UserID:     row.UserID,
		Content:    row.Content,
		PlainText:  row.PlainText,
		Excerpt:    row.Excerpt,
		State:      row.State,
		ArchivedAt: row.ArchivedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func memoItemFromGet(row *querier.GetMemoByIDRow) *memoItem {
	return &memoItem{
		ID:         row.ID,
		UserID:     row.UserID,
		Content:    row.Content,
		PlainText:  row.PlainText,
		Excerpt:    row.Excerpt,
		State:      row.State,
		ArchivedAt: row.ArchivedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func memoItemFromList(row *querier.ListMemosRow) *memoItem {
	return &memoItem{
		ID:         row.ID,
		UserID:     row.UserID,
		Content:    row.Content,
		PlainText:  row.PlainText,
		Excerpt:    row.Excerpt,
		State:      row.State,
		ArchivedAt: row.ArchivedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func memoItemFromBacklink(row *querier.ListMemoBacklinksRow) *memoItem {
	return &memoItem{
		ID:         row.ID,
		UserID:     row.UserID,
		Content:    row.Content,
		PlainText:  row.PlainText,
		Excerpt:    row.Excerpt,
		State:      row.State,
		ArchivedAt: row.ArchivedAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func (s *memoService) syncMemoDerivedData(ctx context.Context, m model.ModelInterface, userID int32, memoID uuid.UUID, tags []string, references []uuid.UUID) error {
	if err := m.DeleteMemoTagsByMemo(ctx, querier.DeleteMemoTagsByMemoParams{MemoID: memoID, UserID: userID}); err != nil {
		return err
	}
	for _, tag := range tags {
		if err := m.CreateMemoTag(ctx, querier.CreateMemoTagParams{MemoID: memoID, UserID: userID, Tag: tag}); err != nil {
			return err
		}
	}

	if err := m.DeleteMemoRelationsBySource(ctx, querier.DeleteMemoRelationsBySourceParams{SourceMemoID: memoID, UserID: userID}); err != nil {
		return err
	}
	for _, refID := range references {
		if err := m.CreateMemoRelation(ctx, querier.CreateMemoRelationParams{SourceMemoID: memoID, TargetMemoID: refID, UserID: userID}); err != nil {
			return err
		}
	}
	return nil
}

func (s *memoService) validateMemoReferences(ctx context.Context, m model.ModelInterface, userID int32, refs []uuid.UUID) error {
	for _, refID := range refs {
		if _, err := s.getMemoByID(ctx, m, userID, refID); err != nil {
			if errors.Is(err, ErrMemoNotFound) {
				return ErrInvalidMemoReference
			}
			return err
		}
	}
	return nil
}

func validateMemoState(state string, allowEmpty bool) error {
	state = strings.TrimSpace(state)
	if state == "" && allowEmpty {
		return nil
	}
	if state != "active" && state != "archived" {
		return ErrInvalidMemoState
	}
	return nil
}

func nextArchivedAt(current *time.Time, state string) *time.Time {
	if state != "archived" {
		return nil
	}
	if current != nil {
		return current
	}
	now := time.Now().UTC()
	return &now
}

func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func clampOffset(offset int32) int32 {
	if offset < 0 {
		return 0
	}
	return offset
}
