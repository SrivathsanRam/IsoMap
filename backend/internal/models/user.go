package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string    `json:"name" gorm:"not null"`
	Email         string    `json:"email" gorm:"uniqueIndex;not null"`
	GoogleSubject *string   `json:"-" gorm:"column:google_subject;uniqueIndex"`
	PictureURL    string    `json:"picture_url"`
	PasswordHash  string    `json:"-" gorm:"column:password_hash"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (user *User) Greet() string {
	return fmt.Sprintf("Hello, Welcome %s", user.Name)
}

func (user *User) BeforeCreate(tx *gorm.DB) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return nil
}
