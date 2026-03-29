package handler

import (
	"errors"

	anclaxauth "github.com/cloudcarver/anclax/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
	"github.com/wibus-wee/allinone/pkg/zgen/taskgen"
)

type Handler struct {
	model       model.ModelInterface
	taskrunner  taskgen.TaskRunner
	todos       TodoService
	memos       MemoService
	attachments AttachmentService
}

func NewHandler(model model.ModelInterface, taskrunner taskgen.TaskRunner) (apigen.ServerInterface, error) {
	return &Handler{
		model:       model,
		taskrunner:  taskrunner,
		todos:       NewTodoService(model),
		memos:       NewMemoService(model),
		attachments: NewAttachmentService(model, LoadAttachmentConfig()),
	}, nil
}

func (h *Handler) GetCounter(c fiber.Ctx) error {
	count, err := h.model.GetCounter(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(apigen.Counter{Count: count.Value})
}

func (h *Handler) IncrementCounter(c fiber.Ctx) error {
	_, err := h.taskrunner.RunIncrementCounter(c.Context(), &taskgen.IncrementCounterParameters{
		Amount: 1,
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusAccepted).SendString("Incremented")
}

func (h *Handler) ListTodos(c fiber.Ctx) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	items, err := h.todos.ListTodos(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result := make([]apigen.TodoItem, 0, len(items))
	for _, item := range items {
		result = append(result, mapTodoToAPI(item))
	}

	return c.JSON(result)
}

func (h *Handler) CreateTodo(c fiber.Ctx) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.CreateTodoRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).SendString("title is required")
	}

	item, err := h.todos.CreateTodo(c.Context(), userID, req.Title)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(mapTodoToAPI(item))
}

func (h *Handler) UpdateTodo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.UpdateTodoRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	var bucket *string
	if req.Bucket != nil {
		b := string(*req.Bucket)
		bucket = &b
	}

	item, err := h.todos.UpdateTodo(c.Context(), userID, id, req.Title, req.Done, bucket, req.Description)
	if err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(mapTodoToAPI(item))
}

func (h *Handler) DeleteTodo(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	if err := h.todos.DeleteTodo(c.Context(), userID, id); err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func mapTodoToAPI(item *querier.TodoItem) apigen.TodoItem {
	return apigen.TodoItem{
		Id:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Done:        item.Done,
		Bucket:      apigen.TodoItemBucket(item.Bucket),
		CreatedAt:   item.CreatedAt,
	}
}
