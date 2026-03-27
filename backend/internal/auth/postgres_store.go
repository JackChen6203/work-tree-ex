package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var (
	ErrRefreshSessionInvalid = errors.New("invalid or expired refresh token")
	ErrRefreshSessionReuse   = errors.New("refresh token reuse detected")
	ErrRefreshSessionExpired = errors.New("refresh token has expired")
)

var (
	poolMu sync.RWMutex
	pool   *pgxpool.Pool
)

func SetPool(p *pgxpool.Pool) {
	poolMu.Lock()
	defer poolMu.Unlock()
	pool = p
}

func getPool() *pgxpool.Pool {
	poolMu.RLock()
	defer poolMu.RUnlock()
	return pool
}

func persistRefreshSessionPostgres(ctx context.Context, user *sessionUser, familyID, refreshRaw string, ttl time.Duration) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres auth store not configured")
	}

	userID, err := ensureAuthUser(ctx, p, user)
	if err != nil {
		return err
	}

	if _, err := uuid.Parse(strings.TrimSpace(familyID)); err != nil {
		familyID = uuid.NewString()
	}

	refreshHash := sha256.Sum256([]byte(refreshRaw))
	hashHex := fmt.Sprintf("%x", refreshHash)

	_, err = p.Exec(ctx, `
		INSERT INTO sessions (
			user_id, refresh_token_hash, family_id, is_revoked, expires_at, created_at, last_used_at
		) VALUES (
			$1::uuid, $2, $3::uuid, false, $4, now(), NULL
		)
	`, userID, hashHex, familyID, time.Now().Add(ttl))
	return err
}

func rotateRefreshTokenPostgres(ctx context.Context, refreshRaw, newRefreshRaw string, ttl time.Duration) (*sessionUser, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres auth store not configured")
	}

	refreshHash := sha256.Sum256([]byte(refreshRaw))
	hashHex := fmt.Sprintf("%x", refreshHash)

	for attempt := 1; ; attempt++ {
		session, err := func() (*sessionUser, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return nil, err
			}
			defer rollbackAuthTx(ctx, tx)

			var (
				sessionID   string
				userID      string
				email       string
				displayName string
				familyID    string
				isRevoked   bool
				expiresAt   time.Time
				lastUsedAt  *time.Time
			)

			err = tx.QueryRow(ctx, `
				SELECT s.id::text,
				       s.user_id::text,
				       u.email::text,
				       u.display_name,
				       s.family_id::text,
				       s.is_revoked,
				       s.expires_at,
				       s.last_used_at
				FROM sessions s
				JOIN users u ON u.id = s.user_id
				WHERE s.refresh_token_hash = $1
				FOR UPDATE
			`, hashHex).Scan(
				&sessionID,
				&userID,
				&email,
				&displayName,
				&familyID,
				&isRevoked,
				&expiresAt,
				&lastUsedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrRefreshSessionInvalid
			}
			if err != nil {
				return nil, err
			}

			now := time.Now()
			if isRevoked {
				return nil, ErrRefreshSessionInvalid
			}
			if lastUsedAt != nil {
				if _, err := tx.Exec(ctx, `
					UPDATE sessions
					SET is_revoked = true
					WHERE family_id = $1::uuid
				`, familyID); err != nil {
					return nil, err
				}
				if err := tx.Commit(ctx); err != nil {
					return nil, err
				}
				return nil, ErrRefreshSessionReuse
			}
			if now.After(expiresAt) {
				if _, err := tx.Exec(ctx, `
					UPDATE sessions
					SET is_revoked = true
					WHERE id = $1::uuid
				`, sessionID); err != nil {
					return nil, err
				}
				if err := tx.Commit(ctx); err != nil {
					return nil, err
				}
				return nil, ErrRefreshSessionExpired
			}

			if _, err := tx.Exec(ctx, `
				UPDATE sessions
				SET last_used_at = $2
				WHERE id = $1::uuid
			`, sessionID, now); err != nil {
				return nil, err
			}

			newHash := sha256.Sum256([]byte(newRefreshRaw))
			newHashHex := fmt.Sprintf("%x", newHash)
			if _, err := tx.Exec(ctx, `
				INSERT INTO sessions (
					user_id, refresh_token_hash, family_id, is_revoked, expires_at, created_at, last_used_at
				) VALUES (
					$1::uuid, $2, $3::uuid, false, $4, now(), NULL
				)
			`, userID, newHashHex, familyID, now.Add(ttl)); err != nil {
				return nil, err
			}

			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}

			return &sessionUser{
				ID:     userID,
				Name:   strings.TrimSpace(displayName),
				Email:  strings.TrimSpace(email),
				Avatar: avatarFromEmail(email),
			}, nil
		}()
		if err == nil {
			return session, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return nil, platformdb.DeadlockRetryExhaustedError(err)
		}
		return nil, err
	}
}

func ensureAuthUser(ctx context.Context, p *pgxpool.Pool, user *sessionUser) (string, error) {
	if user == nil {
		return "", errors.New("user is required")
	}

	userID := strings.TrimSpace(user.ID)
	if _, err := uuid.Parse(userID); err != nil {
		emailKey := strings.ToLower(strings.TrimSpace(user.Email))
		userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(emailKey+"|"+userID)).String()
	}

	email := strings.TrimSpace(user.Email)
	if email == "" {
		email = "user-" + userID + "@time-tree.local"
	}

	displayName := strings.TrimSpace(user.Name)
	if displayName == "" {
		displayName = displayNameFromEmail(email)
	}

	_, err := p.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id)
		DO UPDATE SET
			email = EXCLUDED.email,
			display_name = EXCLUDED.display_name,
			updated_at = now()
	`, userID, email, displayName)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func rollbackAuthTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func cleanupExpiredSessionsPostgres(ctx context.Context) (int64, error) {
	p := getPool()
	if p == nil {
		return 0, errors.New("postgres auth store not configured")
	}

	res, err := p.Exec(ctx, `
		DELETE FROM sessions
		WHERE expires_at <= now()
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

func StartSessionCleanupWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	if getPool() == nil {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = cleanupExpiredSessionsPostgres(ctx)
			}
		}
	}()
}
