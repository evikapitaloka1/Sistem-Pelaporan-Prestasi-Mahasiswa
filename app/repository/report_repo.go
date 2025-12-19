package repository

import (
	"context"
	"fmt"
	"time"

	"sistempelaporan/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)


func GetTopStudentsStats(targetID string, role string) ([]map[string]interface{}, error) {
	
	filterClause := "WHERE ar.status = 'verified' AND ar.deleted_at IS NULL "
	var params []interface{}

	if role == "Mahasiswa" {
		filterClause += "AND s.id = $1 "
		params = append(params, targetID)
	} else if role == "Dosen Wali" {
		filterClause += "AND s.advisor_id = $1 "
		params = append(params, targetID)
	}

	query := fmt.Sprintf(`
        SELECT u.full_name, s.student_id, COUNT(ar.id) as total_achievements
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        JOIN users u ON s.user_id = u.id
        %s
        GROUP BY u.full_name, s.student_id
        ORDER BY total_achievements DESC
        LIMIT 5
    `, filterClause)

	rows, err := database.PostgresDB.Query(query, params...)
	if err != nil { return nil, err }
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var name, nim string
		var total int
		rows.Scan(&name, &nim, &total)
		results = append(results, map[string]interface{}{
			"name": name, "nim": nim, "total_verified": total,
		})
	}
	return results, nil
}


func GetMonthlyTrendStats(targetID string, role string) ([]map[string]interface{}, error) {
	filterClause := "WHERE status IN ('submitted', 'verified') AND deleted_at IS NULL "
	var params []interface{}

	if role == "Mahasiswa" {
		filterClause += "AND student_id = $1 "
		params = append(params, targetID)
	} else if role == "Dosen Wali" {
		
		filterClause = `
            JOIN students s ON achievement_references.student_id = s.id 
            WHERE status IN ('submitted', 'verified') 
            AND s.advisor_id = $1 AND deleted_at IS NULL `
		params = append(params, targetID)
	}

	query := fmt.Sprintf(`
        SELECT TO_CHAR(submitted_at, 'YYYY-MM') as month, COUNT(*) 
        FROM achievement_references 
        %s
        GROUP BY month 
        ORDER BY month DESC 
        LIMIT 12
    `, filterClause)

	rows, err := database.PostgresDB.Query(query, params...)
	if err != nil { return nil, err }
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var month string
		var count int
		if err := rows.Scan(&month, &count); err != nil { continue }
		results = append(results, map[string]interface{}{"month": month, "count": count})
	}
	return results, nil
}


func GetAchievementTypeDistribution(targetID string, role string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.MongoD.Collection("achievements")

	
	matchStage := bson.D{{Key: "achievementType", Value: bson.D{{Key: "$ne", Value: ""}}}}
	
	
	if role == "Mahasiswa" {
		matchStage = append(matchStage, bson.E{Key: "student_id", Value: targetID})
	} else if role == "Dosen Wali" {
		
		matchStage = append(matchStage, bson.E{Key: "advisor_id", Value: targetID})
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "type", Value: "$achievementType"},
				{Key: "level", Value: "$competitionTier"}, 
			}}, 
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil { return nil, err }
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var item struct {
			ID struct {
				Type  string `bson:"type"`
				Level string `bson:"level"`
			} `bson:"_id"`
			Count int `bson:"count"`
		}
		if err := cursor.Decode(&item); err != nil { continue }
		
		results = append(results, map[string]interface{}{
			"type":  item.ID.Type,
			"level": item.ID.Level, // Distribusi tingkat kompetisi
			"count": item.Count,
		})
	}
	return results, nil
}