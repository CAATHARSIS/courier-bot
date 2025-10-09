package repository

import (
	"courier-bot/internal/repository/interfaces"
	"courier-bot/internal/repository/postgres"
	"database/sql"
)

type Repository struct {
	Courier         interfaces.CourierRepository
	OrderAssignment interfaces.OrderAssignment
	Order           interfaces.Order
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Courier:         postgres.NewCourierRepository(db),
		OrderAssignment: postgres.NewOrderAssignmentRepository(db),
		Order:           postgres.NewOrderRepository(db),
	}
}
