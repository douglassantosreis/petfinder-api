package ad

import "time"

type Status string

const (
	StatusOpen     Status = "open"
	StatusResolved Status = "resolved"
	StatusArchived Status = "archived"
)

// GeoPoint is a GeoJSON Point. MongoDB 2dsphere uses [longitude, latitude] order.
type GeoPoint struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"`
}

func NewGeoPoint(lat, lng float64) GeoPoint {
	return GeoPoint{Type: "Point", Coordinates: []float64{lng, lat}}
}

func (g GeoPoint) Latitude() float64 {
	if len(g.Coordinates) < 2 {
		return 0
	}
	return g.Coordinates[1]
}

func (g GeoPoint) Longitude() float64 {
	if len(g.Coordinates) < 1 {
		return 0
	}
	return g.Coordinates[0]
}

type FoundAnimalReport struct {
	ID                    string    `bson:"_id" json:"id"`
	OwnerID               string    `bson:"ownerId" json:"ownerId"`
	PetType               string    `bson:"petType" json:"petType"`
	Title                 string    `bson:"title" json:"title"`
	Description           string    `bson:"description" json:"description"`
	Characteristics       []string  `bson:"characteristics" json:"characteristics"`
	LastSeenLocation      GeoPoint  `bson:"lastSeenLocation" json:"lastSeenLocation"`
	Photos                []string  `bson:"photos" json:"photos"`
	IsShelteredByReporter bool      `bson:"isShelteredByReporter" json:"isShelteredByReporter"`
	Status                Status    `bson:"status" json:"status"`
	// Visible is false while any photo is awaiting moderation.
	// Reports with no photos start as visible immediately.
	Visible               bool      `bson:"visible" json:"visible"`
	CreatedAt             time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time `bson:"updatedAt" json:"updatedAt"`
}
