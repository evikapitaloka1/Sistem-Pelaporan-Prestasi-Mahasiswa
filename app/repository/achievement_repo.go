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
)

// --- FR-003: Submit Prestasi (Hybrid Transaction) ---
func CreateAchievement(ref *model.AchievementReference, detail *model.AchievementMongo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Simpan Detail ke MongoDB
	collection := database.MongoD.Collection("achievements")
	detail.CreatedAt = time.Now()
	detail.UpdatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, detail)
	if err != nil {
		return fmt.Errorf("gagal insert ke mongo: %v", err)
	}

	// Ambil ObjectID dari Mongo
	mongoID := result.InsertedID.(primitive.ObjectID).Hex()

	// 2. Simpan Reference ke PostgreSQL
	// Kita gunakan query parameter ($1, $2, dst) untuk keamanan
	query := `
		INSERT INTO achievement_references (
			id, student_id, mongo_achievement_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err = database.PostgresDB.Exec(query, 
		ref.ID, ref.StudentID, mongoID, model.StatusDraft, time.Now(), time.Now(),
	)

	if err != nil {
		// ROLLBACK MANUAL: Jika Postgres gagal, hapus data di Mongo agar tidak jadi sampah
		_, delErr := collection.DeleteOne(ctx, bson.M{"_id": result.InsertedID})
		if delErr != nil {
			log.Printf("CRITICAL: Gagal rollback mongo (ID: %s): %v", mongoID, delErr)
		}
		return fmt.Errorf("gagal insert ke postgres: %v", err)
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
	// 1. Bangun Query Dasar
	// Kita join ke tabel students agar bisa filter berdasarkan advisor_id (untuk Dosen Wali)
	baseQuery := `
		FROM achievement_references ar
		JOIN students s ON ar.student_id = s.id
		WHERE 1=1
	`

	// 2. Apply Filters
	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND ar.status = '%s'", filter.Status)
	}
	
	// Filter: Mahasiswa hanya lihat punya sendiri
	if filter.StudentID != "" {
		baseQuery += fmt.Sprintf(" AND ar.student_id = '%s'", filter.StudentID)
	}

	// Filter: Dosen Wali hanya lihat mahasiswa bimbingannya (FR-006)
	if filter.AdvisorID != "" {
		baseQuery += fmt.Sprintf(" AND s.advisor_id = '%s'", filter.AdvisorID)
	}

	// 3. Hitung Total Data (untuk Pagination)
	var totalData int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := database.PostgresDB.QueryRow(countQuery).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// 4. Ambil Data dengan Limit & Offset
	selQuery := `
		SELECT ar.id, ar.student_id, ar.mongo_achievement_id, ar.status, 
		       ar.created_at, ar.submitted_at, ar.verified_at 
		` + baseQuery + ` 
		ORDER BY ar.created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	offset := (filter.Page - 1) * filter.Limit
	rows, err := database.PostgresDB.Query(selQuery, filter.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var achievements []model.AchievementReference
	
	// 5. Loop hasil query
	for rows.Next() {
		var ach model.AchievementReference
		err := rows.Scan(
			&ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status, 
			&ach.CreatedAt, &ach.SubmittedAt, &ach.VerifiedAt,
		)
		if err != nil {
			continue
		}
		
		// Opsional: Fetch Judul dari Mongo untuk ditampilkan di list
		// (Bisa diaktifkan jika butuh menampilkan judul di tabel depan)
		/*
		detail, _ := GetAchievementDetailFromMongo(ach.MongoAchievementID)
		if detail != nil {
			log.Println("Title:", detail.Title)
		}
		*/
		
		achievements = append(achievements, ach)
	}

	return achievements, totalData, nil
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

	objID, _ := primitive.ObjectIDFromHex(mongoHexID)
	collection := database.MongoD.Collection("achievements")

	// Update field yang diizinkan
	update := bson.M{
		"$set": bson.M{
			"title":           updateData.Title,
			"description":     updateData.Description,
			"achievementType": updateData.AchievementType,
			"details":         updateData.Details,
			"updatedAt":       time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

// --- FR-Baru: Add Attachment ---
func AddAttachmentToMongo(mongoHexID string, attachment model.Attachment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, _ := primitive.ObjectIDFromHex(mongoHexID)
	collection := database.MongoD.Collection("achievements")

	// Push ke array attachments
	update := bson.M{
		"$push": bson.M{"attachments": attachment},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
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