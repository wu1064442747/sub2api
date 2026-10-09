package welcomeemail

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	EventName      = "new_user_welcome"
	DefaultSubject = "欢迎体验 Sub2API 中转站"
	DefaultBody    = "感谢访问sub2api中转站，余额已经赠送，欢迎体验！\n\n访问网址：https://sub2api.ai-baby-dance.com/\n微信联系方式：wxl13231529281"
	DefaultLimit   = 500
)

const DefaultLookback = 24 * time.Hour

type Candidate struct {
	UserID int64
	Email  string
}

type Options struct {
	Now     time.Time
	Since   time.Time
	Until   time.Time
	Limit   int
	DryRun  bool
	Subject string
	Body    string
}

type Result struct {
	WindowStart    time.Time
	WindowEnd      time.Time
	Candidates     int
	Sent           int
	Failed         int
	DryRun         bool
	AlreadyRunning bool
}

type ListCandidatesParams struct {
	Start time.Time
	End   time.Time
	Limit int
}

type DeliveryRecord struct {
	UserID  int64
	Email   string
	Subject string
	Body    string
	SentAt  time.Time
}

type DeliveryFailure struct {
	UserID  int64
	Email   string
	Subject string
	Body    string
	Error   string
	SentAt  time.Time
}

type Store interface {
	EnsureSchema(ctx context.Context) error
	TryLock(ctx context.Context) (bool, error)
	Unlock(ctx context.Context) error
	ListCandidates(ctx context.Context, params ListCandidatesParams) ([]Candidate, error)
	RecordSent(ctx context.Context, record DeliveryRecord) error
	RecordFailure(ctx context.Context, failure DeliveryFailure) error
}

type Sender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type Runner struct {
	store  Store
	sender Sender
}

func NewRunner(store Store, sender Sender) *Runner {
	return &Runner{store: store, sender: sender}
}

func (r *Runner) Run(ctx context.Context, opts Options) (Result, error) {
	if r == nil || r.store == nil {
		return Result{}, fmt.Errorf("welcome email store is required")
	}
	if !opts.DryRun && r.sender == nil {
		return Result{}, fmt.Errorf("welcome email sender is required")
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	end := opts.Until
	if end.IsZero() {
		end = now
	}
	start := opts.Since
	if start.IsZero() {
		start = end.Add(-DefaultLookback)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	subject := strings.TrimSpace(opts.Subject)
	if subject == "" {
		subject = DefaultSubject
	}
	body := strings.TrimSpace(opts.Body)
	if body == "" {
		body = DefaultBody
	}

	result := Result{
		WindowStart: start,
		WindowEnd:   end,
		DryRun:      opts.DryRun,
	}

	if err := r.store.EnsureSchema(ctx); err != nil {
		return result, fmt.Errorf("ensure welcome email schema: %w", err)
	}
	locked, err := r.store.TryLock(ctx)
	if err != nil {
		return result, fmt.Errorf("acquire welcome email lock: %w", err)
	}
	if !locked {
		result.AlreadyRunning = true
		return result, nil
	}
	defer func() { _ = r.store.Unlock(context.Background()) }()

	candidates, err := r.store.ListCandidates(ctx, ListCandidatesParams{Start: start, End: end, Limit: limit})
	if err != nil {
		return result, fmt.Errorf("list welcome email candidates: %w", err)
	}
	result.Candidates = len(candidates)
	if opts.DryRun {
		return result, nil
	}

	for _, candidate := range candidates {
		sentAt := now
		err := r.sender.SendEmail(ctx, candidate.Email, subject, body)
		if err != nil {
			result.Failed++
			failure := DeliveryFailure{
				UserID:  candidate.UserID,
				Email:   candidate.Email,
				Subject: subject,
				Body:    body,
				Error:   err.Error(),
				SentAt:  sentAt,
			}
			if recordErr := r.store.RecordFailure(ctx, failure); recordErr != nil {
				return result, fmt.Errorf("record welcome email failure for user %d: %w", candidate.UserID, recordErr)
			}
			continue
		}

		record := DeliveryRecord{
			UserID:  candidate.UserID,
			Email:   candidate.Email,
			Subject: subject,
			Body:    body,
			SentAt:  sentAt,
		}
		if err := r.store.RecordSent(ctx, record); err != nil {
			return result, fmt.Errorf("record welcome email delivery for user %d: %w", candidate.UserID, err)
		}
		result.Sent++
	}

	if result.Failed > 0 {
		return result, fmt.Errorf("welcome email run failed for %d recipient(s)", result.Failed)
	}
	return result, nil
}
