package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name          string    `json:"name" gorm:"not null"`
	Email         string    `json:"email" gorm:"uniqueIndex;not null"`
	GoogleSubject string    `json:"-" gorm:"column:google_subject;uniqueIndex"`
	PictureURL    string    `json:"picture_url"`
	PasswordHash  string    `json:"-" gorm:"column:password_hash"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (user *User) Greet() string {
	return fmt.Sprintf("Hello, Welcome %s", user.Name)
}
