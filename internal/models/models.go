package models

import "time"

type Website struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	URL       string    `gorm:"uniqueIndex;not null" json:"url"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
}

type PingResult struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	WebsiteID      uint      `gorm:"index" json:"website_id"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	CheckedAt      time.Time `json:"checked_at"`
}