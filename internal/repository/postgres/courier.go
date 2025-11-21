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
				chat_id,
				name,
				phone,
				is_active,
				last_updated,
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
		courier.ChatID,
		courier.Name,
		courier.Phone,
		courier.IsActive,
		courier.LastUpdated,
		courier.Rating,
		time.Now().UTC(),
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
			chat_id,
			name,
			phone,
			is_active,
			last_updated,
			rating,
			created_at,
			tracking_mode
		FROM
			couriers
		WHERE
			id = $1
	`
	var courier models.Courier

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&courier.ID,
		&courier.ChatID,
		&courier.Name,
		&courier.Phone,
		&courier.IsActive,
		&courier.LastUpdated,
		&courier.Rating,
		&courier.CreatedAt,
		&courier.TrackingMode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("courier not found")
		}
		return nil, fmt.Errorf("failed to get courier: %v", err)
	}

	return &courier, nil
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
			chat_id,
			name,
			phone,
			is_active,
			last_updated,
			rating,
			created_at,
			tracking_mode
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
			&courier.ChatID,
			&courier.Name,
			&courier.Phone,
			&courier.IsActive,
			&courier.LastUpdated,
			&courier.Rating,
			&courier.CreatedAt,
			&courier.TrackingMode,
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
			chat_id,
			name,
			phone,
			is_active,
			last_updated,
			rating,
			created_at,
			tracking_mode
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
			&activeCourier.ChatID,
			&activeCourier.Name,
			&activeCourier.Phone,
			&activeCourier.IsActive,
			&activeCourier.LastUpdated,
			&activeCourier.Rating,
			&activeCourier.CreatedAt,
			&activeCourier.TrackingMode,
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
			chat_id,
			name,
			phone,
			is_active,
			last_updated,
			rating,
			created_at,
			tracking_mode
		FROM
			couriers
		WHERE
			chat_id = $1
	`

	var courier models.Courier

	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&courier.ID,
		&courier.ChatID,
		&courier.Name,
		&courier.Phone,
		&courier.IsActive,
		&courier.LastUpdated,
		&courier.Rating,
		&courier.CreatedAt,
		&courier.TrackingMode,
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

func (r *courierRepository) UpdateCourierStatusIsActive(ctx context.Context, chatID int64, currStatus bool) error {
	query := `
		UPDATE couriers
		SET
			is_active = $1
		WHERE
			chat_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, !currStatus, chatID)
	if err != nil {
		return fmt.Errorf("Failed to update courier status (is_active): %v", err)
	}

	return nil
}

func (r courierRepository) UpdateLocation(ctx context.Context, chatID int64, location models.CourierLocation) error {
	query := `
		UPDATE couriers
		SET
			location = point($1, $2),
			last_updated = $3
		WHERE
			chat_id = $4
	`

	_, err := r.db.ExecContext(ctx, query, location.Longitude, location.Latitude, time.Now().UTC(), chatID)
	if err != nil {
		return fmt.Errorf("Failed to update current location for courier with chatID #%d: %v", chatID, err)
	}

	return nil
}

func (r courierRepository) UpdateTrackingMode(ctx context.Context, chatID int64, trackingMode bool) error {
	query := `
		UPDATE couriers
		SET
			tracking_mode = $1
		WHERE
			chat_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, trackingMode, chatID)
	if err != nil {
		return fmt.Errorf("Failed to update tracking mode for courier with chatID #%d: %v", chatID, err)
	}
	
	return nil
}

func (r *courierRepository) GetIsActiveStatus(ctx context.Context, chatID int64) (bool, error) {
	query :=  `
		SELECT
			is_active
		FROM
			couriers
		WHERE
			chat_id = $1
	`

	var isActive bool
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(&isActive)
	if err != nil {
		return false, fmt.Errorf("Failed to get active status for courier with chatID #%d: %v", chatID, err)
	}

	return isActive, nil
}
