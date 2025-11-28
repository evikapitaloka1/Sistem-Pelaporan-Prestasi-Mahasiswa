package routes

import (
	service "uas/app/service/postgres"
	"uas/helper"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(router *gin.RouterGroup, userService *service.UserService) {

	users := router.Group("/users")

	users.GET("", helper.GetAllUsers(userService))
	users.GET("/:id", helper.GetUserByID(userService))
	users.POST("", helper.CreateUser(userService))
	users.PUT("/:id", helper.UpdateUser(userService))
	users.DELETE("/:id", helper.DeleteUser(userService))
	users.PUT("/:id/role", helper.UpdateUserRole(userService))
}
