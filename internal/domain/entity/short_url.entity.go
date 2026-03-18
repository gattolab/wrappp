package entity

import (
	"time"

	"gorm.io/datatypes"
)

type ShortUrl struct {
	ID             int64          `gorm:"primaryKey"`
	ShortCode      string         `gorm:"column:short_code"`
	OriginalUrl    string         `gorm:"column:original_url"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	ExpiresAt      *time.Time     `gorm:"column:expire_at"`
	LastAccessedAt *time.Time     `gorm:"column:last_accessed_at"`
	IsActive       bool           `gorm:"column:is_active"`
	ClickCount     int64          `gorm:"column:click_count"`
	PasswordHash   *string        `gorm:"column:password_hash"`
	UTM            datatypes.JSON `gorm:"column:utm;type:jsonb"`
}
