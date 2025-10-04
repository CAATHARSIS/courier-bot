package models

import "time"

type Courier struct {
	ID           int       `json:"id"`
	TelegramID   int64     `json:"telegram_id"`
	ChatID       int64     `json:"chat_id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	IsActive     bool      `json:"is_active"`
	LastSeen     time.Time `json:"last_seen"`
	CurrentOrder *int      `json:"current_order"`
	Rating       float64   `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
}
