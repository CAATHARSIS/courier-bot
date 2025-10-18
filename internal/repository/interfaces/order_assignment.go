package interfaces

import (
	"context"

	"github.com/CAATHARSIS/courier-bot/internal/models"
)

type OrderAssignment interface {
	Create(ctx context.Context, orderAssignment *models.OrderAssignment) error
	GetByID(ctx context.Context, id int) (*models.OrderAssignment, error)
	Update(ctx context.Context, orderAssignment *models.OrderAssignment) (*models.OrderAssignment, error)
	DeleteByID(ctx context.Context, id int) error
	List(ctx context.Context) ([]*models.OrderAssignment, error)
	GetRejectedCouriers(ctx context.Context, id int) ([]int, error)
	GetByOrderID(ctx context.Context, orderID int) (*models.OrderAssignment, error)
	UpdateStatus(ctx context.Context, id int, status models.CourierResponseStatus) error
}
