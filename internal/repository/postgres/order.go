package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/CAATHARSIS/courier-bot/internal/models"
)

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *orderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) GetByID(ctx context.Context, ID int) (*models.Order, error) {
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
	err := r.db.QueryRowContext(ctx, query, ID).Scan(
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
		&order.DeliveryDate,
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

func (r *orderRepository) UpdateCourierID(ctx context.Context, id int, courierID int) error {
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

func (r *orderRepository) GetActiveOrdersByCourier(ctx context.Context, courierID int) ([]models.Order, error) {
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
			courier_id = $1
			AND is_paid = true
			AND is_assembled = true
			AND is_received = false
			AND delivery_date >= NOW() - INTERVAL '1 day'
		ORDER BY
			CASE
				WHEN delivery_date <= NOW() THEN 1
				WHEN DATE(delivery_date) = CURRENT_DATE THEN 2
				ELSE 3
			END,
			delivery_date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, courierID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active orders by courier: %v", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Surname,
			&order.Name,
			&order.PhoneNumber,
			&order.City,
			&order.City,
			&order.Address,
			&order.Flat,
			&order.Entrance,
			&order.DeliveryPrice,
			&order.FirstPrice,
			&order.FinalPrice,
			&order.PaidPrice,
			&order.BonusAccrualPercentage,
			&order.RecievedBonuses,
			&order.LostBonuses,
			&order.CreatedAt,
			&order.DeliveryDate,
			&order.RecievedAt,
			&order.IsPaid,
			&order.IsDelivery,
			&order.IsAssembled,
			&order.IsReceived,
			&order.PaymentUrl,
			&order.CourierID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return orders, nil
}

func (r *orderRepository) UpdateStatusReceived(ctx context.Context, ID int, received bool) error {
	query := `
		UPDATE orders
		SET
			is_received = $1
		WHERE
			id = $2
	`

	_, err := r.db.ExecContext(ctx, query, received, ID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %v", err)
	}

	return nil
}
