package welcomeemail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const AdvisoryLockID int64 = 740427338582241103

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("nil SQL store")
	}
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS user_email_deliveries (
	id BIGSERIAL PRIMARY KEY,
	event VARCHAR(100) NOT NULL,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	email VARCHAR(255) NOT NULL,
	subject TEXT NOT NULL,
	body TEXT NOT NULL,
	status VARCHAR(20) NOT NULL,
	error TEXT NOT NULL DEFAULT '',
	sent_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (event, user_id),
	CONSTRAINT user_email_deliveries_status_check CHECK (status IN ('sent', 'failed'))
);
CREATE INDEX IF NOT EXISTS idx_user_email_deliveries_event_status
	ON user_email_deliveries (event, status);
CREATE INDEX IF NOT EXISTS idx_user_email_deliveries_sent_at
	ON user_email_deliveries (sent_at);
`)
	if err != nil {
		return fmt.Errorf("create user_email_deliveries: %w", err)
	}
	return nil
}

func (s *SQLStore) TryLock(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("nil SQL store")
	}
	var locked bool
	if err := s.db.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, AdvisoryLockID).Scan(&locked); err != nil {
		return false, err
	}
	return locked, nil
}

func (s *SQLStore) Unlock(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("nil SQL store")
	}
	_, err := s.db.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, AdvisoryLockID)
	return err
}

func (s *SQLStore) ListCandidates(ctx context.Context, params ListCandidatesParams) ([]Candidate, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil SQL store")
	}
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT u.id, BTRIM(u.email)
FROM users u
WHERE u.role = 'user'
  AND u.status = 'active'
  AND u.deleted_at IS NULL
  AND NULLIF(BTRIM(u.email), '') IS NOT NULL
  AND u.balance > 0
  AND u.created_at >= $1
  AND u.created_at < $2
  AND NOT EXISTS (
		SELECT 1
		FROM user_email_deliveries d
		WHERE d.event = $3
		  AND d.user_id = u.id
		  AND d.status = 'sent'
  )
ORDER BY u.created_at ASC, u.id ASC
LIMIT $4`, params.Start, params.End, EventName, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var candidates []Candidate
	for rows.Next() {
		var candidate Candidate
		if err := rows.Scan(&candidate.UserID, &candidate.Email); err != nil {
			return nil, err
		}
		candidate.Email = strings.TrimSpace(candidate.Email)
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (s *SQLStore) RecordSent(ctx context.Context, record DeliveryRecord) error {
	return s.recordDelivery(ctx, record.UserID, record.Email, record.Subject, record.Body, "sent", "", record.SentAt)
}

func (s *SQLStore) RecordFailure(ctx context.Context, failure DeliveryFailure) error {
	return s.recordDelivery(ctx, failure.UserID, failure.Email, failure.Subject, failure.Body, "failed", truncateDeliveryError(failure.Error), failure.SentAt)
}

func (s *SQLStore) recordDelivery(ctx context.Context, userID int64, email, subject, body, status, message string, sentAt time.Time) error {
	if s == nil || s.db == nil {
		return errors.New("nil SQL store")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_email_deliveries (
	event, user_id, email, subject, body, status, error, sent_at, created_at, updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
)
ON CONFLICT (event, user_id) DO UPDATE SET
	email = EXCLUDED.email,
	subject = EXCLUDED.subject,
	body = EXCLUDED.body,
	status = EXCLUDED.status,
	error = EXCLUDED.error,
	sent_at = EXCLUDED.sent_at,
	updated_at = NOW()`,
		EventName, userID, strings.TrimSpace(email), subject, body, status, message, sentAt)
	if err != nil {
		return fmt.Errorf("upsert welcome email delivery: %w", err)
	}
	return nil
}

func truncateDeliveryError(message string) string {
	const maxDeliveryErrorLength = 4000
	message = strings.TrimSpace(message)
	if len(message) <= maxDeliveryErrorLength {
		return message
	}
	return message[:maxDeliveryErrorLength]
}
