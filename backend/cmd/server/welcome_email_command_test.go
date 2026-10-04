package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/maintenance/welcomeemail"
)

func TestBuildWelcomeEmailOptionsDefaultsToLastDayWindow(t *testing.T) {
	now := time.Date(2026, 10, 3, 22, 0, 0, 0, time.UTC)

	opts, err := buildWelcomeEmailOptions(welcomeEmailCommandInput{DryRun: true}, now)
	if err != nil {
		t.Fatalf("buildWelcomeEmailOptions() error = %v", err)
	}

	if !opts.Now.Equal(now) || !opts.DryRun {
		t.Fatalf("options = %+v, want now and dry-run preserved", opts)
	}
	if !opts.Since.IsZero() || !opts.Until.IsZero() {
		t.Fatalf("since/until = %s/%s, want runner defaults", opts.Since, opts.Until)
	}
	if opts.Limit != welcomeemail.DefaultLimit {
		t.Fatalf("limit = %d, want default %d", opts.Limit, welcomeemail.DefaultLimit)
	}
}

func TestBuildWelcomeEmailOptionsParsesRFC3339Window(t *testing.T) {
	now := time.Date(2026, 10, 3, 22, 0, 0, 0, time.UTC)
	input := welcomeEmailCommandInput{
		Since: "2026-10-02T22:00:00+08:00",
		Until: "2026-10-03T22:00:00+08:00",
		Limit: 25,
	}

	opts, err := buildWelcomeEmailOptions(input, now)
	if err != nil {
		t.Fatalf("buildWelcomeEmailOptions() error = %v", err)
	}

	wantSince := time.Date(2026, 10, 2, 14, 0, 0, 0, time.UTC)
	wantUntil := time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)
	if !opts.Since.Equal(wantSince) || !opts.Until.Equal(wantUntil) {
		t.Fatalf("window = %s/%s, want %s/%s", opts.Since, opts.Until, wantSince, wantUntil)
	}
	if opts.Limit != 25 {
		t.Fatalf("limit = %d, want 25", opts.Limit)
	}
}

func TestBuildWelcomeEmailOptionsRejectsInvalidSince(t *testing.T) {
	_, err := buildWelcomeEmailOptions(welcomeEmailCommandInput{Since: "2026-10-02 22:00:00"}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "new-user-welcome-since") {
		t.Fatalf("error = %v, want invalid since error", err)
	}
}
