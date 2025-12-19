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


func CreateAchievement(ref *model.AchievementReference, detail *model.AchievementMongo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()

	
	collection := database.MongoD.Collection("achievements")
	
	
	detail.CreatedAt = now
	detail.UpdatedAt = now

	
	result, err := collection.InsertOne(ctx, detail)
	if err != nil {
		return fmt.Errorf("gagal insert ke MongoDB: %w", err) 
	}

	mongoID := result.InsertedID.(primitive.ObjectID).Hex()

	query := `
		INSERT INTO achievement_references (
			id, student_id, mongo_achievement_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	
	_, err = database.PostgresDB.Exec(query, 
		ref.ID, ref.StudentID, mongoID, model.StatusDraft, now, now,
	)

	if err != nil {
	
		log.Printf("ERROR: Insert ke PostgreSQL gagal untuk Achievement ID %s. Melakukan rollback di MongoDB...", ref.ID)
		
		
		_, delErr := collection.DeleteOne(ctx, bson.M{"_id": result.InsertedID})
		
		if delErr != nil {
			
			log.Fatalf("CRITICAL ERROR: Gagal rollback MongoDB (ID: %s). Data tidak konsisten. Error MongoDB: %v", mongoID, delErr)
            
		}
        
  
        log.Printf("Rollback MongoDB berhasil untuk ID: %s.", mongoID)
        
		return fmt.Errorf("gagal insert ke PostgreSQL: %w", err)
	}

	return nil
}


type RepoFilter struct {
	model.AchievementFilter
	StudentID string 
	AdvisorID string 
}


func GetAllAchievements(filter RepoFilter) ([]model.AchievementReference, int64, error) {
    
    var totalData int64 
    var err error

    
    baseQuery := `
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        WHERE ar.deleted_at IS NULL
    `
    
    if filter.Status != "" {
        baseQuery += fmt.Sprintf(" AND ar.status = '%s'", filter.Status)
    }
    if filter.StudentID != "" {
        baseQuery += fmt.Sprintf(" AND ar.student_id = '%s'", filter.StudentID)
    }
    if filter.AdvisorID != "" {
        baseQuery += fmt.Sprintf(" AND s.advisor_id = '%s'", filter.AdvisorID)
    }

    
    countQuery := "SELECT COUNT(*) " + baseQuery
    err = database.PostgresDB.QueryRow(countQuery).Scan(&totalData) 
    if err != nil {
        return nil, 0, err
    }

   
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
    
   
    for rows.Next() {
        var ach model.AchievementReference
        
     
        var sqlCreatedAt sql.NullTime    

        var ifUpdatedAt, ifSubmittedAt, ifVerifiedAt, ifDeletedAt interface{}
        var ifVerifiedBy, ifRejectionNote interface{} 
        
     
        err := rows.Scan(
            &ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status, 
            &sqlCreatedAt, &ifUpdatedAt, 
            &ifSubmittedAt, &ifVerifiedAt, 
            &ifVerifiedBy, &ifRejectionNote, 
            &ifDeletedAt, 
        )
        
        if err != nil {
             log.Printf("CRITICAL SCAN ERROR (GetAll): %v", err) 
             continue
        }
        
        
        if sqlCreatedAt.Valid { ach.CreatedAt = sqlCreatedAt.Time } else { ach.CreatedAt = time.Time{} } 

        
        if ifUpdatedAt != nil { 
            t, parseErr := parseInterfaceTime(ifUpdatedAt)
            if parseErr == nil { ach.UpdatedAt = &t } 
        }
        
       
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
        
        
        if ifRejectionNote != nil {
            strVal := getInterfaceString(ifRejectionNote) 
            if strVal != "" { ach.RejectionNote = &strVal }
        }

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


func FindAchievementByID(id string) (*model.AchievementReference, error) {
	query := `SELECT id, student_id, mongo_achievement_id, status FROM achievement_references WHERE id = $1`
	
	var ach model.AchievementReference
	err := database.PostgresDB.QueryRow(query, id).Scan(
		&ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status,
	)
	return &ach, err
}

func UpdateStatus(id string, status model.AchievementStatus, verifiedBy *uuid.UUID, note string) error {
    
    statusSubmittedString := "submitted"
    statusVerifiedString := "verified"
    statusRejectedString := "rejected"

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
    
    
    achievementUUID, err := uuid.Parse(id)
    if err != nil {
        return err 
    }

    
    _, err = database.PostgresDB.Exec(cleanQuery, 
        string(status),                
        verifiedBy,                     
        note,                           
        statusSubmittedString,          
        statusVerifiedString,           
        statusRejectedString,           
        achievementUUID,                
    )
    
    if err != nil {
         log.Printf("DB EXEC ERROR (UpdateStatus): %v", err)
    }

    return err
}

func SoftDeleteAchievementTransaction(postgresID string, mongoHexID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()


	pgQuery := `
		UPDATE achievement_references 
		SET deleted_at = NOW(),  -- Kolom yang benar untuk soft delete
			updated_at = NOW()  
		WHERE id = $1
	`

    
	
	result, err := database.PostgresDB.Exec(pgQuery, postgresID)
	
	if err != nil {
	
		return fmt.Errorf("gagal soft delete di Postgres: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("gagal mendapatkan rows affected: %w", err)
    }
	if rowsAffected == 0 {
		return errors.New("achievement reference not found or already deleted")
	}

	
	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("mongo ID tidak valid: %w", err)
	}


	mongoUpdate := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}
	
	
	collection := database.MongoD.Collection("achievements") 
	_, err = collection.UpdateOne(
		ctx, 
		bson.M{"_id": objID}, 
		mongoUpdate,
	)
	
	if err != nil {
		log.Printf("Gagal soft delete di Mongo (Postgres sukses): %v", err)
		return fmt.Errorf("gagal soft delete di Mongo (Postgres sukses): %w", err)
	}
	
	return nil
}

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


func UpdateAchievementDetail(mongoHexID string, updateData model.AchievementMongo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("invalid object id: %w", err)
	}
	collection := database.MongoD.Collection("achievements")


	setFields := bson.M{}

	
	if updateData.Title != "" {
		setFields["title"] = updateData.Title
	}
	if updateData.Description != "" {
		setFields["description"] = updateData.Description
	}
	if updateData.AchievementType != "" {
		setFields["achievementType"] = updateData.AchievementType
	}

    if len(updateData.Tags) > 0 {
        setFields["tags"] = updateData.Tags
    }
    if updateData.Points != 0 {
        setFields["points"] = updateData.Points
    }

	if updateData.Details != nil && len(updateData.Details) > 0 {
		setFields["details"] = updateData.Details
	}



	setFields["updatedAt"] = time.Now()

	if len(setFields) == 0 {
		
	}

	update := bson.M{"$set": setFields}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	
    if err != nil {
        return fmt.Errorf("gagal update ke mongo: %w", err)
    }

    if result.ModifiedCount == 0 && result.MatchedCount == 1 {
       
        return nil 
    }
    
	return nil
}

func AddAttachmentToMongo(mongoHexID string, attachment model.Attachment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(mongoHexID)
	if err != nil {
		return fmt.Errorf("invalid MongoDB ID format: %w", err)
	}

	collection := database.MongoD.Collection("achievements")


    _, err = collection.UpdateOne(
        ctx,
        bson.M{"_id": objID, "attachments": nil}, // Filter: ID dan attachments = null
        bson.M{"$set": bson.M{"attachments": []model.Attachment{}}}, // Action: Set ke array kosong
    )
   
	update := bson.M{
		"$push": bson.M{"attachments": attachment},
	}


	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	
	if err != nil {
		
		return fmt.Errorf("gagal menambahkan attachment: %w", err)
	}
	
	return nil
}

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


    var logs []map[string]interface{}

    logs = append(logs, map[string]interface{}{
        "status": "draft",
        "timestamp": history.CreatedAt,
        "description": "Prestasi dibuat (draft)",
    })


    if history.SubmittedAt != nil {
        logs = append(logs, map[string]interface{}{
            "status": "submitted",
            "timestamp": *history.SubmittedAt,
            "description": "Diajukan untuk verifikasi",
        })
    }

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