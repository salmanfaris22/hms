package response

import (
	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/pkg/constants"
)

type Envelope struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       any    `json:"data,omitempty"`
}

func Write(c *fiber.Ctx, status int, env Envelope) error {
	c.Status(status)
	return c.JSON(env)
}

func Success(c *fiber.Ctx, status int, message string, data any) error {
	if message == "" {
		message = constants.MsgOK
	}
	return Write(c, status, Envelope{
		Success:    true,
		StatusCode: status,
		Message:   message,
		Data:      data,
	})
}

func OK(c *fiber.Ctx, message string, data any) error {
	return Success(c, constants.StatusOK, message, data)
}

func Created(c *fiber.Ctx, message string, data any) error {
	return Success(c, constants.StatusCreated, message, data)
}

func Error(c *fiber.Ctx, status int, message string) error {
	return Write(c, status, Envelope{
		Success:    false,
		StatusCode: status,
		Message:    message,
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	if message == "" {
		message = constants.MsgInvalidBody
	}
	return Error(c, constants.StatusBadRequest, message)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	if message == "" {
		message = constants.MsgUnauthorized
	}
	return Error(c, constants.StatusUnauthorized, message)
}

func Forbidden(c *fiber.Ctx, message string) error {
	if message == "" {
		message = constants.MsgForbidden
	}
	return Error(c, constants.StatusForbidden, message)
}

func NotFound(c *fiber.Ctx, message string) error {
	if message == "" {
		message = constants.MsgNotFound
	}
	return Error(c, constants.StatusNotFound, message)
}

func Internal(c *fiber.Ctx, message string) error {
	if message == "" {
		message = constants.MsgInternal
	}
	return Error(c, constants.StatusInternalServerError, message)
}