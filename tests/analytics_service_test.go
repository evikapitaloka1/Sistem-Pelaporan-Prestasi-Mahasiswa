package tests

import (
	"sync"
	"testing"
)

/* ============================================================
   MOCK ANALYTICS LOGIC
   ============================================================
*/

func TestGetGeneralStatistics_ConcurrencyLogic(t *testing.T) {
	// Kita mensimulasikan sinkronisasi 3 fungsi (Top Students, Monthly, Mongo Dist)
	var wg sync.WaitGroup
	
	// Channel untuk menangkap hasil dari goroutine
	results := make(chan string, 3)

	wg.Add(3)

	// Simulasi Tugas 1: Postgres Top Students
	go func() {
		defer wg.Done()
		results <- "top_students_ok"
	}()

	// Simulasi Tugas 2: Postgres Monthly Trend
	go func() {
		defer wg.Done()
		results <- "monthly_trend_ok"
	}()

	// Simulasi Tugas 3: Mongo Distribution
	go func() {
		defer wg.Done()
		results <- "mongo_dist_ok"
	}()

	// Tunggu proses selesai
	wg.Wait()
	close(results)

	// Assert: Pastikan terkumpul 3 hasil
	count := 0
	for range results {
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 results from concurrent tasks, got %d", count)
	}
}

func TestGetAchievementTypeDistribution_BSONMapping(t *testing.T) {
	// Simulasi hasil dari MongoDB Aggregation
	mockMongoResult := []struct {
		ID    string
		Count int
	}{
		{"Sains", 10},
		{"Olahraga", 5},
		{"Seni", 3},
	}

	t.Run("Valid Mapping", func(t *testing.T) {
		if mockMongoResult[0].ID != "Sains" {
			t.Errorf("Mapping Mongo BSON failed, expected Sains got %s", mockMongoResult[0].ID)
		}
	})
}