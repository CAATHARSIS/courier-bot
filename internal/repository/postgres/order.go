package postgres

import (
	"context"
	"courier-bot/internal/models"
	"database/sql"
	"errors"
	"fmt"
)

type orderRepostitory struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *orderRepostitory {
	return &orderRepostitory{db: db}
}

func (r *orderRepostitory) GetByID(ctx context.Context, id int) (*models.Order, error) {
	query := `
		SELECT
			id,
			user_id,
			surname,
			name,
			phone_number,
			city,
			address,
			flat,
			entrance,
			delivery_price,
			first_price,
			final_price,
			paid_price,
			bonus_accrual_percentage,
			received_bonuses,
			lost_bonuses,
			created_at,
			delivery_date,
			received_at,
			is_paid,
			is_delivery,
			is_assembled,
			is_received,
			payment_url,
			courier_id
		FROM
			orders
		WHERE
			id = $1
	`

	var order models.Order
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Surname,
		&order.Name,
		&order.PhoneNumber,
		&order.City,
		&order.Address,
		&order.Entrance,
		&order.DeliveryPrice,
		&order.FirstPrice,
		&order.FinalPrice,
		&order.PaidPrice,
		&order.BonusAccrualPercentage,
		&order.RecievedBonuses,
		&order.LostBonuses,
		&order.CreatedAt,
		&order.DeliverDate,
		&order.RecievedAt,
		&order.IsPaid,
		&order.IsDelivery,
		&order.IsAssembled,
		&order.IsReceived,
		&order.PaymentUrl,
		&order.CourierID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("order not found")
		}
		return nil, fmt.Errorf("failed to get order by id: %v", err)
	}

	return &order, nil
}

func (r *orderRepostitory) UpdateCourierID(ctx context.Context, id int, courierID int) error {
	query := `
		UPDATE orders
		SET
			courier_id = $1
		WHERE
			id = $2
	`

	_, err := r.db.ExecContext(ctx, query, courierID, id)
	if err != nil {
		return fmt.Errorf("failed to update courier_id field of order (id: %d) table: %v", id, err)
	}

	return nil
}
