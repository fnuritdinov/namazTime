package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, d Device, tokenHash string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO devices (id, token_hash, platform, app_version, locale, time_zone_id, country)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''))`,
		d.ID, tokenHash, d.Platform, d.AppVersion, d.Locale, d.TimeZoneID, d.Country)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

// Touch — один запрос и находит устройство, и отмечает, что оно было активно.
func (r *Repository) Touch(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		UPDATE devices SET last_seen_at = now()
		WHERE token_hash = $1
		RETURNING id`, tokenHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("touch device: %w", err)
	}
	return id, nil
}

func (r *Repository) SetPush(ctx context.Context, deviceID string, p Push) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Тот же APNs-токен у другого устройства (переустановили приложение) — убираем,
	// иначе телефон получит каждое уведомление дважды.
	if _, err := tx.Exec(ctx,
		`DELETE FROM device_push WHERE apns_token = $1 AND device_id <> $2`, p.APNsToken, deviceID); err != nil {
		return fmt.Errorf("delete duplicate apns token: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO device_push (device_id, apns_token, environment, topics)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id) DO UPDATE SET
		    apns_token  = EXCLUDED.apns_token,
		    environment = EXCLUDED.environment,
		    topics      = EXCLUDED.topics,
		    updated_at  = now()`,
		deviceID, p.APNsToken, p.Environment, p.Topics); err != nil {
		return fmt.Errorf("upsert device push: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, deviceID string) error {
	// device_push удалится сам: ON DELETE CASCADE
	if _, err := r.db.Exec(ctx, `DELETE FROM devices WHERE id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}

// DeleteInactive удаляет устройства, не заходившие с before. Подписки уйдут каскадом.
func (r *Repository) DeleteInactive(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM devices WHERE last_seen_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("delete inactive devices: %w", err)
	}
	return tag.RowsAffected(), nil
}
