package models

import "time"

type Website struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	URL             string    `gorm:"uniqueIndex;not null" json:"url"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	IntervalSeconds int       `gorm:"default:60" json:"interval_seconds"` 
	LastCheckedAt   time.Time `json:"last_checked_at"`                    
	CreatedAt       time.Time `json:"created_at"`
}

type PingResult struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	WebsiteID      uint      `gorm:"index" json:"website_id"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	CheckedAt      time.Time `json:"checked_at"`
}