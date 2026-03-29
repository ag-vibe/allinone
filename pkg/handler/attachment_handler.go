package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	anclaxauth "github.com/cloudcarver/anclax/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

func (h *Handler) UploadAttachment(c fiber.Ctx) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("file is required")
	}
	if fileHeader.Size <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("file is empty")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, _ := file.Read(sniff)
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(sniff[:n])
	}
	reader := io.MultiReader(bytes.NewReader(sniff[:n]), file)

	item, err := h.attachments.CreateAttachment(c.Context(), userID, fileHeader.Filename, contentType, fileHeader.Size, reader)
	if errors.Is(err, ErrAttachmentTooLarge) {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("attachment too large")
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(mapAttachmentToAPI(item))
}

func (h *Handler) GetAttachment(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	item, err := h.attachments.GetAttachment(c.Context(), userID, id)
	if errors.Is(err, ErrAttachmentNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(mapAttachmentToAPI(item))
}

func (h *Handler) DownloadAttachment(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	item, err := h.attachments.GetAttachment(c.Context(), userID, id)
	if errors.Is(err, ErrAttachmentNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	fullPath := h.attachments.ResolveStoragePath(item)
	c.Set("Content-Type", item.ContentType)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(item.Filename)))
	return c.SendFile(fullPath)
}

func (h *Handler) DeleteAttachment(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	_, err = h.attachments.DeleteAttachment(c.Context(), userID, id)
	if errors.Is(err, ErrAttachmentNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) LinkAttachment(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.LinkAttachmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	resourceType := strings.TrimSpace(req.ResourceType)
	if resourceType == "" {
		return c.Status(fiber.StatusBadRequest).SendString("resourceType is required")
	}

	if err := h.attachments.LinkAttachment(c.Context(), userID, id, resourceType, req.ResourceId); err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) UnlinkAttachment(c fiber.Ctx, id uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.LinkAttachmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	resourceType := strings.TrimSpace(req.ResourceType)
	if resourceType == "" {
		return c.Status(fiber.StatusBadRequest).SendString("resourceType is required")
	}

	if err := h.attachments.UnlinkAttachment(c.Context(), userID, id, resourceType, req.ResourceId); err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListAttachmentsByResource(c fiber.Ctx, resourceType string, resourceID uuid.UUID) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return c.Status(fiber.StatusBadRequest).SendString("resourceType is required")
	}

	items, err := h.attachments.ListAttachmentsByResource(c.Context(), userID, resourceType, resourceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	result := make([]apigen.Attachment, 0, len(items))
	for _, item := range items {
		result = append(result, mapAttachmentToAPI(item))
	}

	return c.JSON(result)
}

func mapAttachmentToAPI(item *querier.Attachment) apigen.Attachment {
	downloadPath := fmt.Sprintf("/api/v1/attachments/%s/content", item.ID.String())
	return apigen.Attachment{
		Id:          item.ID,
		Filename:    item.Filename,
		ContentType: item.ContentType,
		SizeBytes:   item.SizeBytes,
		CreatedAt:   item.CreatedAt,
		DownloadUrl: downloadPath,
	}
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `"`, "")
	name = strings.ReplaceAll(name, "\n", "")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.TrimSpace(name)
	if name == "" {
		return "attachment"
	}
	return name
}
