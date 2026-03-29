package handler

import (
	"errors"

	anclaxauth "github.com/cloudcarver/anclax/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

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

	result := make([]apigen.MemoSummary, 0, len(items))
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

	var req apigen.CreateMemoRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	item, err := h.memos.CreateMemo(c.Context(), userID, req.Content)
	if errors.Is(err, ErrInvalidMemoContent) {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if errors.Is(err, ErrInvalidMemoReference) {
		return c.Status(fiber.StatusUnprocessableEntity).SendString(err.Error())
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(mapMemoToAPI(item))
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

	return c.JSON(mapMemoToAPI(item))
}

func (h *Handler) UpdateMemo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.UpdateMemoRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	var state *string
	if req.State != nil {
		value := string(*req.State)
		state = &value
	}

	item, err := h.memos.UpdateMemo(c.Context(), userID, id, req.Content, state)
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

	return c.JSON(mapMemoToAPI(item))
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

	result := make([]apigen.MemoSummary, 0, len(items))
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

func mapMemoToAPI(item *HydratedMemo) apigen.Memo {
	references := make([]uuid.UUID, 0, len(item.References))
	for _, ref := range item.References {
		references = append(references, ref)
	}
	return apigen.Memo{
		ArchivedAt: item.Item.ArchivedAt,
		Content:    item.Item.Content,
		CreatedAt:  item.Item.CreatedAt,
		Excerpt:    item.Item.Excerpt,
		Id:         item.Item.ID,
		References: references,
		State:      apigen.MemoState(item.Item.State),
		Tags:       append([]string(nil), item.Tags...),
		UpdatedAt:  item.Item.UpdatedAt,
	}
}

func mapMemoSummaryToAPI(item *HydratedMemoSummary) apigen.MemoSummary {
	return apigen.MemoSummary{
		ArchivedAt: item.Item.ArchivedAt,
		CreatedAt:  item.Item.CreatedAt,
		Excerpt:    item.Item.Excerpt,
		Id:         item.Item.ID,
		State:      apigen.MemoState(item.Item.State),
		Tags:       append([]string(nil), item.Tags...),
		UpdatedAt:  item.Item.UpdatedAt,
	}
}

func mapTagSummaryToAPI(item *querier.ListTagsRow) apigen.TagSummary {
	return apigen.TagSummary{
		Name:  item.Name,
		Count: item.Count,
	}
}
