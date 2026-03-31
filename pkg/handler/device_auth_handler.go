package handler

import (
	"embed"
	"html"
	"strings"

	anclaxauth "github.com/cloudcarver/anclax/pkg/auth"
	"github.com/gofiber/fiber/v3"

	"github.com/wibus-wee/allinone/pkg/deviceauth"
	"github.com/wibus-wee/allinone/pkg/zgen/apigen"
)

func (h *Handler) DeviceAuthorize(c fiber.Ctx) error {
	var req apigen.DeviceAuthorizeRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	meta := deviceauth.RequestMeta{
		IP:        stringPtr(c.IP()),
		UserAgent: stringPtr(c.Get("User-Agent")),
	}

	resp, err := h.deviceAuth.Authorize(c.Context(), req, meta)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.JSON(resp)
}

func (h *Handler) DeviceToken(c fiber.Ctx) error {
	var req apigen.DeviceTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	resp, err := h.deviceAuth.Token(c.Context(), req.DeviceCode)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.JSON(resp)
}

func (h *Handler) DeviceApprove(c fiber.Ctx) error {
	userID, err := anclaxauth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	var req apigen.DeviceApproveRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	resp, err := h.deviceAuth.Approve(c.Context(), req.UserCode, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.JSON(resp)
}

func (h *Handler) DeviceVerify(c fiber.Ctx) error {
	userCode := strings.TrimSpace(c.Query("user_code"))
	escaped := html.EscapeString(userCode)
	page := strings.ReplaceAll(deviceVerifyHTML, "{{USER_CODE}}", escaped)
	return c.Type("html", "utf-8").SendString(page)
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

//go:embed device-verify.html
var deviceVerifyHTML string

var _ embed.FS
