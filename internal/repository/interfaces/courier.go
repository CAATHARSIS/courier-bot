package interfaces

import (
	"context"
	"courier-bot/internal/models"
)

type CourierRepository interface {
	GetByID(context.Context, int) (*models.Courier, error)
	Create(context.Context, *models.Courier) error
	Update(context.Context, *models.Courier) (*models.Courier, error)
	DeleteByID(context.Context, int) error
	List(context.Context) ([]*models.Courier, error)
}
