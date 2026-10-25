package sms

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store — журнал SMS в таблице sms_log.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

func (s *Store) count(ctx context.Context, q string, args ...any) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

func (s *Store) CountByDevice(ctx context.Context, deviceID string, since time.Time) (int, error) {
	return s.count(ctx, `SELECT count(*) FROM sms_log WHERE device_id = $1 AND created_at >= $2`, deviceID, since)
}

func (s *Store) CountByPhone(ctx context.Context, phoneHash string, since time.Time) (int, error) {
	return s.count(ctx, `SELECT count(*) FROM sms_log WHERE phone_hash = $1 AND created_at >= $2`, phoneHash, since)
}

func (s *Store) CountAll(ctx context.Context, since time.Time) (int, error) {
	return s.count(ctx, `SELECT count(*) FROM sms_log WHERE created_at >= $1`, since)
}

func (s *Store) Log(ctx context.Context, deviceID, phoneHash, providerID string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO sms_log (device_id, phone_hash, provider_id) VALUES ($1, $2, NULLIF($3, ''))`,
		deviceID, phoneHash, providerID)
	return err
}
