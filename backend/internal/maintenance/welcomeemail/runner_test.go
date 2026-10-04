package welcomeemail

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunSendsWelcomeToEligibleUsersAndRecordsDelivery(t *testing.T) {
	now := time.Date(2026, 10, 3, 22, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	store := &recordingStore{
		lockAcquired: true,
		candidates: []Candidate{
			{UserID: 11, Email: "new-a@example.com"},
			{UserID: 12, Email: "new-b@example.com"},
		},
	}
	sender := &recordingSender{}

	result, err := NewRunner(store, sender).Run(context.Background(), Options{Now: now})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantStart := now.Add(-DefaultLookback)
	if !store.listParams.Start.Equal(wantStart) || !store.listParams.End.Equal(now) {
		t.Fatalf("list window = %s..%s, want %s..%s", store.listParams.Start, store.listParams.End, wantStart, now)
	}
	if len(sender.messages) != 2 {
		t.Fatalf("sent messages = %d, want 2", len(sender.messages))
	}
	if sender.messages[0].To != "new-a@example.com" || sender.messages[0].Subject != DefaultSubject || sender.messages[0].Body != DefaultBody {
		t.Fatalf("first message = %+v, want default welcome email", sender.messages[0])
	}
	if got := sentUserIDs(store.sent); got != "11,12" {
		t.Fatalf("recorded sent user ids = %s, want 11,12", got)
	}
	if result.Candidates != 2 || result.Sent != 2 || result.Failed != 0 || result.DryRun {
		t.Fatalf("result = %+v, want 2 candidates and 2 sent", result)
	}
}

func TestRunDryRunDoesNotSendOrRecord(t *testing.T) {
	now := time.Date(2026, 10, 3, 22, 0, 0, 0, time.UTC)
	store := &recordingStore{
		lockAcquired: true,
		candidates:   []Candidate{{UserID: 11, Email: "new-a@example.com"}},
	}
	sender := &recordingSender{}

	result, err := NewRunner(store, sender).Run(context.Background(), Options{Now: now, DryRun: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("sent messages = %d, want 0", len(sender.messages))
	}
	if len(store.sent) != 0 || len(store.failures) != 0 {
		t.Fatalf("delivery records = sent:%d failed:%d, want none", len(store.sent), len(store.failures))
	}
	if result.Candidates != 1 || result.Sent != 0 || !result.DryRun {
		t.Fatalf("result = %+v, want one dry-run candidate and no sends", result)
	}
}

func TestRunRecordsFailuresAndReturnsError(t *testing.T) {
	now := time.Date(2026, 10, 3, 22, 0, 0, 0, time.UTC)
	store := &recordingStore{
		lockAcquired: true,
		candidates: []Candidate{
			{UserID: 11, Email: "bad@example.com"},
			{UserID: 12, Email: "ok@example.com"},
		},
	}
	sender := &recordingSender{failures: map[string]error{"bad@example.com": errors.New("smtp timeout")}}

	result, err := NewRunner(store, sender).Run(context.Background(), Options{Now: now})
	if err == nil || !strings.Contains(err.Error(), "1 recipient") {
		t.Fatalf("Run() error = %v, want recipient failure", err)
	}
	if got := sentUserIDs(store.sent); got != "12" {
		t.Fatalf("recorded sent user ids = %s, want 12", got)
	}
	if len(store.failures) != 1 || store.failures[0].UserID != 11 || !strings.Contains(store.failures[0].Error, "smtp timeout") {
		t.Fatalf("recorded failures = %+v, want smtp timeout for user 11", store.failures)
	}
	if result.Candidates != 2 || result.Sent != 1 || result.Failed != 1 {
		t.Fatalf("result = %+v, want 2 candidates, 1 sent, 1 failed", result)
	}
}

func TestRunSkipsWhenAnotherRunHoldsLock(t *testing.T) {
	store := &recordingStore{lockAcquired: false}
	sender := &recordingSender{}

	result, err := NewRunner(store, sender).Run(context.Background(), Options{Now: time.Now()})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.AlreadyRunning {
		t.Fatalf("AlreadyRunning = false, want true")
	}
	if store.listCalled || len(sender.messages) != 0 {
		t.Fatalf("listCalled=%v sent=%d, want no work without lock", store.listCalled, len(sender.messages))
	}
}

type recordingStore struct {
	lockAcquired bool
	listCalled   bool
	listParams   ListCandidatesParams
	candidates   []Candidate
	sent         []DeliveryRecord
	failures     []DeliveryFailure
}

func (s *recordingStore) EnsureSchema(context.Context) error {
	return nil
}

func (s *recordingStore) TryLock(context.Context) (bool, error) {
	return s.lockAcquired, nil
}

func (s *recordingStore) Unlock(context.Context) error {
	return nil
}

func (s *recordingStore) ListCandidates(_ context.Context, params ListCandidatesParams) ([]Candidate, error) {
	s.listCalled = true
	s.listParams = params
	return append([]Candidate(nil), s.candidates...), nil
}

func (s *recordingStore) RecordSent(_ context.Context, record DeliveryRecord) error {
	s.sent = append(s.sent, record)
	return nil
}

func (s *recordingStore) RecordFailure(_ context.Context, failure DeliveryFailure) error {
	s.failures = append(s.failures, failure)
	return nil
}

type recordingSender struct {
	failures map[string]error
	messages []sentMessage
}

type sentMessage struct {
	To      string
	Subject string
	Body    string
}

func (s *recordingSender) SendEmail(_ context.Context, to, subject, body string) error {
	if err := s.failures[to]; err != nil {
		return err
	}
	s.messages = append(s.messages, sentMessage{To: to, Subject: subject, Body: body})
	return nil
}

func sentUserIDs(records []DeliveryRecord) string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, strconv.FormatInt(record.UserID, 10))
	}
	return strings.Join(ids, ",")
}
