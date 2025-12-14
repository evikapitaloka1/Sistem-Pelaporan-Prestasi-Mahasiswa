package service

import (
    "sistempelaporan/app/repository"
    "sistempelaporan/helper"
    "github.com/gofiber/fiber/v2"
)

// --- Students Service ---

// GET /api/v1/students
func GetStudents(c *fiber.Ctx) error {
    data, err := repository.GetAllStudents()
    if err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data mahasiswa", err.Error())
    }
    return helper.Success(c, data, "Data mahasiswa berhasil diambil")
}

// GET /api/v1/students/:id
func GetStudentByID(c *fiber.Ctx) error {
    id := c.Params("id")
    data, err := repository.GetStudentDetail(id)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", err.Error())
    }
    return helper.Success(c, data, "Detail mahasiswa berhasil diambil")
}

// GET /api/v1/students/:id/achievements
func GetStudentAchievements(c *fiber.Ctx) error {
    id := c.Params("id")
    
    // Cek apakah student ada
    if _, err := repository.GetStudentDetail(id); err != nil {
        return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
    }

    data, err := repository.GetAchievementsByStudentID(id)
    if err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil prestasi", err.Error())
    }
    return helper.Success(c, data, "Prestasi mahasiswa berhasil diambil")
}

// PUT /api/v1/students/:id/advisor
// (Fungsi SetAdvisor yang sebelumnya sudah ada, kita pakai lagi disini untuk konsistensi)
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

// GET /api/v1/lecturers
func GetLecturers(c *fiber.Ctx) error {
    data, err := repository.GetAllLecturers()
    if err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data dosen", err.Error())
    }
    return helper.Success(c, data, "Data dosen berhasil diambil")
}

// GET /api/v1/lecturers/:id/advisees
func GetLecturerAdvisees(c *fiber.Ctx) error {
    lecturerID := c.Params("id")
    
    data, err := repository.GetLecturerAdvisees(lecturerID)
    if err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil mahasiswa bimbingan", err.Error())
    }
    return helper.Success(c, data, "Data mahasiswa bimbingan berhasil diambil")
}