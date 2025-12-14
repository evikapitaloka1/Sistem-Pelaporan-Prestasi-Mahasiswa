package helper

import "github.com/gofiber/fiber/v2"

// Standard Response
type Response struct {
	Code    int         `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func Success(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Code:    200,
		Status:  "OK",
		Message: message,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Code:    201,
		Status:  "Created",
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c *fiber.Ctx, data interface{}, meta interface{}, message string) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Code:    200,
		Status:  "OK",
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Error(c *fiber.Ctx, code int, message string, errs interface{}) error {
	statusText := "Error"
	if code == 400 { statusText = "Bad Request" }
	if code == 401 { statusText = "Unauthorized" }
	if code == 403 { statusText = "Forbidden" }
	if code == 404 { statusText = "Not Found" }
	if code == 500 { statusText = "Internal Server Error" }

	return c.Status(code).JSON(Response{
		Code:    code,
		Status:  statusText,
		Message: message,
		Errors:  errs,
	})
}