package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sistempelaporan/app/model"
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// SubmitAchievement godoc
// @Summary      Submit Prestasi Baru
// @Description  Mahasiswa atau Admin dapat menambahkan laporan prestasi baru ke dalam sistem.
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        achievement  body      model.AchievementMongo  true  "Data Prestasi (MongoDB structure)"
// @Success      201          {object}  helper.Response
// @Failure      400          {object}  helper.Response
// @Router       /achievements [post]
// @Security     BearerAuth
func SubmitAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)

	var req model.AchievementMongo
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	var targetStudentID uuid.UUID

	if strings.EqualFold(role, "Admin") {
		if req.StudentID == "" {
			return helper.Error(c, fiber.StatusBadRequest, "Admin wajib menyertakan 'studentId' Mahasiswa target di body request.", nil)
		}
		studentUUID, err := uuid.Parse(req.StudentID)
		if err != nil {
			return helper.Error(c, fiber.StatusBadRequest, "Format Student ID target tidak valid.", nil)
		}
		targetStudentID = studentUUID
	} else if strings.EqualFold(role, "Mahasiswa") {
		student, err := repository.FindStudentByUserID(userID)
		if err != nil || student.ID == uuid.Nil {
			return helper.Error(c, fiber.StatusForbidden, "Profil mahasiswa tidak ditemukan atau terhubung.", nil)
		}
		targetStudentID = student.ID
	} else {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Hanya Mahasiswa dan Admin yang dapat membuat prestasi.", nil)
	}

	refID := uuid.New()
	now := time.Now()

	ref := model.AchievementReference{
		ID:        refID,
		StudentID: targetStudentID,
		Status:    model.StatusDraft,
	}

	req.StudentID = targetStudentID.String()
	req.CreatedAt = now
	req.UpdatedAt = now

	if err := repository.CreateAchievement(&ref, &req); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan prestasi", err.Error())
	}

	return helper.Created(c, fiber.Map{"id": refID.String()}, "Prestasi berhasil dibuat")
}

// GetListAchievements godoc
// @Summary      Lihat Daftar Prestasi
// @Description  Mengambil daftar prestasi dengan filter role. Mahasiswa melihat miliknya, Dosen Wali melihat bimbingannya, Admin melihat semua.
// @Tags         Achievements
// @Produce      json
// @Param        page    query     int     false  "Halaman"
// @Param        limit   query     int     false  "Jumlah per halaman"
// @Param        status  query     string  false  "Filter Status (draft, submitted, verified, rejected)"
// @Success      200     {object}  helper.Response
// @Router       /achievements [get]
// @Security     BearerAuth
func GetListAchievements(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	roleName := c.Locals("role").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	filter := model.AchievementFilter{
		Page:   page,
		Limit:  limit,
		Status: c.Query("status"),
	}

	repoFilter := repository.RepoFilter{AchievementFilter: filter}

	if roleName == "Mahasiswa" {
		student, err := repository.FindStudentByUserID(userID)
		if err != nil {
			return helper.Error(c, fiber.StatusForbidden, "Profil mahasiswa invalid", err.Error())
		}
		repoFilter.StudentID = student.ID.String()
	} else if roleName == "Dosen Wali" {
		lecturer, err := repository.FindLecturerByUserID(userID)
		if err != nil {
			return helper.Error(c, fiber.StatusForbidden, "Profil dosen invalid", err.Error())
		}
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

// GetAchievementDetail godoc
// @Summary      Dapatkan Detail Prestasi
// @Description  Mengambil data referensi dan detail konten prestasi.
// @Tags         Achievements
// @Param        id   path      string  true  "Achievement ID (UUID)"
// @Success      200  {object}  helper.Response
// @Router       /achievements/{id} [get]
// @Security     BearerAuth
func GetAchievementDetail(c *fiber.Ctx) error {
	achievementID := c.Params("id")

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Data referensi tidak ditemukan", err.Error())
	}

	detail, err := repository.GetAchievementDetailFromMongo(ach.MongoAchievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Detail konten tidak ditemukan", err.Error())
	}

	resp := map[string]interface{}{
		"reference": ach,
		"detail":    detail,
	}
	return helper.Success(c, resp, "Detail prestasi")
}

// UpdateAchievement godoc
// @Summary      Update Konten Prestasi
// @Description  Memperbarui data prestasi yang masih berstatus 'draft'.
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Achievement ID"
// @Param        body  body      model.AchievementMongo  true  "Data Update"
// @Success      200   {object}  helper.Response
// @Router       /achievements/{id} [put]
// @Security     BearerAuth
func UpdateAchievement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)
	achievementID := c.Params("id")

	var newData model.AchievementMongo
	if err := c.BodyParser(&newData); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Request body invalid", nil)
	}

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Data prestasi tidak ditemukan", nil)
	}

	isAuthorized := false
	if strings.EqualFold(role, "Admin") {
		isAuthorized = true
	} else {
		student, findErr := repository.FindStudentByUserID(userID)
		if findErr == nil && student.ID != uuid.Nil && ach.StudentID == student.ID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda tidak memiliki hak edit resource ini.", nil)
	}

	if ach.Status != model.StatusDraft {
		return helper.Error(c, fiber.StatusBadRequest, "Prestasi hanya dapat diedit saat berstatus Draft.", nil)
	}

	if err := repository.UpdateAchievementDetail(ach.MongoAchievementID, newData); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update data", err.Error())
	}

	return helper.Success(c, nil, "Data berhasil diperbarui")
}

// DeleteAchievement godoc
// @Summary      Hapus Prestasi
// @Description  Admin melakukan soft delete pada data prestasi di Postgres dan MongoDB.
// @Tags         Achievements
// @Param        id   path      string  true  "Achievement ID"
// @Success      200  {object}  helper.Response
// @Router       /achievements/{id} [delete]
// @Security     BearerAuth
func DeleteAchievement(c *fiber.Ctx) error {
    // Ambil data dari Locals (Middleware)
    userID := c.Locals("user_id").(string)
    roleString := c.Locals("role").(string)
    userRole := strings.ToUpper(roleString)
    
    achievementID := c.Params("id")

    // Cari datanya dulu
    ach, err := repository.FindAchievementByID(achievementID)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", nil)
    }

    // LOGIKA OTORISASI
    // 1. Ambil StudentID milik user yang sedang login
    mhsID, _ := repository.GetStudentIDByUserID(userID)
    
    // 2. Cek apakah dia Admin atau Pemilik data
    isOwner := (ach.StudentID.String() == mhsID)
    isAdmin := (userRole == "ADMIN")

    if !isAdmin && !isOwner {
        return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda tidak memiliki wewenang.", nil)
    }

    // 3. (Opsional) Mahasiswa tidak boleh hapus jika sudah diverifikasi
   // Pastikan membandingkan dengan nilai string "draft"
if strings.ToLower(string(ach.Status)) != "draft" {
    return helper.Error(c, fiber.StatusForbidden, "Hanya data berstatus DRAFT yang boleh dihapus.", nil)
}
    // Jalankan penghapusan
    if err := repository.SoftDeleteAchievementTransaction(achievementID, ach.MongoAchievementID); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus data", err.Error())
    }

    return helper.Success(c, nil, "Prestasi berhasil dihapus")
}

// RequestVerification godoc
// @Summary      Ajukan Verifikasi
// @Description  Mengubah status prestasi dari 'draft' menjadi 'submitted' untuk diperiksa Dosen Wali.
// @Tags         Achievements
// @Param        id   path      string  true  "Achievement ID"
// @Success      200  {object}  helper.Response
// @Router       /achievements/{id}/submit [post]
// @Security     BearerAuth
func RequestVerification(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)
	achievementID := c.Params("id")

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error())
	}

	if ach.Status != model.StatusDraft {
		return helper.Error(c, fiber.StatusBadRequest, "Hanya status draft yang bisa diajukan", nil)
	}

	isAuthorized := false
	if strings.EqualFold(role, "Admin") {
		isAuthorized = true
	} else if strings.EqualFold(role, "Mahasiswa") {
		student, findErr := repository.FindStudentByUserID(userID)
		if findErr == nil && student.ID != uuid.Nil && ach.StudentID == student.ID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak: Anda bukan pemilik prestasi ini.", nil)
	}

	if err := repository.UpdateStatus(achievementID, model.StatusSubmitted, nil, ""); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update status", err.Error())
	}

	return helper.Success(c, nil, "Berhasil diajukan untuk verifikasi")
}

// VerifyAchievement godoc
// @Summary      Setujui Prestasi
// @Description  Dosen Wali menyetujui prestasi mahasiswa bimbingannya.
// @Tags         Achievements
// @Param        id   path      string  true  "Achievement ID"
// @Success      200  {object}  helper.Response
// @Router       /achievements/{id}/verify [post]
// @Security     BearerAuth
func VerifyAchievement(c *fiber.Ctx) error {
    lecturerUserID := c.Locals("user_id").(string)
    achievementID := c.Params("id")

    // 1. Cari referensi prestasi
    ach, err := repository.FindAchievementByID(achievementID)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error())
    }

    // 2. Ambil detail dari MongoDB untuk menghitung poin
    detail, err := repository.GetAchievementDetailFromMongo(ach.MongoAchievementID)
    if err != nil {
         return helper.Error(c, fiber.StatusNotFound, "Detail prestasi tidak ditemukan", nil)
    }

    // 3. LOGIKA POIN: Hitung poin berdasarkan tingkat kompetisi (CompetitionTier)
    // Misal: Internasional = 100, Nasional = 50, dst.
    finalPoints := 0
    switch strings.ToLower(detail.CompetitionTier) {
    case "internasional":
        finalPoints = 100
    case "nasional":
        finalPoints = 50
    default:
        finalPoints = 10
    }

    // 4. Update Poin di MongoDB
    updateData := model.AchievementMongo{Points: finalPoints}
    repository.UpdateAchievementDetail(ach.MongoAchievementID, updateData)

    // 5. Update Status di Postgres (Verified)
    lecturerUUID, _ := uuid.Parse(lecturerUserID)
    if err := repository.UpdateStatus(achievementID, model.StatusVerified, &lecturerUUID, ""); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal verifikasi", err.Error())
    }

    return helper.Success(c, fiber.Map{"points_awarded": finalPoints}, "Prestasi diverifikasi dan poin diberikan")
}
// RejectAchievement godoc
// @Summary      Tolak Prestasi
// @Description  Dosen Wali menolak prestasi dengan catatan alasan penolakan.
// @Tags         Achievements
// @Accept       json
// @Param        id    path      string              true  "Achievement ID"
// @Param        body  body      model.RejectRequest  true  "Catatan Penolakan"
// @Success      200   {object}  helper.Response
// @Router       /achievements/{id}/reject [post]
// @Security     BearerAuth
func RejectAchievement(c *fiber.Ctx) error {
	lecturerUserID := c.Locals("user_id").(string)
	achievementID := c.Params("id")

	var req model.RejectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	if req.RejectionNote == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Catatan penolakan wajib diisi", nil)
	}

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Prestasi tidak ditemukan", err.Error())
	}

	if ach.Status != model.StatusSubmitted {
		return helper.Error(c, fiber.StatusBadRequest, "Status harus 'submitted'", nil)
	}

	lecturerUUID, _ := uuid.Parse(lecturerUserID)
	verifiedByPointer := &lecturerUUID

	if err := repository.UpdateStatus(achievementID, model.StatusRejected, verifiedByPointer, req.RejectionNote); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menolak prestasi", err.Error())
	}

	return helper.Success(c, nil, "Prestasi berhasil ditolak")
}

// UploadAttachment godoc
// @Summary      Upload Lampiran
// @Description  Mengunggah file bukti prestasi (sertifikat/foto) ke server.
// @Tags         Achievements
// @Accept       mpfd
// @Param        id    path      string  true  "Achievement ID"
// @Param        file  formData  file    true  "File Lampiran"
// @Success      201   {object}  helper.Response
// @Router       /achievements/{id}/attachments [post]
// @Security     BearerAuth
func UploadAttachment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)
	achievementID := c.Params("id")

	file, err := c.FormFile("file")
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "File wajib diupload", nil)
	}

	filename := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(file.Filename))
	savePath := fmt.Sprintf("./uploads/%s", filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan file", err.Error())
	}

	ach, err := repository.FindAchievementByID(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil)
	}

	isAuthorized := false
	if strings.EqualFold(role, "Admin") {
		isAuthorized = true
	} else if strings.EqualFold(role, "Mahasiswa") {
		student, findErr := repository.FindStudentByUserID(userID)
		if findErr == nil && student.ID != uuid.Nil && ach.StudentID == student.ID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return helper.Error(c, fiber.StatusForbidden, "Akses ditolak", nil)
	}

	if ach.Status != model.StatusDraft {
		return helper.Error(c, fiber.StatusBadRequest, "Status harus Draft", nil)
	}

	fileURL := "http://localhost:3000/uploads/" + filename
	att := model.Attachment{FileName: file.Filename, FileURL: fileURL, FileType: filepath.Ext(filename), UploadedAt: time.Now()}

	if err := repository.AddAttachmentToMongo(ach.MongoAchievementID, att); err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal update database", err.Error())
	}

	return helper.Created(c, fiber.Map{"url": fileURL}, "File berhasil diupload")
}

// GetHistory godoc
// @Summary      Riwayat Status Prestasi
// @Description  Melihat log perubahan status prestasi.
// @Tags         Achievements
// @Param        id   path      string  true  "Achievement ID"
// @Success      200  {object}  helper.Response
// @Router       /achievements/{id}/history [get]
// @Security     BearerAuth
func GetHistory(c *fiber.Ctx) error {
	achievementID := c.Params("id")
	historyData, err := repository.GetAchievementHistory(achievementID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "History tidak ditemukan", err.Error())
	}
	return helper.Success(c, historyData, "History status berhasil diambil")
}