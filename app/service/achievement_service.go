package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"
	"log"
	"strings"
	"sistempelaporan/app/model"
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- FR-003: Submit Prestasi ---
func SubmitAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	var req model.AchievementMongo
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	// Identifikasi Mahasiswa
	student, err := repository.FindStudentByUserID(userID)
	if err != nil {
		return helper.Error(c, fiber.StatusForbidden, "Profil mahasiswa tidak ditemukan", err.Error())
	}

	// Prepare Data
	refID := uuid.New()
	ref := model.AchievementReference{
		ID:        refID,
		StudentID: student.ID,
	}
	req.StudentID = student.ID.String() 
	
	// Simpan ke Repo
	if err := repository.CreateAchievement(&ref, &req); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan prestasi", err.Error())
	}

	return helper.Created(c, nil, "Prestasi berhasil dibuat")
}

// --- FR-010, FR-006: View Achievements ---
func GetListAchievements(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	roleName := c.Locals("role").(string)

	// Parsing Params
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	filter := model.AchievementFilter{
		Page:   page,
		Limit:  limit,
		Status: c.Query("status"),
	}

	repoFilter := repository.RepoFilter{AchievementFilter: filter}

	// Filter Logic Berdasarkan Role
	if roleName == "Mahasiswa" {
		student, err := repository.FindStudentByUserID(userID)
		if err != nil { return helper.Error(c, fiber.StatusForbidden, "Profil mahasiswa invalid", err.Error()) }
		repoFilter.StudentID = student.ID.String()
	} else if roleName == "Dosen Wali" {
		lecturer, err := repository.FindLecturerByUserID(userID)
		if err != nil { return helper.Error(c, fiber.StatusForbidden, "Profil dosen invalid", err.Error()) }
		repoFilter.AdvisorID = lecturer.ID.String()
	}

	data, total, err := repository.GetAllAchievements(repoFilter)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data", err.Error())
	}

	meta := &model.Meta{
		Page:      filter.Page,
		Limit:     filter.Limit,
		TotalData: total,
		TotalPage: int(total) / filter.Limit,
	}

	return helper.SuccessWithMeta(c, data, meta, "List prestasi berhasil diambil")
}

// --- FR-Baru: Get Detail ---
func GetAchievementDetail(c *fiber.Ctx) error {
	achievementID := c.Params("id")
	
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Data referensi tidak ditemukan", err.Error()) }

	detail, err := repository.GetAchievementDetailFromMongo(ach.MongoAchievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Detail konten tidak ditemukan", err.Error()) }

	resp := map[string]interface{}{
		"reference": ach,
		"detail":    detail,
	}
	return helper.Success(c, resp, "Detail prestasi")
}

// --- FR-Baru: Update Content (Edit Draft) ---
func UpdateAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("id")
	
	var newData model.AchievementMongo
	if err := c.BodyParser(&newData); err != nil { return helper.Error(c, fiber.StatusBadRequest, "Request body invalid", nil) }

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil) }

	student, _ := repository.FindStudentByUserID(userID)
	if ach.StudentID != student.ID { return helper.Error(c, fiber.StatusForbidden, "Akses ditolak", nil) }
	if ach.Status != model.StatusDraft { return helper.Error(c, fiber.StatusBadRequest, "Hanya status draft yang bisa diedit", nil) }

	if err := repository.UpdateAchievementDetail(ach.MongoAchievementID, newData); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update data", err.Error())
	}

	return helper.Success(c, nil, "Data berhasil diperbarui")
}

// --- FR-005: Delete ---
func DeleteAchievement(c *fiber.Ctx) error {
	// 1. Ambil User Info dengan Safe Type Assertion
	rawUserID := c.Locals("user_id")
	if rawUserID == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Token tidak valid atau sesi berakhir", nil)
	}
	userID := rawUserID.(string)

	rawUserRole := c.Locals("user_role")
	if rawUserRole == nil {
		log.Println("WARNING: user_role is nil, setting to unknown.")
		rawUserRole = "UNKNOWN" 
	}
	// userRole sekarang pasti "ADMIN" atau "MAHASISWA" (HURUF BESAR SEMUA)
	userRole := strings.ToUpper(rawUserRole.(string)) 

	achievementID := c.Params("id") // ID PostgreSQL (UUID)

	// 2. Cari Prestasi Berdasarkan ID PostgreSQL
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") { 
			return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", nil)
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mencari prestasi", err.Error())
	}

	// 3. Cek Kepemilikan (Role-Based Bypass)
	// Jika pengguna bukan pemilik (ach.StudentID.String() != userID)
	if ach.StudentID.String() != userID {
		// Jika bukan pemilik DAN bukan Admin, tolak.
		if userRole != "ADMIN" { 
			return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda bukan pemilik prestasi ini", nil)
		}
		// Jika Admin, dia bypass dan lanjut (Force Delete)
	}

	// 4. Cek Pembatasan Status (Mahasiswa vs. Admin)
	// Jika Role bukan ADMIN DAN status bukan DRAFT, tolak.
	if userRole != "ADMIN" && ach.Status != model.StatusDraft { 
		return helper.Error(c, fiber.StatusBadRequest, "Prestasi sudah disubmit/diverifikasi dan tidak dapat dihapus oleh Mahasiswa", nil)
	}
	// Admin (Role == ADMIN) akan melewati pengecekan status ini.

	// 5. Eksekusi Soft Delete Hybrid
	if err := repository.SoftDeleteAchievementTransaction(achievementID, ach.MongoAchievementID); err != nil {
		if strings.Contains(err.Error(), "not found or already deleted") {
			return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan atau sudah terhapus", nil)
		}
		
		// Penanganan Kegagalan Transaksi Hybrid
		if strings.Contains(err.Error(), "gagal soft delete di Mongo") {
			log.Printf("ERROR: Rollback parsial. Prestasi ID %s sukses di Postgres tapi gagal di Mongo: %v", achievementID, err)
			return helper.Error(c, fiber.StatusInternalServerError, "Gagal soft delete di MongoDB. Data di Postgres sudah ditandai terhapus.", nil)
		}

		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus data", err.Error())
	}

	return helper.Success(c, nil, "Prestasi berhasil dihapus (soft delete).")
}
// --- FR-004: Request Verification ---
func RequestVerification(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("id")

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error()) }

	student, err := repository.FindStudentByUserID(userID)
	if err != nil { return helper.Error(c, fiber.StatusForbidden, "Akses ditolak", err.Error()) }

	if ach.StudentID != student.ID { return helper.Error(c, fiber.StatusForbidden, "Bukan milik anda", nil) }
	if ach.Status != model.StatusDraft { return helper.Error(c, fiber.StatusBadRequest, "Hanya status draft yang bisa diajukan", nil) }

	if err := repository.UpdateStatus(achievementID, model.StatusSubmitted, nil, ""); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update status", err.Error())
	}

	return helper.Success(c, nil, "Berhasil diajukan untuk verifikasi")
}

// --- FR-007: Verify (Approve) ---
func VerifyAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("id")
	
	var input struct { Note string `json:"note"` }
	c.BodyParser(&input) // Note opsional

	lecturer, err := repository.FindLecturerByUserID(userID)
	if err != nil { return helper.Error(c, fiber.StatusForbidden, "Bukan dosen", nil) }

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", nil) }

	if ach.Status != model.StatusSubmitted { return helper.Error(c, fiber.StatusBadRequest, "Status bukan submitted", nil) }

	lecIDStr := lecturer.UserID.String()
	if err := repository.UpdateStatus(achievementID, model.StatusVerified, &lecIDStr, input.Note); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal verifikasi", err.Error())
	}

	return helper.Success(c, nil, "Prestasi berhasil diverifikasi (Verified)")
}

// --- FR-008: Reject ---
func RejectAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("id")
	
	var input struct { Note string `json:"note"` }
	if err := c.BodyParser(&input); err != nil { return helper.Error(c, fiber.StatusBadRequest, "Input invalid", nil) }
	if input.Note == "" { return helper.Error(c, fiber.StatusBadRequest, "Catatan penolakan wajib diisi", nil) }

	lecturer, err := repository.FindLecturerByUserID(userID)
	if err != nil { return helper.Error(c, fiber.StatusForbidden, "Bukan dosen", nil) }

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", nil) }

	if ach.Status != model.StatusSubmitted { return helper.Error(c, fiber.StatusBadRequest, "Status bukan submitted", nil) }

	lecIDStr := lecturer.UserID.String()
	if err := repository.UpdateStatus(achievementID, model.StatusRejected, &lecIDStr, input.Note); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menolak prestasi", err.Error())
	}

	return helper.Success(c, nil, "Prestasi ditolak (Rejected)")
}

// --- FR-Baru: Upload Attachment ---
func UploadAttachment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("id")

	file, err := c.FormFile("file")
	if err != nil { return helper.Error(c, fiber.StatusBadRequest, "File wajib diupload", nil) }

	filename := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(file.Filename))
	savePath := fmt.Sprintf("./uploads/%s", filename)
	
	if err := c.SaveFile(file, savePath); err != nil { return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan file", err.Error()) }

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil) }
	
	student, _ := repository.FindStudentByUserID(userID)
	if ach.StudentID != student.ID { return helper.Error(c, fiber.StatusForbidden, "Akses ditolak", nil) }
	if ach.Status != model.StatusDraft { return helper.Error(c, fiber.StatusBadRequest, "Harus status draft", nil) }

	fileURL := "http://localhost:3000/uploads/" + filename
	att := model.Attachment{FileName: file.Filename, FileURL: fileURL, FileType: filepath.Ext(filename), UploadedAt: time.Now()}
	
	if err := repository.AddAttachmentToMongo(ach.MongoAchievementID, att); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update database", err.Error())
	}

	return helper.Created(c, fiber.Map{"url": fileURL}, "File berhasil diupload")
}

// --- FR-012: Get History ---
// GET /api/v1/achievements/:id/history
func GetHistory(c *fiber.Ctx) error {
    achievementID := c.Params("id")
    
    // Panggil fungsi repository yang sudah kita buat sebelumnya
    historyData, err := repository.GetAchievementHistory(achievementID)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "History tidak ditemukan", err.Error())
    }

    return helper.Success(c, historyData, "History status berhasil diambil")
}