package handler

import (
	"encoding/json"
	"errors"
	"time"

	anclaxauth "github.com/cloudcarver/anclax/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

type memoUpsertRequest struct {
	Content    json.RawMessage `json:"content"`
	PlainText  string          `json:"plainText"`
	Excerpt    string          `json:"excerpt"`
	Tags       []string        `json:"tags"`
	References []uuid.UUID     `json:"references"`
	State      *string         `json:"state,omitempty"`
}

type memoResponse struct {
	ArchivedAt *time.Time      `json:"archivedAt,omitempty"`
	Content    json.RawMessage `json:"content"`
	CreatedAt  time.Time       `json:"createdAt"`
	Excerpt    string          `json:"excerpt"`
	Id         uuid.UUID       `json:"id"`
	PlainText  string          `json:"plainText"`
	References []uuid.UUID     `json:"references"`
	State      string          `json:"state"`
	Tags       []string        `json:"tags"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

type memoSummaryResponse struct {
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	Excerpt    string     `json:"excerpt"`
	Id         uuid.UUID  `json:"id"`
	PlainText  string     `json:"plainText"`
	State      string     `json:"state"`
	Tags       []string   `json:"tags"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (h *Handler) ListMemos(c fiber.Ctx, params apigen.ListMemosParams) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	filter := MemoListFilter{}
	if params.Q != nil {
		filter.Query = *params.Q
	}
	if params.Tag != nil {
		filter.Tag = *params.Tag
	}
	if params.State != nil {
		filter.State = string(*params.State)
	}
	if params.Limit != nil {
		filter.Limit = *params.Limit
	}
	if params.Offset != nil {
		filter.Offset = *params.Offset
	}

	items, err := h.memos.ListMemos(c.Context(), userID, filter)
	if errors.Is(err, ErrInvalidMemoState) {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result := make([]memoSummaryResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapMemoSummaryToAPI(item))
	}
	return c.JSON(result)
}

func (h *Handler) CreateMemo(c fiber.Ctx) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req memoUpsertRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	item, err := h.memos.CreateMemo(c.Context(), userID, req.Content, req.PlainText, req.Excerpt, req.Tags, req.References)
	if errors.Is(err, ErrInvalidMemoContent) {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if errors.Is(err, ErrInvalidMemoReference) {
		return c.Status(fiber.StatusUnprocessableEntity).SendString(err.Error())
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result, err := mapMemoToAPI(item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *Handler) GetMemo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	item, err := h.memos.GetMemo(c.Context(), userID, id)
	if errors.Is(err, ErrMemoNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result, err := mapMemoToAPI(item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(result)
}

func (h *Handler) UpdateMemo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req memoUpsertRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	item, err := h.memos.UpdateMemo(
		c.Context(),
		userID,
		id,
		req.Content,
		req.PlainText,
		req.Excerpt,
		req.Tags,
		req.References,
		req.State,
	)
	switch {
	case errors.Is(err, ErrMemoNotFound):
		return c.SendStatus(fiber.StatusNotFound)
	case errors.Is(err, ErrInvalidMemoReference):
		return c.Status(fiber.StatusUnprocessableEntity).SendString(err.Error())
	case errors.Is(err, ErrInvalidMemoContent), errors.Is(err, ErrInvalidMemoState):
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result, err := mapMemoToAPI(item)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(result)
}

func (h *Handler) DeleteMemo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	if err := h.memos.DeleteMemo(c.Context(), userID, id); err != nil {
		if errors.Is(err, ErrMemoNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListMemoBacklinks(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	items, err := h.memos.ListMemoBacklinks(c.Context(), userID, id)
	if errors.Is(err, ErrMemoNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result := make([]memoSummaryResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapMemoSummaryToAPI(item))
	}
	return c.JSON(result)
}

func (h *Handler) ListTags(c fiber.Ctx, params apigen.ListTagsParams) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	filter := TagListFilter{}
	if params.Q != nil {
		filter.Query = *params.Q
	}
	if params.Limit != nil {
		filter.Limit = *params.Limit
	}
	if params.Offset != nil {
		filter.Offset = *params.Offset
	}

	items, err := h.memos.ListTags(c.Context(), userID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result := make([]apigen.TagSummary, 0, len(items))
	for _, item := range items {
		result = append(result, mapTagSummaryToAPI(item))
	}
	return c.JSON(result)
}

func mapMemoToAPI(item *HydratedMemo) (memoResponse, error) {
	content := item.Item.Content
	if len(content) == 0 {
		content = json.RawMessage(`{}`)
	}
	return memoResponse{
		ArchivedAt: item.Item.ArchivedAt,
		Content:    content,
		CreatedAt:  item.Item.CreatedAt,
		Excerpt:    item.Item.Excerpt,
		Id:         item.Item.ID,
		PlainText:  item.Item.PlainText,
		References: nonNilUUIDs(item.References),
		State:      item.Item.State,
		Tags:       nonNilStrings(item.Tags),
		UpdatedAt:  item.Item.UpdatedAt,
	}, nil
}

func mapMemoSummaryToAPI(item *HydratedMemoSummary) memoSummaryResponse {
	return memoSummaryResponse{
		ArchivedAt: item.Item.ArchivedAt,
		CreatedAt:  item.Item.CreatedAt,
		Excerpt:    item.Item.Excerpt,
		Id:         item.Item.ID,
		PlainText:  item.Item.PlainText,
		State:      item.Item.State,
		Tags:       nonNilStrings(item.Tags),
		UpdatedAt:  item.Item.UpdatedAt,
	}
}

func nonNilStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func nonNilUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return []uuid.UUID{}
	}
	return append([]uuid.UUID(nil), values...)
}

func mapTagSummaryToAPI(item *querier.ListTagsRow) apigen.TagSummary {
	return apigen.TagSummary{
		Name:  item.Name,
		Count: item.Count,
	}
}
