package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Address struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	FormattedAddress string    `json:"formatted_address" gorm:"not null"`
	PlaceID          string    `json:"place_id" gorm:"uniqueIndex"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	CreatedAt        time.Time `json:"created_at"`
}

type AddressSearch struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	AddressID  uuid.UUID `json:"address_id" gorm:"type:uuid;not null;index"`
	QueryText  string    `json:"query_text"`
	SearchedAt time.Time `json:"searched_at" gorm:"autoCreateTime"`

	User    User    `json:"user" gorm:"foreignKey:UserID"`
	Address Address `json:"address" gorm:"foreignKey:AddressID"`
}

type SavedAddress struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_user_saved_address"`
	AddressID uuid.UUID `json:"address_id" gorm:"type:uuid;not null;uniqueIndex:idx_user_saved_address"`
	Nickname  string    `json:"nickname"`
	SavedAt   time.Time `json:"saved_at" gorm:"autoCreateTime"`

	User    User    `json:"user" gorm:"foreignKey:UserID"`
	Address Address `json:"address" gorm:"foreignKey:AddressID"`
}

func (address *Address) BeforeCreate(tx *gorm.DB) error {
	if address.ID == uuid.Nil {
		address.ID = uuid.New()
	}
	return nil
}

func (search *AddressSearch) BeforeCreate(tx *gorm.DB) error {
	if search.ID == uuid.Nil {
		search.ID = uuid.New()
	}
	return nil
}

func (saved *SavedAddress) BeforeCreate(tx *gorm.DB) error {
	if saved.ID == uuid.Nil {
		saved.ID = uuid.New()
	}
	return nil
}
