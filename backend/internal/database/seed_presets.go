package database

import (
	"errors"

	"github.com/SrivathsanRam/IsoMap/internal/models"
	"gorm.io/gorm"
)

func (database *Database) seedCommunityPresets() error {
	for _, preset := range mockCommunityPresets() {
		var existing models.CommunityPreset
		err := database.DB.Where("name = ?", preset.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := database.DB.Create(&preset).Error; err != nil {
			return err
		}
	}
	return nil
}

func mockCommunityPresets() []models.CommunityPreset {
	return []models.CommunityPreset{
		{
			Name:      "NUS Study Circuit",
			Source:    "community",
			Upvotes:   42,
			Downvotes: 3,
			Locations: []models.CommunityPresetLocation{
				{Name: "NUS Central Library", Address: "12 Kent Ridge Crescent, Singapore", Latitude: 1.2966, Longitude: 103.7738},
				{Name: "Education Resource Centre", Address: "8 College Avenue West, Singapore", Latitude: 1.3067, Longitude: 103.7732},
				{Name: "University Town Starbucks", Address: "2 College Avenue West, Singapore", Latitude: 1.3049, Longitude: 103.7739},
			},
		},
		{
			Name:      "Weekend Food Trail",
			Source:    "community",
			Upvotes:   31,
			Downvotes: 5,
			Locations: []models.CommunityPresetLocation{
				{Name: "Lau Pa Sat", Address: "18 Raffles Quay, Singapore", Latitude: 1.2807, Longitude: 103.8504},
				{Name: "Tiong Bahru Market", Address: "30 Seng Poh Road, Singapore", Latitude: 1.2852, Longitude: 103.8321},
				{Name: "Old Airport Road Food Centre", Address: "51 Old Airport Road, Singapore", Latitude: 1.3082, Longitude: 103.8858},
			},
		},
		{
			Name:      "Rainy Day Mall Hop",
			Source:    "community",
			Upvotes:   24,
			Downvotes: 2,
			Locations: []models.CommunityPresetLocation{
				{Name: "VivoCity", Address: "1 HarbourFront Walk, Singapore", Latitude: 1.2644, Longitude: 103.8223},
				{Name: "Suntec City", Address: "3 Temasek Boulevard, Singapore", Latitude: 1.2947, Longitude: 103.8583},
				{Name: "Jewel Changi Airport", Address: "78 Airport Boulevard, Singapore", Latitude: 1.3602, Longitude: 103.9895},
			},
		},
		{
			Name:      "Quiet Green Escapes",
			Source:    "community",
			Upvotes:   18,
			Downvotes: 1,
			Locations: []models.CommunityPresetLocation{
				{Name: "Singapore Botanic Gardens", Address: "1 Cluny Road, Singapore", Latitude: 1.3138, Longitude: 103.8159},
				{Name: "HortPark", Address: "33 Hyderabad Road, Singapore", Latitude: 1.2787, Longitude: 103.8006},
				{Name: "MacRitchie Reservoir Park", Address: "MacRitchie Reservoir Park, Singapore", Latitude: 1.3448, Longitude: 103.8318},
			},
		},
		{
			Name:      "Heritage Walk Starter Pack",
			Source:    "community",
			Upvotes:   15,
			Downvotes: 4,
			Locations: []models.CommunityPresetLocation{
				{Name: "National Museum of Singapore", Address: "93 Stamford Road, Singapore", Latitude: 1.2966, Longitude: 103.8485},
				{Name: "Sultan Mosque", Address: "3 Muscat Street, Singapore", Latitude: 1.3023, Longitude: 103.859},
				{Name: "Thian Hock Keng Temple", Address: "158 Telok Ayer Street, Singapore", Latitude: 1.2809, Longitude: 103.8476},
			},
		},
	}
}
