package model


type WebResponse struct {
	Code   int         `json:"code"`             
	Status string      `json:"status"`           
	Data   interface{} `json:"data,omitempty"`   
	Meta   *Meta       `json:"meta,omitempty"`  
}


type Meta struct {
	Page      int   `json:"page"`       
	Limit     int   `json:"limit"`      
	TotalData int64 `json:"total_data"` 
	TotalPage int   `json:"total_page"` 
}


type AchievementFilter struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Status string `json:"status"` 
	Search string `json:"search"` 
}