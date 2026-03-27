package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsDeadlock(t *testing.T) {
	t.Parallel()

	err := &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
	if !IsDeadlock(err) {
		t.Fatalf("expected deadlock to be detected")
	}

	if IsDeadlock(errors.New("other")) {
		t.Fatalf("did not expect non-deadlock error to match")
	}
}

func TestIsPoolExhausted(t *testing.T) {
	t.Parallel()

	if !IsPoolExhausted(context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded to map to pool exhausted")
	}

	pgErr := &pgconn.PgError{Code: "53300", Message: "too many connections"}
	if !IsPoolExhausted(pgErr) {
		t.Fatalf("expected pg code 53300 to map to pool exhausted")
	}

	if IsPoolExhausted(errors.New("validation failed")) {
		t.Fatalf("did not expect arbitrary error to map to pool exhausted")
	}
}

func TestShouldRetryDeadlock(t *testing.T) {
	t.Parallel()

	err := &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
	if !ShouldRetryDeadlock(err, 1) {
		t.Fatalf("expected attempt 1 deadlock to be retryable")
	}
	if ShouldRetryDeadlock(err, 3) {
		t.Fatalf("did not expect attempt 3 deadlock to be retryable")
	}
}
