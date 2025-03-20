package models

import (
	"time"
)

type Model struct {
	ID        string    `gorm:"column:id;primaryKey;type:char(36)" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
