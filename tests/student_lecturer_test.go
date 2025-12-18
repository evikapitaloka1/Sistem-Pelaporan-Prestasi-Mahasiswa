package tests

import (
	"testing"
	"sistempelaporan/app/repository"
)

/* ============================================================
   MOCK REPOSITORY (STUDENT & LECTURER RELATION)
   ============================================================
*/

type MockPeopleRepository struct {
	// Key: StudentID, Value: AdvisorID
	adviseeMap map[string]string
	studentDB  map[string]string // StudentID -> Name
}

func NewMockPeopleRepository() *MockPeopleRepository {
	return &MockPeopleRepository{
		adviseeMap: make(map[string]string),
		studentDB:  make(map[string]string),
	}
}

// Simulasi GetLecturerAdvisees logic
func (m *MockPeopleRepository) GetAdviseesCount(lecturerID string) int {
	count := 0
	for _, advisorID := range m.adviseeMap {
		if advisorID == lecturerID {
			count++
		}
	}
	return count
}

/* ============================================================
   TEST CASES
   ============================================================
*/

func TestLecturerAdvisees_Logic(t *testing.T) {
	mock := NewMockPeopleRepository()

	// Arrange: Setup 1 Dosen dengan 2 Mahasiswa Bimbingan
	lecturerID := "dosen-001"
	mock.adviseeMap["mhs-01"] = lecturerID
	mock.adviseeMap["mhs-02"] = lecturerID
	mock.adviseeMap["mhs-03"] = "dosen-lain"

	t.Run("Count Advisees Correctly", func(t *testing.T) {
		count := mock.GetAdviseesCount(lecturerID)
		if count != 2 {
			t.Errorf("Expected 2 advisees for lecturer %s, got %d", lecturerID, count)
		}
	})
}

func TestExtractAdvisorID_Helper(t *testing.T) {
	t.Run("Extract String AdvisorID", func(t *testing.T) {
		// Simulasi data map yang dikembalikan repository.GetStudentDetail
		mockStudent := map[string]interface{}{
			"id":         "std-uuid",
			"advisor_id": "lecturer-uuid-123",
		}

		result := repository.ExtractAdvisorID(mockStudent)
		if result != "lecturer-uuid-123" {
			t.Errorf("Expected lecturer-uuid-123, got %s", result)
		}
	})

	t.Run("Handle Null AdvisorID", func(t *testing.T) {
		mockStudent := map[string]interface{}{
			"id":         "std-uuid",
			"advisor_id": nil,
		}

		result := repository.ExtractAdvisorID(mockStudent)
		if result != "" {
			t.Errorf("Expected empty string for nil advisor_id, got %s", result)
		}
	})
}