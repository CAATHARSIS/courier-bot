package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/CAATHARSIS/courier-bot/internal/models"
	"github.com/CAATHARSIS/courier-bot/internal/repository/interfaces"
)

type orderAssignmentRepository struct {
	db *sql.DB
}

func NewOrderAssignmentRepository(db *sql.DB) interfaces.OrderAssignment {
	return &orderAssignmentRepository{db: db}
}

func (r *orderAssignmentRepository) Create(ctx context.Context, orderAssignment *models.OrderAssignment) error {
	query := `
		INSERT INTO
			order_assignments (
				order_id,
				courier_id,
				assigned_at,
				expired_at,
				courier_response_status
			)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING
			id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		orderAssignment.OrderID,
		orderAssignment.CourierID,
		orderAssignment.AssignedAt,
		orderAssignment.ExpiredAt,
		orderAssignment.CourierResponseStatus,
	).Scan(&orderAssignment.ID)

	if err != nil {
		return fmt.Errorf("failed to create orderAssignment: %v", err)
	}

	return nil
}

func (r *orderAssignmentRepository) GetByID(ctx context.Context, id int) (*models.OrderAssignment, error) {
	query := `
		SELECT
			id,
			order_id,
			courier_id,
			assigned_at,
			expired_at,
			courier_response_status
		FROM
			order_assignments
		WHERE
			id = $1
	`

	var orderAssignment models.OrderAssignment

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&orderAssignment.ID,
		&orderAssignment.OrderID,
		&orderAssignment.CourierID,
		&orderAssignment.AssignedAt,
		&orderAssignment.ExpiredAt,
		&orderAssignment.CourierResponseStatus,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("order assignment not found")
		}
		return nil, fmt.Errorf("failed to get order assignment: %v", err)
	}

	return &orderAssignment, nil
}

func (r *orderAssignmentRepository) Update(ctx context.Context, orderAssignment *models.OrderAssignment) (*models.OrderAssignment, error) {
	query := `
		UPDATE order_assignments
		SET
			order_id = $1,
			courier_id = $2,
			assigned_at = $3,
			expired_at = $4,
			courier_response_status = $5
		WHERE
			id = $ 6
		RETURNING
			id,
			order_id,
			courier_id,
			assigned_at,
			expired_at,
			courier_response_status
	`

	oldOrderAssignment, err := r.GetByID(ctx, orderAssignment.ID)
	if err != nil {
		return nil, errors.New("invalid order assignment id")
	}

	if orderAssignment.OrderID == 0 {
		orderAssignment.OrderID = oldOrderAssignment.OrderID
	}

	if orderAssignment.CourierID == 0 {
		orderAssignment.CourierID = oldOrderAssignment.CourierID
	}

	if orderAssignment.AssignedAt.IsZero() {
		orderAssignment.AssignedAt = oldOrderAssignment.AssignedAt
	}

	if orderAssignment.ExpiredAt.IsZero() {
		orderAssignment.ExpiredAt = oldOrderAssignment.ExpiredAt
	}

	if orderAssignment.CourierResponseStatus == "" {
		orderAssignment.CourierResponseStatus = oldOrderAssignment.CourierResponseStatus
	}

	var updatedOrderAssignment models.OrderAssignment

	err = r.db.QueryRowContext(
		ctx,
		query,
		orderAssignment.OrderID,
		orderAssignment.CourierID,
		orderAssignment.AssignedAt,
		orderAssignment.ExpiredAt,
		orderAssignment.CourierResponseStatus,
	).Scan(
		&updatedOrderAssignment.ID,
		&updatedOrderAssignment.OrderID,
		&updatedOrderAssignment.CourierID,
		&updatedOrderAssignment.AssignedAt,
		&updatedOrderAssignment.ExpiredAt,
		&updatedOrderAssignment.CourierResponseStatus,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update order assignment: %v", err)
	}

	return &updatedOrderAssignment, nil
}

func (r *orderAssignmentRepository) DeleteByID(ctx context.Context, id int) error {
	query := `
		DELETE FROM order_assignments
		WHERE
			id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order assignment: %v", err)
	}

	return nil
}

func (r *orderAssignmentRepository) List(ctx context.Context) ([]*models.OrderAssignment, error) {
	query := `
		SELECT
			id,
			order_id,
			courier_id,
			assigned_at,
			expired_at,
			courier_response_status
		FROM
			order_assignments
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list order assignments: %v", err)
	}
	defer rows.Close()

	var orderAssignments []*models.OrderAssignment

	for rows.Next() {
		var orderAssignment models.OrderAssignment

		err := rows.Scan(
			&orderAssignment.ID,
			&orderAssignment.OrderID,
			&orderAssignment.CourierID,
			&orderAssignment.AssignedAt,
			&orderAssignment.ExpiredAt,
			&orderAssignment.CourierResponseStatus,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan order assignment: %v", err)
		}

		orderAssignments = append(orderAssignments, &orderAssignment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return orderAssignments, nil
}

func (r *orderAssignmentRepository) GetRejectedCouriers(ctx context.Context, orderID int) ([]int, error) {
	query := `
		SELECT
			counter_id
		FROM
			order-assignments
		WHERE
			order_id = $1 AND courier_response_status IN ('rejected', 'expired')
	`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected couriers: %v", err)
	}
	defer rows.Close()

	var couriersIDs []int
	for rows.Next() {
		var id int

		err := rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rejected courier id: %v", err)
		}

		couriersIDs = append(couriersIDs, id)
	}

	return couriersIDs, nil
}

func (r *orderAssignmentRepository) GetByOrderID(ctx context.Context, orderID int) (*models.OrderAssignment, error) {
	query := `
		SELECT
			id,
			order_id,
			courier_id,
			assigned_at,
			expired_at,
			courier_response_status
		FROM
			order_assignments
		WHERE
			order_id = $1
	`

	var orderAssignment models.OrderAssignment

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&orderAssignment.ID,
		&orderAssignment.OrderID,
		&orderAssignment.CourierID,
		&orderAssignment.AssignedAt,
		&orderAssignment.ExpiredAt,
		&orderAssignment.CourierResponseStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order assignment with order id (%d): %v", orderID, err)
	}

	return &orderAssignment, nil
}

func (r *orderAssignmentRepository) UpdateStatus(ctx context.Context, orderID int, newStatus models.CourierResponseStatus) error {
	query := `
		UPDATE order_assignments
		SET
			courier_response_status = $1
		WHERE
			order_id = $2 
	`

	_, err := r.db.ExecContext(ctx, query, newStatus, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order assignment (with order id %d) status: %v", orderID, err)
	}

	return nil
}
