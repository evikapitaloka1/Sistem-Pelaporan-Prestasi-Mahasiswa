package repository

import (
    "context"
    "time"

    "sistempelaporan/database"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

// --- PostgreSQL Analytics ---

// 1. Top 5 Mahasiswa dengan Prestasi Terbanyak (Verified)
func GetTopStudentsStats() ([]map[string]interface{}, error) {
    query := `
        SELECT u.full_name, s.student_id, COUNT(ar.id) as total_achievements
        FROM achievement_references ar
        JOIN students s ON ar.student_id = s.id
        JOIN users u ON s.user_id = u.id
        WHERE ar.status = 'verified'
        GROUP BY u.full_name, s.student_id
        ORDER BY total_achievements DESC
        LIMIT 5
    `
    rows, err := database.PostgresDB.Query(query)
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

// 2. Tren Prestasi per Bulan (Berdasarkan waktu submit)
func GetMonthlyTrendStats() ([]map[string]interface{}, error) {
    query := `
        SELECT TO_CHAR(submitted_at, 'YYYY-MM') as month, COUNT(*) 
        FROM achievement_references 
        WHERE status IN ('submitted', 'verified')
        GROUP BY month 
        ORDER BY month DESC 
        LIMIT 12
    `
    rows, err := database.PostgresDB.Query(query)
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

// --- MongoDB Analytics ---

// 3. Distribusi Tipe Prestasi (FR-011 Point 1)
func GetAchievementTypeDistribution() ([]map[string]interface{}, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := database.MongoD.Collection("achievements")

    // Pipeline Aggregation MongoDB: Group by 'achievementType' & Count
    pipeline := mongo.Pipeline{
        
        // KOREKSI KRITIS: 1. Tambahkan $match untuk mengecualikan achievementType yang kosong
        {{Key: "$match", Value: bson.D{
            {Key: "achievementType", Value: bson.D{
                {Key: "$ne", Value: ""}, // achievementType TIDAK SAMA DENGAN string kosong
            }},
        }}},
        
        // 2. Grouping (hanya data yang valid)
        {{Key: "$group", Value: bson.D{
            {Key: "_id", Value: "$achievementType"}, 
            {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
        }}},
        
        // 3. (Opsional) Sortir hasilnya
        {{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil { return nil, err }
    defer cursor.Close(ctx) // Pastikan cursor ditutup

    var results []map[string]interface{}
    for cursor.Next(ctx) {
        var item struct {
            ID      string `bson:"_id"`
            Count int      `bson:"count"`
        }
        if err := cursor.Decode(&item); err != nil { continue }
        
        // Karena kita sudah memfilter di $match, kita hanya perlu menggunakan item.ID
        results = append(results, map[string]interface{}{
            "type": item.ID, // item.ID sekarang pasti memiliki nilai yang valid
            "count": item.Count,
        })
    }
    
    // Jika tidak ada hasil sama sekali, kembalikan slice kosong []
    if len(results) == 0 {
        return make([]map[string]interface{}, 0), nil
    }

    return results, nil
}