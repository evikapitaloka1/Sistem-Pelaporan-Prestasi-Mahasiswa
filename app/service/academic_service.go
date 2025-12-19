package service

import (
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"
	"github.com/gofiber/fiber/v2"
)

// --- Students Service ---

// GetStudents godoc
// @Summary      Dapatkan semua mahasiswa
// @Description  Mengambil daftar lengkap mahasiswa dari database PostgreSQL
// @Tags         Students
// @Produce      json
// @Success      200  {object}  helper.Response
// @Failure      500  {object}  helper.Response
// @Router       /students [get]
// @Security     BearerAuth
func GetStudents(c *fiber.Ctx) error {
	data, err := repository.GetAllStudents()
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data mahasiswa", err.Error())
	}
	return helper.Success(c, data, "Data mahasiswa berhasil diambil")
}

// GetStudentByID godoc
// @Summary      Dapatkan detail mahasiswa
// @Description  Mengambil data spesifik mahasiswa berdasarkan UUID
// @Tags         Students
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  helper.Response
// @Failure      404  {object}  helper.Response
// @Router       /students/{id} [get]
// @Security     BearerAuth
func GetStudentByID(c *fiber.Ctx) error {
	id := c.Params("id")
	data, err := repository.GetStudentDetail(id)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", err.Error())
	}
	return helper.Success(c, data, "Detail mahasiswa berhasil diambil")
}

// GetStudentAchievements godoc
// @Summary      Lihat prestasi mahasiswa
// @Description  Mengambil daftar semua prestasi milik satu mahasiswa tertentu
// @Tags         Students
// @Produce      json
// @Param        id   path      string  true  "Student ID"
// @Success      200  {object}  helper.Response
// @Router       /students/{id}/achievements [get]
// @Security     BearerAuth
func GetStudentAchievements(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := repository.GetStudentDetail(id); err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	data, err := repository.GetAchievementsByStudentID(id)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil prestasi", err.Error())
	}
	return helper.Success(c, data, "Prestasi mahasiswa berhasil diambil")
}

// UpdateStudentAdvisor godoc
// @Summary      Update Dosen Wali
// @Description  Mengatur atau mengganti ID Dosen Wali untuk mahasiswa tertentu
// @Tags         Students
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Student ID"
// @Param        body  body      object  true  "Advisor ID JSON"
// @Success      200   {object}  helper.Response
// @Router       /students/{id}/advisor [put]
// @Security     BearerAuth
func UpdateStudentAdvisor(c *fiber.Ctx) error {
	studentID := c.Params("id")
	var req struct {
		AdvisorID string `json:"advisor_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", nil)
	}

	if err := repository.AssignAdvisorToStudent(studentID, req.AdvisorID); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update dosen wali", err.Error())
	}
	return helper.Success(c, nil, "Dosen wali berhasil diupdate")
}

// --- Lecturers Service ---

// GetLecturers godoc
// @Summary      Dapatkan semua dosen
// @Tags         Lecturers
// @Produce      json
// @Router       /lecturers [get]
// @Security     BearerAuth
func GetLecturers(c *fiber.Ctx) error {
	data, err := repository.GetAllLecturers()
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data dosen", err.Error())
	}
	return helper.Success(c, data, "Data dosen berhasil diambil")
}

// GetLecturerAdvisees godoc
// @Summary      Dapatkan mahasiswa bimbingan
// @Description  Mengambil daftar mahasiswa yang berada di bawah bimbingan dosen tertentu
// @Tags         Lecturers
// @Param        id   path      string  true  "Lecturer ID"
// @Success      200  {object}  helper.Response
// @Router       /lecturers/{id}/advisees [get]
// @Security     BearerAuth
func GetLecturerAdvisees(c *fiber.Ctx) error {
	lecturerID := c.Params("id")
	data, err := repository.GetLecturerAdvisees(lecturerID)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil mahasiswa bimbingan", err.Error())
	}
	return helper.Success(c, data, "Data mahasiswa bimbingan berhasil diambil")
}