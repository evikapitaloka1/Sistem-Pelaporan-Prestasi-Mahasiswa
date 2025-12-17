package repository

import (
	"context"
	"fmt"
	"log"
	"sistempelaporan/app/model"
	"sistempelaporan/database"
	"time"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/google/uuid"
	"database/sql"
)

// --- FR-003: Submit Prestasi (Hybrid Transaction) ---
func CreateAchievement(ref *model.AchievementReference, detail *model.AchievementMongo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()

	// 1. Simpan Detail ke MongoDB
	collection := database.MongoD.Collection("achievements")
	
	// Pastikan detail memiliki ID yang valid (biasanya diisi oleh service/model)
	// Kita set waktu di sini agar konsisten:
	detail.CreatedAt = now
	detail.UpdatedAt = now

	// A. Insert ke MongoDB
	result, err := collection.InsertOne(ctx, detail)
	if err != nil {
		return fmt.Errorf("gagal insert ke MongoDB: %w", err) // Menggunakan %w untuk wrapping error
	}

	mongoID := result.InsertedID.(primitive.ObjectID).Hex()

	// 2. Simpan Reference ke PostgreSQL
	query := `
		INSERT INTO achievement_references (
			id, student_id, mongo_achievement_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	// B. Eksekusi ke PostgreSQL
	_, err = database.PostgresDB.Exec(query, 
		ref.ID, ref.StudentID, mongoID, model.StatusDraft, now, now,
	)

	if err != nil {
		// ROLLBACK MANUAL: Jika Postgres gagal, hapus data di Mongo
		log.Printf("ERROR: Insert ke PostgreSQL gagal untuk Achievement ID %s. Melakukan rollback di MongoDB...", ref.ID)
		
		// Attempt to delete the inserted document in Mongo
		_, delErr := collection.DeleteOne(ctx, bson.M{"_id": result.InsertedID})
		
		if delErr != nil {
			// Jika rollback gagal, ini CRITICAL karena terjadi inkonsistensi data
			log.Fatalf("CRITICAL ERROR: Gagal rollback MongoDB (ID: %s). Data tidak konsisten. Error MongoDB: %v", mongoID, delErr)
            // Dalam kasus fatal ini, Anda mungkin ingin melempar error yang lebih jelas atau memicu notifikasi.
		}
        
        // Log keberhasilan rollback
        log.Printf("Rollback MongoDB berhasil untuk ID: %s.", mongoID)
        
		return fmt.Errorf("gagal insert ke PostgreSQL: %w", err)
	}

	return nil
}

// --- Struktur Filter Khusus Repository ---
type RepoFilter struct {
	model.AchievementFilter
	StudentID string // Filter khusus untuk Mahasiswa (lihat punya sendiri)
	AdvisorID string // Filter khusus untuk Dosen Wali (lihat bimbingan)
}

// --- FR-010 & FR-006: Get All Achievements ---
func GetAllAchievements(filter RepoFilter) ([]model.AchievementReference, int64, error) {
    
    var totalData int64 
    var err error

    // 1. Bangun Query Dasar
    baseQuery := `
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        WHERE ar.deleted_at IS NULL
    `
    // 2. Apply Filters (Logika sama seperti sebelumnya)
    if filter.Status != "" {
        baseQuery += fmt.Sprintf(" AND ar.status = '%s'", filter.Status)
    }
    if filter.StudentID != "" {
        baseQuery += fmt.Sprintf(" AND ar.student_id = '%s'", filter.StudentID)
    }
    if filter.AdvisorID != "" {
        baseQuery += fmt.Sprintf(" AND s.advisor_id = '%s'", filter.AdvisorID)
    }

    // 3. Hitung Total Data
    countQuery := "SELECT COUNT(*) " + baseQuery
    err = database.PostgresDB.QueryRow(countQuery).Scan(&totalData) 
    if err != nil {
        return nil, 0, err
    }

    // 4. Ambil Data dengan Limit & Offset
    selQuery := `
        SELECT 
            ar.id, ar.student_id, ar.mongo_achievement_id, ar.status, 
            ar.created_at, ar.updated_at,  
            ar.submitted_at, ar.verified_at, 
            ar.verified_by, ar.rejection_note, 
            ar.deleted_at                     
        ` + baseQuery + ` 
        ORDER BY ar.created_at DESC 
        LIMIT $1 OFFSET $2
    `
    
    offset := (filter.Page - 1) * filter.Limit
    rows, err := database.PostgresDB.Query(selQuery, filter.Limit, offset)
    if err != nil {
        log.Printf("DB Query Error: %v", err)
        return nil, 0, err
    }
    defer rows.Close()

    var achievements []model.AchievementReference
    
    // 5. Loop hasil query dan Scanning (FIX LENGKAP)
    for rows.Next() {
        var ach model.AchievementReference
        
        // Helper Nullables
        var sqlCreatedAt sql.NullTime    
        
        // MENGGUNAKAN interface{} UNTUK SEMUA KOLOM NULLABLE
        var ifUpdatedAt, ifSubmittedAt, ifVerifiedAt, ifDeletedAt interface{}
        var ifVerifiedBy, ifRejectionNote interface{} 
        
        // PEMANGGILAN SCAN (11 field - URUTAN KRITIS)
        err := rows.Scan(
            &ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status, 
            &sqlCreatedAt, &ifUpdatedAt, // CreatedAt (SQL Null), UpdatedAt (Interface)
            &ifSubmittedAt, &ifVerifiedAt, 
            &ifVerifiedBy, &ifRejectionNote, 
            &ifDeletedAt, 
        )
        
        if err != nil {
             log.Printf("CRITICAL SCAN ERROR (GetAll): %v", err) 
             continue
        }
        
        // --- Konversi NULLable Fields ke Model ---
        
        // CreatedAt (Non-pointer)
        if sqlCreatedAt.Valid { ach.CreatedAt = sqlCreatedAt.Time } else { ach.CreatedAt = time.Time{} } 

        // UpdatedAt (*time.Time)
        if ifUpdatedAt != nil { 
            t, parseErr := parseInterfaceTime(ifUpdatedAt)
            if parseErr == nil { ach.UpdatedAt = &t } 
        }
        
        // Time Pointers Lain (*time.Time)
        if ifSubmittedAt != nil { 
            t, parseErr := parseInterfaceTime(ifSubmittedAt)
            if parseErr == nil { ach.SubmittedAt = &t }
        }
        if ifVerifiedAt != nil { 
            t, parseErr := parseInterfaceTime(ifVerifiedAt)
            if parseErr == nil { ach.VerifiedAt = &t }
        }
        if ifDeletedAt != nil { 
            t, parseErr := parseInterfaceTime(ifDeletedAt)
            if parseErr == nil { ach.DeletedAt = &t }
        }
        
        // Rejection Note (*string)
        if ifRejectionNote != nil {
            strVal := getInterfaceString(ifRejectionNote) 
            if strVal != "" { ach.RejectionNote = &strVal }
        }

        // Verified By (*uuid.UUID)
        if ifVerifiedBy != nil {
            uuidStr := getInterfaceString(ifVerifiedBy)
            if uuidStr != "" {
                 uuidVal, parseErr := uuid.Parse(uuidStr)
                 if parseErr == nil { ach.VerifiedBy = &uuidVal }
            }
        }
        
        achievements = append(achievements, ach)
    }
    
    if err = rows.Err(); err != nil {
        return nil, 0, err
    }

    return achievements, totalData, nil
}

// --- Tambahkan 3 fungsi helper ini di bagian bawah file repository Anda ---

func getInterfaceString(data interface{}) string {
    if data == nil { return "" }
    if val, ok := data.([]byte); ok { return string(val) } 
    if val, ok := data.(string); ok { return val }
    return ""
}

func parseInterfaceTime(data interface{}) (time.Time, error) {
    if data == nil { return time.Time{}, errors.New("data is nil") }
    if val, ok := data.(time.Time); ok { return val, nil }
    if val, ok := data.([]byte); ok { 
        return time.Parse("2006-01-02 15:04:05.999999", string(val)) // Format PostgreSQL
    }
    if val, ok := data.(string); ok {
         return time.Parse("2006-01-02 15:04:05.999999", val)
    }
    return time.Time{}, errors.New("unsupported time format")
}
// --- Helper: Ambil Detail Mongo ---
func GetAchievementDetailFromMongo(hexID string) (*model.AchievementMongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(hexID)
	if err != nil {
		return nil, err
	}

	var result model.AchievementMongo
	err = database.MongoD.Collection("achievements").FindOne(ctx, bson.M{"_id": objID}).Decode(&result)
	return &result, err
}

// --- Helper: Cari berdasarkan ID Postgres ---
func FindAchievementByID(id string) (*model.AchievementReference, error) {
	query := `SELECT id, student_id, mongo_achievement_id, status FROM achievement_references WHERE id = $1`
	
	var ach model.AchievementReference
	err := database.PostgresDB.QueryRow(query, id).Scan(
		&ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status,
	)
	return &ach, err
}

// --- FR-004, FR-007, FR-008: Update Status ---
func UpdateStatus(id string, status model.AchievementStatus, verifiedBy *uuid.UUID, note string) error {
    
    statusSubmittedString := "submitted"
    statusVerifiedString := "verified"
    statusRejectedString := "rejected"
    
    // Asumsi nama tipe ENUM di DB adalah 'achievement_status_type'
    cleanQuery := `
        UPDATE achievement_references 
        SET 
            status = $1, 
            verified_by = $2, 
            rejection_note = $3, 
            
            submitted_at = CASE 
                            WHEN $1 = $4::achievement_status_type THEN NOW() 
                            ELSE submitted_at 
                           END,
            
            verified_at = CASE 
                          WHEN $1 = $5::achievement_status_type OR $1 = $6::achievement_status_type THEN NOW() 
                          ELSE verified_at 
                          END,
            
            updated_at = NOW()
        WHERE id = $7
    `
    
    // Konversi ID string ke tipe uuid.UUID
    achievementUUID, err := uuid.Parse(id)
    if err != nil {
        return err // Gagal parsing ID UUID
    }

    // PERBAIKAN 2: Hapus context.Background() jika database.PostgresDB menggunakan database/sql
    // (Line Number 206)
    _, err = database.PostgresDB.Exec(cleanQuery, 
        string(status),                 // $1 (ENUM)
        verifiedBy,                     // $2 (*uuid.UUID)
        note,                           // $3 (TEXT)
        statusSubmittedString,          // $4 (STRING literal)
        statusVerifiedString,           // $5 (STRING literal)
        statusRejectedString,           // $6 (STRING literal)
        achievementUUID,                // $7 (UUID)
    )
    
    if err != nil {
         log.Printf("DB EXEC ERROR (UpdateStatus): %v", err)
    }

    return err
}
// --- FR-005: Delete (Soft/Hard) ---
func SoftDeleteAchievementTransaction(postgresID string, mongoHexID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Soft Delete di PostgreSQL (Reference)
	pgQuery := `
		UPDATE achievement_references 
		SET deleted_at = NOW(),  -- Kolom yang benar untuk soft delete
			updated_at = NOW()  
		WHERE id = $1
	`
    // CATATAN: Baris "status = 'deleted'" SUDAH DIHAPUS untuk menghindari conflict ENUM.
    
	// Gunakan Exec untuk menjalankan UPDATE
    // Asumsi database.PostgresDB.Exec berasal dari database/sql
	result, err := database.PostgresDB.Exec(pgQuery, postgresID)
	
	if err != nil {
		// Error ini kini hanya akan muncul jika kolom deleted_at belum di-ALTER
		return fmt.Errorf("gagal soft delete di Postgres: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("gagal mendapatkan rows affected: %w", err)
    }
	if rowsAffected == 0 {
		return errors.New("achievement reference not found or already deleted")
	}

	// 2. Soft Delete di MongoDB (Data Detail)

	// Konversi Hex ID MongoDB ke ObjectID
	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("mongo ID tidak valid: %w", err)
	}

	// Payload untuk update: Menambahkan field deleted_at
	mongoUpdate := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}
	
	// Panggil UpdateOne untuk menambahkan field deleted_at di dokumen MongoDB
    // Perhatikan penggunaan ctx (wajib untuk MongoDB driver)
	collection := database.MongoD.Collection("achievements") 
	_, err = collection.UpdateOne(
		ctx, 
		bson.M{"_id": objID}, // Filter berdasarkan ObjectID
		mongoUpdate,
	)
	
	if err != nil {
		log.Printf("Gagal soft delete di Mongo (Postgres sukses): %v", err)
		return fmt.Errorf("gagal soft delete di Mongo (Postgres sukses): %w", err)
	}
	
	return nil
}
// --- FR-011: Statistics ---
func GetStatsByStatus() (map[string]int, error) {
	rows, err := database.PostgresDB.Query("SELECT status, COUNT(*) FROM achievement_references GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		stats[status] = count
	}
	return stats, nil
}

// --- FR-Baru: Update Achievement (Mongo Content) ---
func UpdateAchievementDetail(mongoHexID string, updateData model.AchievementMongo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("invalid object id: %w", err)
	}
	collection := database.MongoD.Collection("achievements")

	// 1. BUAT BSON MAPPING DINAMIS
	setFields := bson.M{}

	// Cek dan tambahkan field yang TIDAK kosong atau TIDAK nil

	// Field String
	if updateData.Title != "" {
		setFields["title"] = updateData.Title
	}
	if updateData.Description != "" {
		setFields["description"] = updateData.Description
	}
	if updateData.AchievementType != "" {
		setFields["achievementType"] = updateData.AchievementType
	}
    // Tambahkan Tags dan Points jika diizinkan untuk di-update
    if len(updateData.Tags) > 0 {
        setFields["tags"] = updateData.Tags
    }
    if updateData.Points != 0 {
        setFields["points"] = updateData.Points
    }

	// Field Map (Pastikan tidak nil sebelum ditambahkan)
	if updateData.Details != nil && len(updateData.Details) > 0 {
		setFields["details"] = updateData.Details
	}
    
    // Asumsi: Attachments akan diupdate melalui endpoint/function terpisah

	// 2. Tambahkan UpdatedAt (Wajib)
	setFields["updatedAt"] = time.Now()

	// 3. Eksekusi Update
	if len(setFields) == 0 {
		// Jika tidak ada data yang perlu di-update selain updatedAt (yang sudah ditambahkan)
		// atau jika Anda ingin mengabaikan update jika hanya updatedAt yang berubah.
        // Kita tetap lanjutkan karena updatedAt sudah ada di map.
	}

	update := bson.M{"$set": setFields}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	
    if err != nil {
        return fmt.Errorf("gagal update ke mongo: %w", err)
    }

    if result.ModifiedCount == 0 && result.MatchedCount == 1 {
        // Dokumen cocok, tapi tidak ada yang berubah (misal semua field sama)
        return nil 
    }
    
	return nil
}
// --- FR-Baru: Add Attachment ---
func AddAttachmentToMongo(mongoHexID string, attachment model.Attachment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("invalid MongoDB ID format: %w", err)
	}

	collection := database.MongoD.Collection("achievements")

	// --- 1. KOREKSI TAHAP 1: Memastikan 'attachments' bukan null ---
	// Update ini hanya akan dijalankan jika dokumen ditemukan. 
	// Kita akan menggunakan $set untuk memastikan 'attachments' adalah array kosong JIKA nilainya null.
	// Jika nilainya array, $set ini tidak akan mengubahnya menjadi array kosong.
    
    // Query ini bertujuan untuk memastikan field 'attachments' ada dan bertipe array.
    // Jika 'attachments' benar-benar 'null', $set bisa membuatnya menjadi [].
    // NOTE: MongoDB driver (BSON) biasanya mengirim nil Go sebagai $unset atau null. 
    // Jika null, $set ini akan bekerja.

    _, err = collection.UpdateOne(
        ctx,
        bson.M{"_id": objID, "attachments": nil}, // Filter: ID dan attachments = null
        bson.M{"$set": bson.M{"attachments": []model.Attachment{}}}, // Action: Set ke array kosong
    )
    // Kita tidak perlu memeriksa error di sini karena update ini bersifat pencegahan.

	// --- 2. TAHAP 2: Melakukan $push yang Sesungguhnya ---
	update := bson.M{
		"$push": bson.M{"attachments": attachment},
	}

	// Eksekusi Update $push
	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	
	if err != nil {
		// Log error di sini jika diperlukan
		return fmt.Errorf("gagal menambahkan attachment: %w", err)
	}
	
	return nil
}
// --- FR-012: Status History (Simple Version) ---
// Mengembalikan riwayat waktu perubahan status berdasarkan kolom timestamp yang ada
func GetAchievementHistory(id string) (map[string]interface{}, error) {
    query := `
        SELECT created_at, submitted_at, verified_at, rejection_note, status 
        FROM achievement_references 
        WHERE id = $1
    `
    
    var history struct {
        CreatedAt    time.Time
        SubmittedAt  *time.Time
        VerifiedAt   *time.Time
        RejectionNote *string
        Status       string
    }

    err := database.PostgresDB.QueryRow(query, id).Scan(
        &history.CreatedAt, &history.SubmittedAt, &history.VerifiedAt, 
        &history.RejectionNote, &history.Status,
    )
    if err != nil {
        return nil, err
    }

    // Format output menjadi list event
    var logs []map[string]interface{}

    // 1. Event Dibuat
    logs = append(logs, map[string]interface{}{
        "status": "draft",
        "timestamp": history.CreatedAt,
        "description": "Prestasi dibuat (draft)",
    })

    // 2. Event Disubmit
    if history.SubmittedAt != nil {
        logs = append(logs, map[string]interface{}{
            "status": "submitted",
            "timestamp": *history.SubmittedAt,
            "description": "Diajukan untuk verifikasi",
        })
    }

    // 3. Event Final (Verified/Rejected)
    if history.VerifiedAt != nil {
        desc := "Prestasi telah diverifikasi"
        status := "verified"
        if history.Status == "rejected" {
            desc = "Prestasi ditolak. Alasan: " + *history.RejectionNote
            status = "rejected"
        }
        
        logs = append(logs, map[string]interface{}{
            "status": status,
            "timestamp": *history.VerifiedAt,
            "description": desc,
        })
    }

    return map[string]interface{}{
        "achievement_id": id,
        "history": logs,
    }, nil
}
// GetLecturerIDByUserID mencari ID Dosen di tabel lecturers berdasarkan ID User yang login
func GetLecturerIDByUserID(userID string) (string, error) {
    var lecturerID string
    query := `SELECT id FROM lecturers WHERE user_id = $1`
    
    err := database.PostgresDB.QueryRow(query, userID).Scan(&lecturerID)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", fmt.Errorf("user ini bukan merupakan dosen")
        }
        return "", err
    }
    return lecturerID, nil
}