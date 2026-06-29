package users

import (
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoogleProfile struct {
	Subject    string
	Email      string
	Name       string
	PictureURL string
}

func List(db *database.Database) ([]models.User, error) {
	var users []models.User

	err := db.DB.Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func FindByID(db *database.Database, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func FindOrCreateGoogle(db *database.Database, profile GoogleProfile) (*models.User, error) {
	var user models.User

	err := db.DB.Where("google_subject = ?", profile.Subject).First(&user).Error
	if err == nil {
		user.Name = profile.Name
		user.Email = profile.Email
		user.PictureURL = profile.PictureURL
		if err := db.DB.Save(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	err = db.DB.Where("email = ?", profile.Email).First(&user).Error
	if err == nil {
		user.GoogleSubject = profile.Subject
		user.Name = profile.Name
		user.PictureURL = profile.PictureURL
		if err := db.DB.Save(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	user = models.User{
		Name:          profile.Name,
		Email:         profile.Email,
		GoogleSubject: profile.Subject,
		PictureURL:    profile.PictureURL,
		PasswordHash:  "",
	}
	if err := db.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
