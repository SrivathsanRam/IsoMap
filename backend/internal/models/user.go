package models

import (
	"fmt"
	"time"
	"github.com/google/uuid"


)

type User struct {
	ID   uuid.UUID `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name string    `json:"name" gorm:"not null"`
	Email string    `json:"email" gorm:"uniqueIndex"`
	PasswordHash string    `json:"-" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
}

func (user *User) Greet() string {
	return fmt.Sprintf("Hello, Welcome %s", user.Name)
}
