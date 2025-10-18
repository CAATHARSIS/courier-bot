package repository

import (
	"database/sql"

	"github.com/CAATHARSIS/courier-bot/internal/repository/interfaces"
	"github.com/CAATHARSIS/courier-bot/internal/repository/postgres"
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
