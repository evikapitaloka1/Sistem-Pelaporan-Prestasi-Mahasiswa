package helper

import (
	"net/http"

	model "uas/app/model/postgres"
	service "uas/app/service/postgres"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"fmt"
)

// ================= GET ALL USERS =================
func GetAllUsers(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := s.GetAllUsers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}

// ================= GET USER BY ID =================
func GetUserByID(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}

		user, err := s.GetUserByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// ================= CREATE USER =================
// ================= CREATE USER =================
func CreateUser(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.CreateUserRequest

		// Step 1: Bind JSON
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON tidak valid: " + err.Error()})
			return
		}
		// Debug: tampilkan request
		fmt.Printf("DEBUG CreateUser request: %+v\n", req)

		// Step 2: Panggil service
		id, err := s.CreateUser(c.Request.Context(), req)
		if err != nil {
			// Debug: tampilkan error
			fmt.Printf("DEBUG CreateUser error from service: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Service error: " + err.Error()})
			return
		}

		// Step 3: Sukses
		fmt.Printf("DEBUG CreateUser success: ID=%s\n", id)
		c.JSON(http.StatusCreated, gin.H{
			"message": "User berhasil dibuat",
			"id":      id,
		})
	}
}


// ================= UPDATE USER =================
func UpdateUser(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}

		var req model.UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := s.UpdateUser(c.Request.Context(), id, req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User berhasil diperbarui"})
	}
}

// ================= DELETE USER =================
func DeleteUser(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}

		if err := s.DeleteUser(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
	}
}

// ================= UPDATE USER ROLE =================
func UpdateUserRole(s *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}

		var req model.UpdateUserRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := s.UpdateUserRole(c.Request.Context(), id, req.RoleID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Role user berhasil diupdate"})
	}
}
