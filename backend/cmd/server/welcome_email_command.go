package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/maintenance/welcomeemail"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type welcomeEmailCommandInput struct {
	DryRun  bool
	Since   string
	Until   string
	Limit   int
	Subject string
	Body    string
}

func buildWelcomeEmailOptions(input welcomeEmailCommandInput, now time.Time) (welcomeemail.Options, error) {
	if now.IsZero() {
		now = time.Now()
	}

	since, err := parseOptionalRFC3339Flag("new-user-welcome-since", input.Since)
	if err != nil {
		return welcomeemail.Options{}, err
	}
	until, err := parseOptionalRFC3339Flag("new-user-welcome-until", input.Until)
	if err != nil {
		return welcomeemail.Options{}, err
	}
	if !since.IsZero() && !until.IsZero() && !since.Before(until) {
		return welcomeemail.Options{}, fmt.Errorf("new-user-welcome-since must be before new-user-welcome-until")
	}

	limit := input.Limit
	if limit == 0 {
		limit = welcomeemail.DefaultLimit
	}
	if limit < 0 {
		return welcomeemail.Options{}, fmt.Errorf("new-user-welcome-limit must be non-negative")
	}

	return welcomeemail.Options{
		Now:     now,
		Since:   since,
		Until:   until,
		Limit:   limit,
		DryRun:  input.DryRun,
		Subject: input.Subject,
		Body:    input.Body,
	}, nil
}

func parseOptionalRFC3339Flag(name, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339, got %q: %w", name, value, err)
	}
	return parsed, nil
}

func runNewUserWelcomeEmailCommand(ctx context.Context, input welcomeEmailCommandInput) error {
	opts, err := buildWelcomeEmailOptions(input, time.Now())
	if err != nil {
		return err
	}

	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	entClient, sqlDB, err := repository.InitEnt(cfg)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer func() { _ = entClient.Close() }()

	settingRepo := repository.NewSettingRepository(entClient)
	emailService := service.NewEmailService(settingRepo, nil)
	runner := welcomeemail.NewRunner(welcomeemail.NewSQLStore(sqlDB), emailService)

	result, runErr := runner.Run(ctx, opts)
	log.Printf(
		"new user welcome email run: window=%s..%s candidates=%d sent=%d failed=%d dry_run=%t already_running=%t",
		result.WindowStart.Format(time.RFC3339),
		result.WindowEnd.Format(time.RFC3339),
		result.Candidates,
		result.Sent,
		result.Failed,
		result.DryRun,
		result.AlreadyRunning,
	)
	return runErr
}
