package helper

import(
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"errors"
) 


// 200 OK + data
func SendSuccess(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   data,
	})
}

// 200 OK tanpa data
func SendSuccessNoData(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   nil,
	})
}

// 201 Created
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   data,
	})
}

// Error response
func SendError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}
func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {

	raw := c.Locals("userID")

	switch v := raw.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	}

	return uuid.Nil, errors.New("user ID missing from token")
}

func GetRole(c *fiber.Ctx) (string, error) {
	role, ok := c.Locals("role").(string)
	if !ok || role == "" {
		return "", errors.New("role missing from token")
	}
	return role, nil
}