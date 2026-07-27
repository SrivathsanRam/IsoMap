package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityPreset struct {
	ID        uuid.UUID                 `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string                    `json:"name" gorm:"not null"`
	Source    string                    `json:"source" gorm:"-"`
	Upvotes   int                       `json:"upvotes" gorm:"not null;default:0"`
	Downvotes int                       `json:"downvotes" gorm:"not null;default:0"`
	Locations []CommunityPresetLocation `json:"locations" gorm:"foreignKey:PresetID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

type CommunityPresetLocation struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	PresetID  uuid.UUID `json:"preset_id" gorm:"type:uuid;not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Address   string    `json:"address"`
	Latitude  float64   `json:"latitude" gorm:"not null"`
	Longitude float64   `json:"longitude" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

func (preset *CommunityPreset) BeforeCreate(tx *gorm.DB) error {
	if preset.ID == uuid.Nil {
		preset.ID = uuid.New()
	}
	preset.Source = "community"
	return nil
}

func (location *CommunityPresetLocation) BeforeCreate(tx *gorm.DB) error {
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}
	return nil
}
