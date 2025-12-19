package service

import (
	"sistempelaporan/app/model"
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

// CreateNewUser godoc
// @Summary      Buat User Baru (Admin)
// @Description  Admin dapat membuat user baru beserta profil Mahasiswa atau Dosen dalam satu transaksi.
// @Tags         Users (Admin)
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "Data User, Profil, dan Password"
// @Success      201   {object}  helper.Response
// @Failure      400   {object}  helper.Response
// @Failure      409   {object}  helper.Response
// @Router       /users [post]
// @Security     BearerAuth
func CreateNewUser(c *fiber.Ctx) error {
	var req struct {
		User     model.User      `json:"user"`
		Student  *model.Student  `json:"student"`
		Lecturer *model.Lecturer `json:"lecturer"`
		Password string          `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	if req.Password == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Password wajib diisi.", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal hash password", err.Error())
	}
	req.User.PasswordHash = string(hashedPassword)

	if req.User.ID == uuid.Nil {
		req.User.ID = uuid.New()
	}
	
	if req.Student != nil {
		if req.Student.ID == uuid.Nil {
			req.Student.ID = uuid.New()
		}
	}
	if req.Lecturer != nil {
		if req.Lecturer.ID == uuid.Nil {
			req.Lecturer.ID = uuid.New()
		}
	}

	if err := repository.CreateUserWithProfile(&req.User, req.Student, req.Lecturer); err != nil {
		if strings.Contains(err.Error(), "Role ID yang dimasukkan tidak ditemukan") {
			return helper.Error(c, fiber.StatusConflict, "Role ID tidak valid atau tidak ada di database.", nil)
		}
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			return helper.Error(c, fiber.StatusConflict, "Username atau Email sudah digunakan.", nil)
		}
		
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal membuat user", err.Error())
	}

	return helper.Created(c, nil, "User berhasil dibuat")
}

// GetAllUsers godoc
// @Summary      Dapatkan Semua User
// @Description  Mengambil daftar lengkap semua user yang terdaftar di sistem.
// @Tags         Users (Admin)
// @Produce      json
// @Success      200   {array}   model.User
// @Failure      500   {object}  helper.Response
// @Router       /users [get]
// @Security     BearerAuth
func GetAllUsers(c *fiber.Ctx) error {
	users, err := repository.FindAllUsers()
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user", err.Error())
	}
	return helper.Success(c, users, "Data semua user berhasil diambil")
}

// GetUserByID godoc
// @Summary      Dapatkan User by ID
// @Description  Mengambil detail data user tertentu berdasarkan UUID.
// @Tags         Users (Admin)
// @Produce      json
// @Param        id    path      string  true  "User ID (UUID)"
// @Success      200   {object}  model.User
// @Failure      404   {object}  helper.Response
// @Router       /users/{id} [get]
// @Security     BearerAuth
func GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := repository.FindUserByID(id)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User tidak ditemukan", err.Error())
	}
	return helper.Success(c, user, "Detail user berhasil diambil")
}

// UpdateUser godoc
// @Summary      Update Data Umum User
// @Description  Memperbarui username, nama lengkap, dan email user.
// @Tags         Users (Admin)
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "User ID"
// @Param        body  body      object  true  "Data Profil Baru"
// @Success      200   {object}  helper.Response
// @Router       /users/{id} [put]
// @Security     BearerAuth
func UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Username string `json:"username"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	if err := repository.UpdateUserGeneral(id, req.Username, req.FullName, req.Email); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update user", err.Error())
	}

	return helper.Success(c, nil, "User berhasil diupdate")
}

// UpdateUserRole godoc
// @Summary      Update Role User
// @Description  Admin dapat mengubah hak akses (role) user tertentu.
// @Tags         Users (Admin)
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "User ID"
// @Param        body  body      object  true  "Role ID Baru"
// @Success      200   {object}  helper.Response
// @Router       /users/{id}/role [put]
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func UpdateUserRole(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		RoleID string `json:"role_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	if err := repository.UpdateUserRole(id, req.RoleID); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update role user", err.Error())
	}

	return helper.Success(c, nil, "Role user berhasil diupdate")
}

// DeleteUser godoc
// @Summary      Hapus User
// @Description  Menghapus data user dari sistem (Soft Delete).
// @Tags         Users (Admin)
// @Param        id    path      string  true  "User ID"
// @Success      200   {object}  helper.Response
// @Router       /users/{id} [delete]
// @Security     BearerAuth
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := repository.DeleteUserByID(id); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus user", err.Error())
	}

	return helper.Success(c, nil, "User berhasil dihapus (soft delete)")
}

// SetAdvisor godoc
// @Summary      Assign Dosen Wali
// @Description  Menghubungkan mahasiswa dengan dosen wali tertentu.
// @Tags         Students
// @Accept       json
// @Produce      json
// @Param        studentId  path      string  true  "ID Mahasiswa"
// @Param        body       body      object  true  "ID Dosen Wali"
// @Success      200        {object}  helper.Response
// @Router       /students/{studentId}/advisor [put]
// @Security     BearerAuth
func SetAdvisor(c *fiber.Ctx) error {
	studentID := c.Params("studentId")
	
	var req struct {
		AdvisorID string `json:"advisor_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	if err := repository.AssignAdvisorToStudent(studentID, req.AdvisorID); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal assign dosen wali", err.Error())
	}

	return helper.Success(c, nil, "Dosen wali berhasil diassign")
}