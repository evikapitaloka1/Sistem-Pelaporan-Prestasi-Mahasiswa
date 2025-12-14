package model

// WebResponse adalah format standar untuk semua balasan API
type WebResponse struct {
	Code   int         `json:"code"`             // HTTP Status Code (200, 400, 500)
	Status string      `json:"status"`           // "OK", "Bad Request", etc
	Data   interface{} `json:"data,omitempty"`   // Data utama (bisa object, array, atau null)
	Meta   *Meta       `json:"meta,omitempty"`   // Metadata pagination (opsional)
}

// Meta khusus untuk Pagination
type Meta struct {
	Page      int   `json:"page"`       // Halaman saat ini
	Limit     int   `json:"limit"`      // Batas data per halaman
	TotalData int64 `json:"total_data"` // Total semua data di database
	TotalPage int   `json:"total_page"` // Total halaman yang tersedia
}

// Struct khusus untuk Query Parameter Filter (FR-010)
// Gunakan ini di layer Handler/Service untuk menampung input dari URL
// Contoh: ?page=1&limit=10&status=verified
type AchievementFilter struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Status string `json:"status"` // Filter by status (draft, submitted, etc)
	Search string `json:"search"` // Search by title/student name
}