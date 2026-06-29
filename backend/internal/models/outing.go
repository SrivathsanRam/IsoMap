package models

import (
	"time"

	"github.com/google/uuid"
)

type Outing struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	JoinToken     string         `json:"join_token" gorm:"uniqueIndex;not null"`
	Title         string         `json:"title" gorm:"not null"`
	CreatedByUser *uuid.UUID     `json:"created_by_user_id" gorm:"type:uuid"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Members       []OutingMember `json:"members" gorm:"foreignKey:OutingID"`
}

type OutingMember struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OutingID         uuid.UUID  `json:"outing_id" gorm:"type:uuid;not null;index"`
	UserID           *uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	DisplayName      string     `json:"display_name" gorm:"not null"`
	LocationName     string     `json:"location_name"`
	Latitude         *float64   `json:"latitude"`
	Longitude        *float64   `json:"longitude"`
	MaxTravelMinutes int        `json:"max_travel_minutes" gorm:"not null;default:15"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Outing           Outing     `json:"-" gorm:"foreignKey:OutingID"`
}
