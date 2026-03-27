package trips

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var (
	ErrMemberNotFound          = errors.New("member not found")
	ErrMemberAlreadyExists     = errors.New("member already exists")
	ErrLastOwner               = errors.New("cannot remove or demote last owner")
	ErrInvitationNotFound      = errors.New("invitation not found")
	ErrInvitationNotPending    = errors.New("invitation is not pending")
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrShareLinkNotFound       = errors.New("share link not found")
	ErrShareLinkAlreadyRevoked = errors.New("share link already revoked")
	ErrShareLinkRevoked        = errors.New("share link revoked")
	ErrShareLinkExpired        = errors.New("share link expired")
)

var (
	collabPoolMu sync.RWMutex
	collabPool   *pgxpool.Pool
)

func SetCollaborationPool(pool *pgxpool.Pool) {
	collabPoolMu.Lock()
	defer collabPoolMu.Unlock()
	collabPool = pool
}

func getCollaborationPool() *pgxpool.Pool {
	collabPoolMu.RLock()
	defer collabPoolMu.RUnlock()
	return collabPool
}

func listTripMembersPostgres(ctx context.Context, tripID, roleFilter string) ([]tripMember, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return nil, errors.New("postgres collaboration store not configured")
	}

	rows, err := pool.Query(ctx, `
		SELECT tm.id::text,
		       tm.user_id::text,
		       COALESCE(u.email::text, ''),
		       COALESCE(u.display_name, ''),
		       tm.role,
		       tm.status,
		       COALESCE(tm.joined_at, tm.created_at),
		       tm.created_at
		FROM trip_memberships tm
		LEFT JOIN users u ON u.id = tm.user_id
		WHERE tm.trip_id = $1::uuid
		  AND tm.status = 'active'
		  AND ($2 = '' OR tm.role = $2)
		ORDER BY tm.created_at ASC
	`, tripID, roleFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]tripMember, 0)
	for rows.Next() {
		var m tripMember
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Email,
			&m.DisplayName,
			&m.Role,
			&m.Status,
			&m.JoinedAt,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func addTripMemberPostgres(ctx context.Context, tripID string, in addTripMemberInput) (tripMember, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return tripMember{}, errors.New("postgres collaboration store not configured")
	}

	for attempt := 1; ; attempt++ {
		item, err := func() (tripMember, error) {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return tripMember{}, err
			}
			defer rollbackTx(ctx, tx)

			userID, email, displayName, err := resolveOrCreateMemberUserTx(ctx, tx, strings.TrimSpace(in.UserID), strings.TrimSpace(in.Email), strings.TrimSpace(in.DisplayName))
			if err != nil {
				return tripMember{}, err
			}

			var existingID string
			err = tx.QueryRow(ctx, `
				SELECT id::text
				FROM trip_memberships
				WHERE trip_id = $1::uuid
				  AND user_id = $2::uuid
				  AND status = 'active'
			`, tripID, userID).Scan(&existingID)
			if err == nil {
				return tripMember{}, ErrMemberAlreadyExists
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return tripMember{}, err
			}

			now := time.Now().UTC()
			var item tripMember
			item.UserID = userID
			item.Email = email
			item.DisplayName = displayName
			err = tx.QueryRow(ctx, `
				INSERT INTO trip_memberships (trip_id, user_id, role, status, joined_at, created_at, updated_at)
				VALUES ($1::uuid, $2::uuid, $3, 'active', $4, $4, $4)
				RETURNING id::text, role, status, joined_at, created_at
			`, tripID, userID, strings.TrimSpace(in.Role), now).Scan(
				&item.ID,
				&item.Role,
				&item.Status,
				&item.JoinedAt,
				&item.CreatedAt,
			)
			if err != nil {
				return tripMember{}, err
			}

			if err := tx.Commit(ctx); err != nil {
				return tripMember{}, err
			}
			return item, nil
		}()
		if err == nil {
			return item, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return tripMember{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return tripMember{}, err
	}
}

func patchTripMemberPostgres(ctx context.Context, tripID, memberID, role string) (tripMember, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return tripMember{}, errors.New("postgres collaboration store not configured")
	}

	for attempt := 1; ; attempt++ {
		item, err := func() (tripMember, error) {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return tripMember{}, err
			}
			defer rollbackTx(ctx, tx)

			var item tripMember
			err = tx.QueryRow(ctx, `
				SELECT tm.id::text,
				       tm.user_id::text,
				       COALESCE(u.email::text, ''),
				       COALESCE(u.display_name, ''),
				       tm.role,
				       tm.status,
				       COALESCE(tm.joined_at, tm.created_at),
				       tm.created_at
				FROM trip_memberships tm
				LEFT JOIN users u ON u.id = tm.user_id
				WHERE tm.trip_id = $1::uuid
				  AND tm.id = $2::uuid
				  AND tm.status = 'active'
				FOR UPDATE
			`, tripID, memberID).Scan(
				&item.ID,
				&item.UserID,
				&item.Email,
				&item.DisplayName,
				&item.Role,
				&item.Status,
				&item.JoinedAt,
				&item.CreatedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return tripMember{}, ErrMemberNotFound
			}
			if err != nil {
				return tripMember{}, err
			}

			if item.Role == "owner" && role != "owner" {
				var ownerCount int
				if err := tx.QueryRow(ctx, `
					SELECT COUNT(1)
					FROM trip_memberships
					WHERE trip_id = $1::uuid
					  AND status = 'active'
					  AND role = 'owner'
				`, tripID).Scan(&ownerCount); err != nil {
					return tripMember{}, err
				}
				if ownerCount <= 1 {
					return tripMember{}, ErrLastOwner
				}
			}

			if _, err := tx.Exec(ctx, `
				UPDATE trip_memberships
				SET role = $3, updated_at = $4
				WHERE trip_id = $1::uuid
				  AND id = $2::uuid
			`, tripID, memberID, role, time.Now().UTC()); err != nil {
				return tripMember{}, err
			}
			item.Role = role

			if err := tx.Commit(ctx); err != nil {
				return tripMember{}, err
			}
			return item, nil
		}()
		if err == nil {
			return item, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return tripMember{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return tripMember{}, err
	}
}

func removeTripMemberPostgres(ctx context.Context, tripID, memberID string) error {
	pool := getCollaborationPool()
	if pool == nil {
		return errors.New("postgres collaboration store not configured")
	}

	for attempt := 1; ; attempt++ {
		err := func() error {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return err
			}
			defer rollbackTx(ctx, tx)

			var role string
			err = tx.QueryRow(ctx, `
				SELECT role
				FROM trip_memberships
				WHERE trip_id = $1::uuid
				  AND id = $2::uuid
				  AND status = 'active'
				FOR UPDATE
			`, tripID, memberID).Scan(&role)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMemberNotFound
			}
			if err != nil {
				return err
			}

			if role == "owner" {
				var ownerCount int
				if err := tx.QueryRow(ctx, `
					SELECT COUNT(1)
					FROM trip_memberships
					WHERE trip_id = $1::uuid
					  AND status = 'active'
					  AND role = 'owner'
				`, tripID).Scan(&ownerCount); err != nil {
					return err
				}
				if ownerCount <= 1 {
					return ErrLastOwner
				}
			}

			if _, err := tx.Exec(ctx, `
				UPDATE trip_memberships
				SET status = 'removed',
				    updated_at = $3
				WHERE trip_id = $1::uuid
				  AND id = $2::uuid
			`, tripID, memberID, time.Now().UTC()); err != nil {
				return err
			}

			return tx.Commit(ctx)
		}()
		if err == nil {
			return nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return platformdb.DeadlockRetryExhaustedError(err)
		}
		return err
	}
}

func createInvitationPostgres(ctx context.Context, tripID string, in createInvitationInput) (invitation, bool, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return invitation{}, false, errors.New("postgres collaboration store not configured")
	}

	email := strings.TrimSpace(in.Email)
	existing, found, err := findPendingInvitationByEmail(ctx, pool, tripID, email)
	if err != nil {
		return invitation{}, false, err
	}
	if found {
		return existing, false, nil
	}

	if err := ensureSystemUser(ctx, pool); err != nil {
		return invitation{}, false, err
	}

	now := time.Now().UTC()
	rawToken := uuid.NewString()
	hash := sha256.Sum256([]byte(rawToken))

	var inv invitation
	inv.Token = rawToken
	err = pool.QueryRow(ctx, `
		INSERT INTO trip_invitations (
			trip_id,
			invited_by_user_id,
			invitee_email,
			role,
			token_hash,
			status,
			expires_at,
			created_at
		) VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			$5,
			'pending',
			$6,
			$7
		)
		RETURNING id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
	`, tripID, defaultOwnerUserID, email, strings.TrimSpace(in.Role), hex.EncodeToString(hash[:]), now.Add(7*24*time.Hour), now).Scan(
		&inv.ID,
		&inv.TripID,
		&inv.InvitedBy,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if err != nil {
		return invitation{}, false, err
	}
	return inv, true, nil
}

func getInvitationByIDPostgres(ctx context.Context, invitationID string) (invitation, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return invitation{}, errors.New("postgres collaboration store not configured")
	}

	var inv invitation
	err := pool.QueryRow(ctx, `
		SELECT id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
		FROM trip_invitations
		WHERE id = $1::uuid
	`, invitationID).Scan(
		&inv.ID,
		&inv.TripID,
		&inv.InvitedBy,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return invitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return invitation{}, err
	}
	return inv, nil
}

func listInvitationsPostgres(ctx context.Context, tripID string) ([]invitation, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return nil, errors.New("postgres collaboration store not configured")
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
		FROM trip_invitations
		WHERE trip_id = $1::uuid
		ORDER BY created_at DESC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]invitation, 0)
	for rows.Next() {
		var inv invitation
		if err := rows.Scan(
			&inv.ID,
			&inv.TripID,
			&inv.InvitedBy,
			&inv.Email,
			&inv.Role,
			&inv.TokenHash,
			&inv.Status,
			&inv.ExpiresAt,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func revokeInvitationPostgres(ctx context.Context, tripID, invitationID string) (invitation, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return invitation{}, errors.New("postgres collaboration store not configured")
	}

	var inv invitation
	err := pool.QueryRow(ctx, `
		UPDATE trip_invitations
		SET status = 'revoked'
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND status = 'pending'
		RETURNING id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
	`, tripID, invitationID).Scan(
		&inv.ID,
		&inv.TripID,
		&inv.InvitedBy,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if err == nil {
		return inv, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return invitation{}, err
	}

	var status string
	err = pool.QueryRow(ctx, `
		SELECT status
		FROM trip_invitations
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
	`, tripID, invitationID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return invitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return invitation{}, err
	}
	return invitation{}, ErrInvitationNotPending
}

func acceptInvitationPostgres(ctx context.Context, tripID, invitationID string) (invitation, tripMember, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return invitation{}, tripMember{}, errors.New("postgres collaboration store not configured")
	}

	for attempt := 1; ; attempt++ {
		inv, member, err := func() (invitation, tripMember, error) {
			tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return invitation{}, tripMember{}, err
			}
			defer rollbackTx(ctx, tx)

			var inv invitation
			err = tx.QueryRow(ctx, `
				SELECT id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
				FROM trip_invitations
				WHERE trip_id = $1::uuid
				  AND id = $2::uuid
				FOR UPDATE
			`, tripID, invitationID).Scan(
				&inv.ID,
				&inv.TripID,
				&inv.InvitedBy,
				&inv.Email,
				&inv.Role,
				&inv.TokenHash,
				&inv.Status,
				&inv.ExpiresAt,
				&inv.AcceptedAt,
				&inv.CreatedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return invitation{}, tripMember{}, ErrInvitationNotFound
			}
			if err != nil {
				return invitation{}, tripMember{}, err
			}

			now := time.Now().UTC()
			if inv.Status == "expired" || inv.ExpiresAt.Before(now) {
				if inv.Status == "pending" {
					if _, err := tx.Exec(ctx, `
						UPDATE trip_invitations
						SET status = 'expired'
						WHERE id = $1::uuid
					`, invitationID); err != nil {
						return invitation{}, tripMember{}, err
					}
				}
				return invitation{}, tripMember{}, ErrInvitationExpired
			}

			if inv.Status != "pending" {
				return invitation{}, tripMember{}, ErrInvitationNotPending
			}

			if _, err := tx.Exec(ctx, `
				UPDATE trip_invitations
				SET status = 'accepted',
				    accepted_at = $2
				WHERE id = $1::uuid
			`, invitationID, now); err != nil {
				return invitation{}, tripMember{}, err
			}
			inv.Status = "accepted"
			inv.AcceptedAt = &now

			userID, email, displayName, err := resolveOrCreateMemberUserTx(ctx, tx, "", inv.Email, inv.Email)
			if err != nil {
				return invitation{}, tripMember{}, err
			}

			member, err := getActiveMembershipByUserIDTx(ctx, tx, tripID, userID)
			if err != nil && !errors.Is(err, ErrMemberNotFound) {
				return invitation{}, tripMember{}, err
			}
			if errors.Is(err, ErrMemberNotFound) {
				member.UserID = userID
				member.Email = email
				member.DisplayName = displayName
				if err := tx.QueryRow(ctx, `
					INSERT INTO trip_memberships (trip_id, user_id, role, status, joined_at, created_at, updated_at)
					VALUES ($1::uuid, $2::uuid, $3, 'active', $4, $4, $4)
					RETURNING id::text, role, status, joined_at, created_at
				`, tripID, userID, inv.Role, now).Scan(
					&member.ID,
					&member.Role,
					&member.Status,
					&member.JoinedAt,
					&member.CreatedAt,
				); err != nil {
					return invitation{}, tripMember{}, err
				}
			}

			if err := tx.Commit(ctx); err != nil {
				return invitation{}, tripMember{}, err
			}
			return inv, member, nil
		}()
		if err == nil {
			return inv, member, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return invitation{}, tripMember{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return invitation{}, tripMember{}, err
	}
}

func createShareLinkPostgres(ctx context.Context, tripID string) (shareLink, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return shareLink{}, errors.New("postgres collaboration store not configured")
	}

	rawToken := uuid.NewString()
	hash := sha256.Sum256([]byte(rawToken))
	now := time.Now().UTC()

	var sl shareLink
	sl.Token = rawToken
	err := pool.QueryRow(ctx, `
		INSERT INTO share_links (trip_id, token_hash, access_scope, created_at)
		VALUES ($1::uuid, $2, 'read_only', $3)
		RETURNING id::text, trip_id::text, token_hash, access_scope, expires_at, revoked_at, created_at
	`, tripID, hex.EncodeToString(hash[:]), now).Scan(
		&sl.ID,
		&sl.TripID,
		&sl.TokenHash,
		&sl.AccessScope,
		&sl.ExpiresAt,
		&sl.RevokedAt,
		&sl.CreatedAt,
	)
	if err != nil {
		return shareLink{}, err
	}
	return sl, nil
}

func getShareLinkByIDPostgres(ctx context.Context, linkID string) (shareLink, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return shareLink{}, errors.New("postgres collaboration store not configured")
	}

	var sl shareLink
	err := pool.QueryRow(ctx, `
		SELECT id::text, trip_id::text, token_hash, access_scope, expires_at, revoked_at, created_at
		FROM share_links
		WHERE id = $1::uuid
	`, linkID).Scan(
		&sl.ID,
		&sl.TripID,
		&sl.TokenHash,
		&sl.AccessScope,
		&sl.ExpiresAt,
		&sl.RevokedAt,
		&sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return shareLink{}, ErrShareLinkNotFound
	}
	if err != nil {
		return shareLink{}, err
	}
	return sl, nil
}

func listShareLinksPostgres(ctx context.Context, tripID string) ([]shareLink, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return nil, errors.New("postgres collaboration store not configured")
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, trip_id::text, token_hash, access_scope, expires_at, revoked_at, created_at
		FROM share_links
		WHERE trip_id = $1::uuid
		ORDER BY created_at DESC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]shareLink, 0)
	for rows.Next() {
		var sl shareLink
		if err := rows.Scan(
			&sl.ID,
			&sl.TripID,
			&sl.TokenHash,
			&sl.AccessScope,
			&sl.ExpiresAt,
			&sl.RevokedAt,
			&sl.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func revokeShareLinkPostgres(ctx context.Context, tripID, linkID string) (shareLink, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return shareLink{}, errors.New("postgres collaboration store not configured")
	}

	now := time.Now().UTC()
	var sl shareLink
	err := pool.QueryRow(ctx, `
		UPDATE share_links
		SET revoked_at = $3
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND revoked_at IS NULL
		RETURNING id::text, trip_id::text, token_hash, access_scope, expires_at, revoked_at, created_at
	`, tripID, linkID, now).Scan(
		&sl.ID,
		&sl.TripID,
		&sl.TokenHash,
		&sl.AccessScope,
		&sl.ExpiresAt,
		&sl.RevokedAt,
		&sl.CreatedAt,
	)
	if err == nil {
		return sl, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return shareLink{}, err
	}

	var revokedAt *time.Time
	err = pool.QueryRow(ctx, `
		SELECT revoked_at
		FROM share_links
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
	`, tripID, linkID).Scan(&revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return shareLink{}, ErrShareLinkNotFound
	}
	if err != nil {
		return shareLink{}, err
	}
	return shareLink{}, ErrShareLinkAlreadyRevoked
}

func getShareLinkByRawTokenPostgres(ctx context.Context, tripID, rawToken string) (shareLink, error) {
	pool := getCollaborationPool()
	if pool == nil {
		return shareLink{}, errors.New("postgres collaboration store not configured")
	}

	hash := sha256.Sum256([]byte(rawToken))
	hashHex := hex.EncodeToString(hash[:])

	var sl shareLink
	err := pool.QueryRow(ctx, `
		SELECT id::text, trip_id::text, token_hash, access_scope, expires_at, revoked_at, created_at
		FROM share_links
		WHERE trip_id = $1::uuid
		  AND token_hash = $2
		LIMIT 1
	`, tripID, hashHex).Scan(
		&sl.ID,
		&sl.TripID,
		&sl.TokenHash,
		&sl.AccessScope,
		&sl.ExpiresAt,
		&sl.RevokedAt,
		&sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return shareLink{}, ErrShareLinkNotFound
	}
	if err != nil {
		return shareLink{}, err
	}

	if sl.RevokedAt != nil {
		return shareLink{}, ErrShareLinkRevoked
	}
	if sl.ExpiresAt != nil && time.Now().After(*sl.ExpiresAt) {
		return shareLink{}, ErrShareLinkExpired
	}
	return sl, nil
}

func findPendingInvitationByEmail(ctx context.Context, pool *pgxpool.Pool, tripID, email string) (invitation, bool, error) {
	var inv invitation
	err := pool.QueryRow(ctx, `
		SELECT id::text, trip_id::text, invited_by_user_id::text, invitee_email::text, role, token_hash, status, expires_at, accepted_at, created_at
		FROM trip_invitations
		WHERE trip_id = $1::uuid
		  AND invitee_email = $2
		  AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, tripID, email).Scan(
		&inv.ID,
		&inv.TripID,
		&inv.InvitedBy,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return invitation{}, false, nil
	}
	if err != nil {
		return invitation{}, false, err
	}
	return inv, true, nil
}

func ensureSystemUser(ctx context.Context, q interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) error {
	_, err := q.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, defaultOwnerUserID, "system@time-tree.local", "System")
	return err
}

func resolveOrCreateMemberUserTx(ctx context.Context, tx pgx.Tx, userID, email, displayName string) (string, string, string, error) {
	if userID != "" {
		var existingEmail, existingName string
		err := tx.QueryRow(ctx, `
			SELECT email::text, display_name
			FROM users
			WHERE id = $1::uuid
		`, userID).Scan(&existingEmail, &existingName)
		if err == nil {
			return userID, existingEmail, existingName, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", err
		}

		if email == "" {
			email = "user-" + userID + "@time-tree.local"
		}
		if displayName == "" {
			displayName = email
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
			VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
			ON CONFLICT (id) DO NOTHING
		`, userID, email, displayName)
		if err != nil {
			return "", "", "", err
		}

		if err := tx.QueryRow(ctx, `
			SELECT email::text, display_name
			FROM users
			WHERE id = $1::uuid
		`, userID).Scan(&existingEmail, &existingName); err != nil {
			return "", "", "", err
		}
		return userID, existingEmail, existingName, nil
	}

	if email == "" {
		return "", "", "", errors.New("userId or email is required")
	}

	if displayName == "" {
		displayName = email
	}

	var resolvedUserID string
	err := tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1, $2, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (email)
		DO UPDATE SET
			display_name = CASE WHEN EXCLUDED.display_name = '' THEN users.display_name ELSE EXCLUDED.display_name END,
			updated_at = now()
		RETURNING id::text, email::text, display_name
	`, email, displayName).Scan(&resolvedUserID, &email, &displayName)
	if err != nil {
		return "", "", "", err
	}

	return resolvedUserID, email, displayName, nil
}

func getActiveMembershipByUserIDTx(ctx context.Context, tx pgx.Tx, tripID, userID string) (tripMember, error) {
	var item tripMember
	err := tx.QueryRow(ctx, `
		SELECT tm.id::text,
		       tm.user_id::text,
		       COALESCE(u.email::text, ''),
		       COALESCE(u.display_name, ''),
		       tm.role,
		       tm.status,
		       COALESCE(tm.joined_at, tm.created_at),
		       tm.created_at
		FROM trip_memberships tm
		LEFT JOIN users u ON u.id = tm.user_id
		WHERE tm.trip_id = $1::uuid
		  AND tm.user_id = $2::uuid
		  AND tm.status = 'active'
	`, tripID, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Email,
		&item.DisplayName,
		&item.Role,
		&item.Status,
		&item.JoinedAt,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return tripMember{}, ErrMemberNotFound
	}
	if err != nil {
		return tripMember{}, err
	}
	return item, nil
}
