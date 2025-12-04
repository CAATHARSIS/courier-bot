package models

import "time"

type CourierLocation struct {
	Longitude float64
	Latitude  float64
}

type Courier struct {
	ID           int          `json:"id"`
	ChatID       int64        `json:"chat_id"`
	Name         string       `json:"name"`
	Phone        string       `json:"phone"`
	IsActive     bool         `json:"is_active"`
	LastUpdated  time.Time    `json:"last_seen"`
	Rating       float64      `json:"rating"`
	CreatedAt    time.Time    `json:"created_at"`
}
