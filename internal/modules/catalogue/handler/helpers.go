package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

func writeJSON(c *fiber.Ctx, status int, v any) error {
	msg := constants.MsgOK
	if status == constants.StatusCreated {
		msg = constants.MsgCreated
	}
	return response.Success(c, status, msg, v)
}

func writeErr(c *fiber.Ctx, status int, msg string) error {
	return response.Error(c, status, msg)
}