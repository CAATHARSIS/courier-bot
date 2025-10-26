package interfaces

import (
	"context"

	"github.com/CAATHARSIS/courier-bot/internal/models"
)

type CourierRepository interface {
	GetByID(ctx context.Context, id int) (*models.Courier, error)
	Create(ctx context.Context, courier *models.Courier) error
	Update(ctx context.Context, couier *models.Courier) (*models.Courier, error)
	DeleteByID(ctx context.Context, id int) error
	List(ctx context.Context) ([]*models.Courier, error)
	GetActiveCouriers(ctx context.Context) ([]*models.Courier, error)
	GetByChatID(ctx context.Context, chatID int64) (*models.Courier, error)
	CheckCourierByChatID(ctx context.Context, chatID int64) bool
}
