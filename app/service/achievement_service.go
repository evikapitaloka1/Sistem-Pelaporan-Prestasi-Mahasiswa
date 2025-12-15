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

	// Ambil rawUserID
	rawUserID := c.Locals("user_id")
	if rawUserID == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Sesi tidak valid: User ID hilang", nil)
	}
	userID, ok := rawUserID.(string) 
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Token error: User ID format salah", nil)
	}

	// [PERBAIKAN UTAMA DI SINI]: Ambil rawUserRole menggunakan kunci "role"
	// Sesuai dengan permintaan Anda.
	rawUserRole := c.Locals("role") 
	
	// Safety check: Jika role hilang (nil), berikan nilai default "UNKNOWN"
	if rawUserRole == nil {
		log.Println("WARNING: user_role is nil in locals, defaulting to UNKNOWN.")
		rawUserRole = "UNKNOWN" 
	}
	
	// Konversi role ke string dan amankan
	roleString, ok := rawUserRole.(string)
	if !ok {
		log.Printf("ERROR: User Role type assertion failed, type is %T. Defaulting to UNKNOWN.", rawUserRole)
		roleString = "UNKNOWN"
	}

	// Normalisasi Role yang diterima (misal "Admin" menjadi "ADMIN")
	userRole := strings.ToUpper(roleString)

	achievementID := c.Params("id") // ID PostgreSQL (UUID)

	// 2. Cari Prestasi Berdasarkan ID PostgreSQL
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") { 
			return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", nil)
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mencari prestasi", err.Error())
	}

	// --- 3. LOGIC OTORISASI SUPERADMIN (Admin Only) ---
	
	// Pengecekan FINAL: Hanya Admin yang diizinkan untuk menghapus.
	// Sekarang pengecekan ini akan berhasil bagi Admin karena userRole sudah benar.
	if userRole != "ADMIN" {
		// Mahasiswa, Dosen, atau role lain yang TIDAK memiliki claim "ADMIN" di token akan diblokir.
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya Admin yang dapat menghapus data prestasi.", nil)
	}

	// JIKA SAMPAI DI SINI, PENGGUNA ADALAH ADMIN.
	// Bypass penuh berlaku (tidak perlu cek kepemilikan atau status draft).
	log.Printf("INFO: Admin %s melakukan force soft delete pada prestasi %s", userID, achievementID)

	// 4. Eksekusi Soft Delete Hybrid
	// ... (kode repository.SoftDeleteAchievementTransaction tetap sama) ...
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
	// Ambil userID dari Local (ID user Dosen Wali yang melakukan verifikasi)
	lecturerUserID := c.Locals("user_id").(string)
	achievementID := c.Params("id")

	// 1. Validasi Prestasi
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { 
		log.Printf("Achievement not found: %s, Error: %v", achievementID, err)
		return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error()) 
	}
    
	// 2. Validasi Status (Harus 'submitted')
	if ach.Status != model.StatusSubmitted { 
		return helper.Error(c, fiber.StatusBadRequest, "Prestasi belum diajukan (status harus 'submitted')", nil) 
	}
    
    // 3. [TODO: VALIDASI TAMBAHAN - FR-007]
    // Dosen Wali HANYA boleh memverifikasi prestasi mahasiswa bimbingannya.
    // Anda perlu membandingkan ach.StudentID -> Student -> Student.AdvisorID 
    // dengan Lecturer.ID yang memiliki user_id = lecturerUserID
    // if !repository.IsLecturerAdvisorForStudent(lecturerUserID, ach.StudentID) {
    //     return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Bukan mahasiswa bimbingan Anda", nil)
    // }

	// 4. Konversi ID Verifikator (Dosen Wali) ke *uuid.UUID
	lecturerUUID, err := uuid.Parse(lecturerUserID)
	if err != nil {
		log.Printf("Error parsing lecturer User ID to UUID: %v", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Kesalahan format ID verifikator", err.Error())
	}
	verifiedByPointer := &lecturerUUID 
	
	// 5. Update Status ke Verified
    // verifiedByPointer dilewatkan sebagai parameter $2 (*uuid.UUID)
    // note dikosongkan karena ini adalah verifikasi, bukan penolakan
	if err := repository.UpdateStatus(
        achievementID, 
        model.StatusVerified, 
        verifiedByPointer, 
        "" /* note kosong */); err != nil {
		
		log.Printf("Failed to update status to Verified: %v", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal verifikasi prestasi", err.Error())
	}

	// 6. [TODO: Implementasi FR-007] - Buat notifikasi untuk Mahasiswa
	// repository.CreateNotification(ach.StudentID, ach.ID, "Prestasi Anda telah diverifikasi")

	return helper.Success(c, nil, "Prestasi berhasil diverifikasi")
}

// --- FR-008: Reject ---
func RejectAchievement(c *fiber.Ctx) error {
	// Ambil userID dari Local (ID user Dosen Wali yang menolak)
	lecturerUserID := c.Locals("user_id").(string)
	achievementID := c.Params("id")
	
	// 1. Parsing Request Body (Rejection Note)
	var req model.RejectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	
	if req.RejectionNote == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Catatan penolakan (Rejection Note) wajib diisi", nil)
	}

	// 2. Validasi Prestasi
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { 
		log.Printf("Achievement not found: %s, Error: %v", achievementID, err)
		return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error()) 
	}
    

	if ach.Status != model.StatusSubmitted { 
		return helper.Error(c, fiber.StatusBadRequest, "Prestasi belum diajukan (status harus 'submitted')", nil) 
	}
    
    // 4. [TODO: VALIDASI TAMBAHAN - FR-008]
    // Dosen Wali HANYA boleh menolak prestasi mahasiswa bimbingannya.

	// 5. Konversi ID Verifikator (Dosen Wali) ke *uuid.UUID
    // Ini adalah KOREKSI FINAL untuk Error Baris 298
	lecturerUUID, err := uuid.Parse(lecturerUserID)
	if err != nil {
		log.Printf("Error parsing lecturer User ID to UUID: %v", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Kesalahan format ID verifikator", err.Error())
	}
	verifiedByPointer := &lecturerUUID // Tipe: *uuid.UUID. Ini yang benar.
	

    // Parameter verifiedByPointer (UUID) dan req.RejectionNote (TEXT) dilewatkan.
	if err := repository.UpdateStatus(
        achievementID, 
        model.StatusRejected, 
        verifiedByPointer, 
        req.RejectionNote); err != nil {
		
		log.Printf("Failed to update status to Rejected: %v", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menolak prestasi", err.Error())
	}

	
	// repository.CreateNotification(ach.StudentID, ach.ID, "Prestasi Anda ditolak: " + req.RejectionNote)

	return helper.Success(c, nil, "Prestasi berhasil ditolak dengan catatan")
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