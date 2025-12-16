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
	role := c.Locals("role").(string) // Ambil role dari Locals
	
    // PERHATIAN: Asumsi req.StudentID akan diisi dari body jika ada.
	var req model.AchievementMongo 
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	var targetStudentID uuid.UUID
	
	// 1. LOGIKA PENENTUAN MAHASISWA TARGET BERDASARKAN ROLE
	if strings.EqualFold(role, "Admin") {
		// KASUS ADMIN
		if req.StudentID == "" {
			// Admin harus secara eksplisit menentukan ID Mahasiswa target
			return helper.Error(c, fiber.StatusBadRequest, "Admin wajib menyertakan 'studentId' Mahasiswa target di body request.", nil)
		}
		
		// Konversi StudentID dari body (string) ke UUID
		studentUUID, err := uuid.Parse(req.StudentID)
		if err != nil {
			return helper.Error(c, fiber.StatusBadRequest, "Format Student ID target tidak valid.", nil)
		}
		targetStudentID = studentUUID
        
        // Opsional: Cek apakah ID Mahasiswa target benar-benar ada di database

	} else if strings.EqualFold(role, "Mahasiswa") {
		// KASUS MAHASISWA
		
		// Identifikasi Mahasiswa dari token (Logic lama, tapi sekarang aman)
		student, err := repository.FindStudentByUserID(userID)
		if err != nil || student.ID == uuid.Nil { // Mencegah panic jika student nil
			// Ini adalah error yang muncul jika user bukan mahasiswa
			return helper.Error(c, fiber.StatusForbidden, "Profil mahasiswa tidak ditemukan atau terhubung.", nil)
		}
		targetStudentID = student.ID

	} else {
        // Role lain (Dosen Wali, dll) tidak diizinkan membuat prestasi
        return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya Mahasiswa dan Admin yang dapat membuat prestasi.", nil)
    }

	// 2. Prepare Data (Menggunakan targetStudentID yang sudah diverifikasi)
	refID := uuid.New()
    now := time.Now()

    // Data Reference (Postgres)
	ref := model.AchievementReference{
		ID: refID,
		StudentID: targetStudentID, // <--- ID yang benar (dari Admin/Mahasiswa)
        Status:    model.StatusDraft, // Asumsi status awal adalah DRAFT
        // Tambahkan CreatedAt dan UpdatedAt jika diperlukan di struct Anda
        // CreatedAt: now,
        // UpdatedAt: now,
	}
    
    // Data Detail (Mongo)
	req.StudentID = targetStudentID.String() // Pastikan detail Mongo juga memiliki ID yang benar
    req.CreatedAt = now
    req.UpdatedAt = now
	
	// 3. Simpan ke Repo
	if err := repository.CreateAchievement(&ref, &req); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan prestasi", err.Error())
	}

	return helper.Created(c, fiber.Map{"id": refID.String()}, "Prestasi berhasil dibuat")
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
    role := c.Locals("role").(string)
    achievementID := c.Params("id")
    
    // 1. Binding Data Baru
    var newData model.AchievementMongo
    if err := c.BodyParser(&newData); err != nil { 
        return helper.Error(c, fiber.StatusBadRequest, "Request body invalid", nil) 
    }

    // 2. Cari Data Achievement (Reference)
    ach, err := repository.FindAchievementByID(achievementID)
    if err != nil { 
        return helper.Error(c, fiber.StatusNotFound, "Data prestasi tidak ditemukan", nil) 
    }
    
    // 3. LOGIKA OTORISASI (Menggunakan early return)
    // Asumsikan Admin memiliki hak penuh untuk mengupdate resource apa pun.
    isAuthorized := false
    
    if strings.EqualFold(role, "Admin") {
        isAuthorized = true // Admin selalu diotorisasi
    
    } else {
        // KASUS NON-ADMIN (Hanya Mahasiswa yang bisa update, harus pemilik)
        
        student, findErr := repository.FindStudentByUserID(userID)
        // Jika user yang login bukan Mahasiswa (Dosen, dll) atau tidak terhubung
        if findErr != nil || student.ID == uuid.Nil {
            // Jika user tidak ditemukan sebagai mahasiswa, biarkan isAuthorized = false
            isAuthorized = false 
        } else if ach.StudentID == student.ID {
            // Jika user adalah Mahasiswa pemilik
            isAuthorized = true
        }
    }
    
    if !isAuthorized {
        return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda tidak memiliki hak edit resource ini.", nil)
    }
    
    // 4. Cek Status (Harus Draft, Berlaku untuk Admin dan Mahasiswa)
    if ach.Status != model.StatusDraft { 
        return helper.Error(c, fiber.StatusBadRequest, "Prestasi hanya dapat diedit saat berstatus Draft.", nil) 
    }

    // 5. Update Timestamp dan Eksekusi
    // ... (Set updated_at di newData)
    
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
	// Ambil role dari Locals. Pastikan middleware Anda menyediakannya.
	role := c.Locals("role").(string) 
	
	achievementID := c.Params("id")

	// 1. Cari Data Achievement
	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil { 
		return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error()) 
	}
	
	// 2. Cek Status DRAFT (Berlaku untuk semua role)
	if ach.Status != model.StatusDraft { 
		return helper.Error(c, fiber.StatusBadRequest, "Hanya status draft yang bisa diajukan", nil) 
	}

	// 3. LOGIKA OTORISASI (Admin vs Mahasiswa)
	isAuthorized := false

	// Cek jika user adalah ADMIN
	if strings.EqualFold(role, "Admin") {
		// Jika Admin, langsung diizinkan untuk mengajukan verifikasi.
		isAuthorized = true 
	} 
	
	// Cek jika user adalah MAHASISWA
	if strings.EqualFold(role, "Mahasiswa") {
		// Cari ID Mahasiswa yang login
		student, findErr := repository.FindStudentByUserID(userID)
		
		// Jika ada error (misal 'no rows' yang tidak diharapkan untuk role Mahasiswa)
		// atau ID Mahasiswa tidak valid.
		if findErr != nil || student.ID == uuid.Nil { 
			return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Profil mahasiswa tidak ditemukan.", findErr.Error()) 
		}
		
		// Cek kepemilikan
		if ach.StudentID == student.ID {
			isAuthorized = true
		}
	}
	
	// 4. Final Check Otorisasi
	if !isAuthorized { 
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda bukan pemilik prestasi ini.", nil) 
	}

	// 5. Update Status
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
    role := c.Locals("role").(string) // <--- AMBIL ROLE
    achievementID := c.Params("id")

    file, err := c.FormFile("file")
    if err != nil { return helper.Error(c, fiber.StatusBadRequest, "File wajib diupload", nil) }

    // ... (File saving logic remains the same) ...
    filename := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(file.Filename))
    savePath := fmt.Sprintf("./uploads/%s", filename)
    
    if err := c.SaveFile(file, savePath); err != nil { return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan file", err.Error()) }

    // 1. Cari Achievement
    ach, err := repository.FindAchievementByID(achievementID)
    if err != nil { return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil) }
    
    // 2. LOGIKA OTORISASI (Mencegah Panic)
    isAuthorized := false

    if strings.EqualFold(role, "Admin") {
        isAuthorized = true // Admin selalu diizinkan
    } else if strings.EqualFold(role, "Mahasiswa") {
        // KASUS MAHASISWA: Harus pemilik
        
        student, findErr := repository.FindStudentByUserID(userID)
        
        // Cek error/nil dengan aman
        if findErr == nil && student.ID != uuid.Nil && ach.StudentID == student.ID {
            isAuthorized = true
        }
    }
    
    // 3. Final Check Otorisasi
    if !isAuthorized { 
        return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda tidak memiliki hak untuk mengupload attachment ini.", nil)
    }

    // 4. Cek Status (Harus Draft, Berlaku untuk Admin dan Mahasiswa)
    if ach.Status != model.StatusDraft { 
        return helper.Error(c, fiber.StatusBadRequest, "Hanya status draft yang bisa diubah (upload attachment).", nil) 
    }

    // ... (Logic untuk update database) ...
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