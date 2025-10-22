package interfaces

import (
	"context"

	"github.com/CAATHARSIS/courier-bot/internal/models"
)

type Order interface {
	GetByID(ctx context.Context, id int) (*models.Order, error)
	UpdateCourierID(ctx context.Context, id int, courierID int) error
	GetActiveOrdersByCourier(ctx context.Context, courierID int) ([]models.Order, error)
	UpdateStatusReceived(ctx context.Context, id int, received bool) error
}
