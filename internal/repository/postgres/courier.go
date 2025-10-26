package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/models"
	"github.com/CAATHARSIS/courier-bot/internal/repository/interfaces"
)

type courierRepository struct {
	db *sql.DB
}

func NewCourierRepository(db *sql.DB) interfaces.CourierRepository {
	return &courierRepository{db: db}
}

func (r *courierRepository) Create(ctx context.Context, courier *models.Courier) error {
	query := `
		INSERT INTO
			couriers (
				telegram_id,
				chat_id,
				name,
				phone,
				is_active,
				last_seen,
				current_order_id,
				rating,
				created_at
			)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		courier.TelegramID,
		courier.ChatID,
		courier.Name,
		courier.Phone,
		courier.IsActive,
		courier.LastSeen,
		courier.CurrentOrderID,
		courier.Rating,
		time.Now(),
	).Scan(&courier.ID)

	if err != nil {
		return fmt.Errorf("failed to create courier: %v", err)
	}
	return nil
}

func (r *courierRepository) GetByID(ctx context.Context, id int) (*models.Courier, error) {
	query := `
		SELECT
			id,
			telegram_id,
			chat_id,
			name,
			phone,
			is_active,
			last_seen,
			current_order_id,
			rating,
			created_at
		FROM
			couriers
		WHERE
			id = $1
	`
	var courier models.Courier

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&courier.ID,
		&courier.TelegramID,
		&courier.ChatID,
		&courier.Name,
		&courier.Phone,
		&courier.IsActive,
		&courier.LastSeen,
		&courier.CurrentOrderID,
		&courier.Rating,
		&courier.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("courier not found")
		}
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	return &courier, nil
}

func (r *courierRepository) Update(ctx context.Context, courier *models.Courier) (*models.Courier, error) {
	query := `
		UPDATE
			couriers
		SET
			telegram_id = $1,
			chat_id = $2,
			name = $3,
			phone = $4,
			is_active = $5,
			last_seen = $6,
			current_order = $7,
			rating = $8
		WHERE
			id = $9
		RETURNING
			id,
			telegram_id,
			chat_id,
			name,
			phone,
			is_active,
			last_seen,
			current_order,
			rating,
			created_at
	`

	oldCourier, err := r.GetByID(ctx, courier.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid courier id: %v", err)
	}

	if courier.TelegramID == 0 {
		courier.TelegramID = oldCourier.TelegramID
	}

	if courier.ChatID == 0 {
		courier.ChatID = oldCourier.ChatID
	}

	if courier.Name == "" {
		courier.Name = oldCourier.Name
	}

	if courier.Phone == "" {
		courier.Phone = oldCourier.Phone
	}

	if courier.LastSeen.IsZero() {
		courier.LastSeen = oldCourier.LastSeen
	}

	if courier.Rating == 0.0 {
		courier.Rating = oldCourier.Rating
	}

	var updatedCourier models.Courier

	err = r.db.QueryRowContext(
		ctx,
		query,
		courier.TelegramID,
		courier.ChatID,
		courier.Name,
		courier.Phone,
		courier.IsActive,
		courier.LastSeen,
		courier.CurrentOrderID,
		courier.Rating,
		courier.CreatedAt,
	).Scan(
		&updatedCourier.ID,
		&updatedCourier.TelegramID,
		&updatedCourier.ChatID,
		&updatedCourier.Name,
		&updatedCourier.Phone,
		&updatedCourier.IsActive,
		&updatedCourier.LastSeen,
		&updatedCourier.CurrentOrderID,
		&updatedCourier.Rating,
		&updatedCourier.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update courier: %v", err)
	}

	return &updatedCourier, nil
}

func (r *courierRepository) DeleteByID(ctx context.Context, id int) error {
	query := `
		DELETE FROM couriers
		WHERE
			id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete courier: %v", err)
	}

	return nil
}

func (r *courierRepository) List(ctx context.Context) ([]*models.Courier, error) {
	query := `
		SELECT
			id,
			telegram_id,
			chat_id,
			name,
			phone,
			is_active,
			last_seen,
			current_order_id,
			rating,
			created_at
		FROM
			couriers
		ORDER BY
			created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list couriers: %v", err)
	}
	defer rows.Close()

	var couriers []*models.Courier

	for rows.Next() {
		var courier models.Courier

		err := rows.Scan(
			&courier.ID,
			&courier.TelegramID,
			&courier.ChatID,
			&courier.Name,
			&courier.Phone,
			&courier.IsActive,
			&courier.LastSeen,
			&courier.CurrentOrderID,
			&courier.Rating,
			&courier.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan courier: %v", err)
		}

		couriers = append(couriers, &courier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return couriers, nil
}

func (r *courierRepository) GetActiveCouriers(ctx context.Context) ([]*models.Courier, error) {
	query := `
		SELECT
			id,
			telegram_id,
			chat_id,
			name,
			phone,
			is_active,
			last_seen,
			current_order_id,
			rating,
			created_at
		FROM
			couriers
		WHERE
			is_active = true
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active couriers: %v", err)
	}
	defer rows.Close()

	var activeCouriers []*models.Courier

	for rows.Next() {
		var activeCourier models.Courier

		err := rows.Scan(
			&activeCourier.ID,
			&activeCourier.TelegramID,
			&activeCourier.ChatID,
			&activeCourier.Name,
			&activeCourier.Phone,
			&activeCourier.IsActive,
			&activeCourier.LastSeen,
			&activeCourier.CurrentOrderID,
			&activeCourier.Rating,
			&activeCourier.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active courier: %v", err)
		}

		activeCouriers = append(activeCouriers, &activeCourier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return activeCouriers, nil
}

func (r *courierRepository) GetByChatID(ctx context.Context, chatID int64) (*models.Courier, error) {
	query := `
		SELECT
			id,
			telegram_id,
			chat_id,
			name,
			phone,
			is_active,
			last_seen,
			current_order_id,
			rating,
			created_at
		FROM
			couriers
		WHERE
			chat_id = $1
	`

	var courier models.Courier

	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&courier.ID,
		&courier.TelegramID,
		&courier.ChatID,
		&courier.Name,
		&courier.Phone,
		&courier.IsActive,
		&courier.LastSeen,
		&courier.CurrentOrderID,
		&courier.Rating,
		&courier.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier by chatID: %v", err)
	}

	return &courier, nil
}

func (r *courierRepository) CheckCourierByChatID(ctx context.Context, chatID int64) bool {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM
				couriers
			WHERE
				chat_id = $1
		) exists
	`

	var exists bool
	r.db.QueryRowContext(ctx, query, chatID).Scan(&exists)

	return exists
}
