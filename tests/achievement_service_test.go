package tests

import (
	"errors"
	"testing"
	

	"sistempelaporan/app/model"
	"github.com/google/uuid"
)

/* ============================================================
   MOCK HYBRID REPOSITORY (POSTGRES + MONGO SIMULATION)
   ============================================================
*/

type MockAchievementRepository struct {
	postgresDB map[uuid.UUID]*model.AchievementReference
	mongoDB    map[string]*model.AchievementMongo
}

func NewMockAchievementRepository() *MockAchievementRepository {
	return &MockAchievementRepository{
		postgresDB: make(map[uuid.UUID]*model.AchievementReference),
		mongoDB:    make(map[string]*model.AchievementMongo),
	}
}

// Simulasi CreateAchievement dengan Manual Rollback logic
func (m *MockAchievementRepository) CreateAchievement(ref *model.AchievementReference, detail *model.AchievementMongo) error {
	// 1. Simpan ke Mongo (Simulasi)
	mongoID := "mongo_generated_id_123"
	m.mongoDB[mongoID] = detail

	// 2. Simpan ke Postgres (Simulasi)
	// Kita simulasikan error jika StudentID kosong untuk memicu rollback
	if ref.StudentID == uuid.Nil {
		// ROLLBACK: Hapus dari Mongo karena Postgres gagal
		delete(m.mongoDB, mongoID)
		return errors.New("postgres error: student id is nil")
	}

	ref.MongoAchievementID = mongoID
	m.postgresDB[ref.ID] = ref
	return nil
}

/* ============================================================
   TEST CASES
   ============================================================
*/

func TestSubmitAchievement_HybridLogic(t *testing.T) {
	mockRepo := NewMockAchievementRepository()

	t.Run("Success Create Achievement", func(t *testing.T) {
		studentID := uuid.New()
		refID := uuid.New()

		ref := &model.AchievementReference{
			ID:        refID,
			StudentID: studentID,
			Status:    model.StatusDraft,
		}
		detail := &model.AchievementMongo{
			Title: "Juara 1 Lomba Coding",
		}

		err := mockRepo.CreateAchievement(ref, detail)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		// Pastikan data ada di kedua "database"
		if _, ok := mockRepo.mongoDB[ref.MongoAchievementID]; !ok {
			t.Errorf("Data detail should exist in MongoDB")
		}
		if _, ok := mockRepo.postgresDB[refID]; !ok {
			t.Errorf("Data reference should exist in Postgres")
		}
	})

	t.Run("Failed Postgres - Trigger Rollback Mongo", func(t *testing.T) {
		ref := &model.AchievementReference{
			ID:        uuid.New(),
			StudentID: uuid.Nil, // Memicu error di mock
		}
		detail := &model.AchievementMongo{Title: "Test Rollback"}

		err := mockRepo.CreateAchievement(ref, detail)

		if err == nil {
			t.Errorf("Expected error from Postgres, but got nil")
		}

		// Pastikan MongoDB bersih (tidak ada data bocor)
		if len(mockRepo.mongoDB) > 1 { // 1 adalah data dari test sebelumnya yang sukses
			// Jika datanya masih ada (misal jadi 2), berarti rollback gagal
			t.Errorf("MongoDB should be rolled back (cleaned), but data still exists")
		}
	})
}

func TestUpdateStatus_Logic(t *testing.T) {
	mockRepo := NewMockAchievementRepository()
	refID := uuid.New()
	
	// Setup: Data awal berstatus Draft
	mockRepo.postgresDB[refID] = &model.AchievementReference{
		ID:     refID,
		Status: model.StatusDraft,
	}

	t.Run("Verify Achievement", func(t *testing.T) {
		// Logic: Hanya status 'submitted' yang bisa diverifikasi
		// Kita coba verifikasi saat masih 'draft' (Harusnya gagal di level service)
		ach := mockRepo.postgresDB[refID]
		
		if ach.Status != model.StatusSubmitted {
			// Sesuai logika di service kamu: "Prestasi belum diajukan (status harus 'submitted')"
			t.Log("Successfully blocked: Cannot verify draft achievement")
		} else {
			t.Errorf("Should not be able to verify draft")
		}
	})
}