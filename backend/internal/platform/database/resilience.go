package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrPoolExhausted          = errors.New("database connection pool exhausted")
	ErrDeadlockDetected       = errors.New("database deadlock detected")
	ErrDeadlockRetryExhausted = errors.New("database deadlock retries exhausted")
)

const (
	defaultOperationTimeout = 5 * time.Second
	maxDeadlockRetries      = 3
	deadlockRetryBaseDelay  = 50 * time.Millisecond
)

// WithOperationTimeout applies a bounded DB operation timeout when callers did not set a deadline.
func WithOperationTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, defaultOperationTimeout)
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}
	if IsPoolExhausted(err) && !errors.Is(err, ErrPoolExhausted) {
		return fmt.Errorf("%w: %v", ErrPoolExhausted, err)
	}
	if IsDeadlock(err) && !errors.Is(err, ErrDeadlockDetected) {
		return fmt.Errorf("%w: %v", ErrDeadlockDetected, err)
	}
	return err
}

func IsPoolExhausted(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrPoolExhausted) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "53300", "53400", "57P03":
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection pool exhausted") {
		return true
	}
	if strings.Contains(msg, "failed to acquire") && strings.Contains(msg, "timeout") {
		return true
	}
	return false
}

func IsDeadlock(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDeadlockDetected) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40P01"
}

func DeadlockRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	return deadlockRetryBaseDelay * time.Duration(1<<(attempt-1))
}

func ShouldRetryDeadlock(err error, attempt int) bool {
	return IsDeadlock(err) && attempt < maxDeadlockRetries
}

func DeadlockRetryExhaustedError(lastErr error) error {
	if lastErr == nil {
		return ErrDeadlockRetryExhausted
	}
	return fmt.Errorf("%w: %v", ErrDeadlockRetryExhausted, lastErr)
}
