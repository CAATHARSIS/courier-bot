package interfaces

import (
	"context"

	"github.com/CAATHARSIS/courier-bot/internal/models"
)

type Order interface {
	GetByID(ctx context.Context, id int) (*models.Order, error)
	UpdateCourierID(ctx context.Context, ID int, courierID int) error
}
