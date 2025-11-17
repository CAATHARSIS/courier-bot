package models

import "time"

type Courier struct {
	ID             int       `json:"id"`
	ChatID         int64     `json:"chat_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	IsActive       bool      `json:"is_active"`
	LastSeen       time.Time `json:"last_seen"`
	Rating         float64   `json:"rating"`
	CreatedAt      time.Time `json:"created_at"`
}
