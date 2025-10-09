package postgres

import (
	"context"
	"courier-bot/internal/models"
	"database/sql"
)

type orderRepostitory struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *orderRepostitory {
	return &orderRepostitory{db: db}
}

func (r *orderRepostitory) GetByID(ctx context.Context, id int) (*models.Order, error) {
	return nil, nil
}

func (r *orderRepostitory) UpdateCourierID(ctx context.Context, id int, courier *models.Courier) error {
	return nil
}
