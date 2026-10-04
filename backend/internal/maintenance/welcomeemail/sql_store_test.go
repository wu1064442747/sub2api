package welcomeemail

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLStoreListCandidatesFiltersActiveGiftedUnsentUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	start := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 3, 22, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "email"}).AddRow(int64(11), "new@example.com")
	mock.ExpectQuery(`(?s)SELECT\s+u\.id,\s+BTRIM\(u\.email\)\s+FROM users u.*u\.role = 'user'.*u\.status = 'active'.*u\.deleted_at IS NULL.*NULLIF\(BTRIM\(u\.email\), ''\) IS NOT NULL.*u\.balance > 0.*u\.created_at >= \$1.*u\.created_at < \$2.*NOT EXISTS.*d\.event = \$3.*d\.status = 'sent'.*LIMIT \$4`).
		WithArgs(start, end, EventName, 50).
		WillReturnRows(rows)

	got, err := NewSQLStore(db).ListCandidates(context.Background(), ListCandidatesParams{
		Start: start,
		End:   end,
		Limit: 50,
	})
	if err != nil {
		t.Fatalf("ListCandidates() error = %v", err)
	}
	if len(got) != 1 || got[0].UserID != 11 || got[0].Email != "new@example.com" {
		t.Fatalf("ListCandidates() = %+v, want user 11", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSQLStoreRecordSentUpsertsDeliveryAsSent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	sentAt := time.Date(2026, 10, 3, 22, 1, 0, 0, time.UTC)
	mock.ExpectExec(`(?s)INSERT INTO user_email_deliveries.*ON CONFLICT \(event, user_id\) DO UPDATE.*status = EXCLUDED\.status.*sent_at = EXCLUDED\.sent_at`).
		WithArgs(EventName, int64(11), "new@example.com", "subject", "body", "sent", "", sentAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = NewSQLStore(db).RecordSent(context.Background(), DeliveryRecord{
		UserID:  11,
		Email:   "new@example.com",
		Subject: "subject",
		Body:    "body",
		SentAt:  sentAt,
	})
	if err != nil {
		t.Fatalf("RecordSent() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSQLStoreTryLockUsesPostgresAdvisoryLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(AdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))

	locked, err := NewSQLStore(db).TryLock(context.Background())
	if err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	if !locked {
		t.Fatalf("TryLock() = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
