package users

import (
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	//"github.com/google/uuid"
)

func List(db *database.Database) ([]models.User, error) {
	var users []models.User

	err := db.DB.Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}
