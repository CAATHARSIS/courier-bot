package interfaces

import (
	"context"
	"courier-bot/internal/models"
)

type OrderAssignment interface {
	Create(context.Context, *models.OrderAssignment) error
	GetByID(context.Context, int) (*models.OrderAssignment, error)
	Update(context.Context, *models.OrderAssignment) (*models.OrderAssignment, error)
	DeleteByID(context.Context, int) error
	List(context.Context) ([]*models.OrderAssignment, error)
}
