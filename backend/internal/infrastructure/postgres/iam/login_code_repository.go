package iam

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// LoginCodeRepo implements repository.LoginCodeRepository with PostgreSQL.
type LoginCodeRepo struct {
	db *pgxpool.Pool
}

// NewLoginCodeRepo creates a new LoginCodeRepo.
func NewLoginCodeRepo(db *pgxpool.Pool) *LoginCodeRepo {
	return &LoginCodeRepo{db: db}
}

const loginCodeColumns = `id, email, code_digest, purpose, attempts, max_attempts, expires_at, consumed_at, host(request_ip), created_at`

func (r *LoginCodeRepo) Create(ctx context.Context, code *model.LoginCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO login_codes (id, email, code_digest, purpose, attempts, max_attempts, expires_at, request_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		code.ID, code.Email, code.CodeDigest, code.Purpose, code.Attempts, code.MaxAttempts,
		code.ExpiresAt, nullableIP(code.RequestIP), code.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create login code", err)
	}
	return nil
}

func (r *LoginCodeRepo) GetLatestActive(ctx context.Context, email string, purpose model.LoginCodePurpose, now time.Time) (*model.LoginCode, error) {
	// "Active" is evaluated in SQL rather than in Go so that an expired or
	// exhausted row can never be handed to the comparison step by mistake.
	row := r.db.QueryRow(ctx,
		`SELECT `+loginCodeColumns+`
		 FROM login_codes
		 WHERE email = $1 AND purpose = $2
		   AND consumed_at IS NULL
		   AND expires_at > $3
		   AND (max_attempts <= 0 OR attempts < max_attempts)
		 ORDER BY created_at DESC
		 LIMIT 1`, email, purpose, now,
	)
	return scanLoginCode(row, "failed to get active login code")
}

func (r *LoginCodeRepo) GetLatest(ctx context.Context, email string, purpose model.LoginCodePurpose) (*model.LoginCode, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+loginCodeColumns+`
		 FROM login_codes
		 WHERE email = $1 AND purpose = $2
		 ORDER BY created_at DESC
		 LIMIT 1`, email, purpose,
	)
	return scanLoginCode(row, "failed to get latest login code")
}

func (r *LoginCodeRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error) {
	// Incremented and read back in one statement: two concurrent guesses must
	// consume two units of the budget, not one.
	var attempts int
	err := r.db.QueryRow(ctx,
		`UPDATE login_codes SET attempts = attempts + 1 WHERE id = $1 RETURNING attempts`, id,
	).Scan(&attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domainErr.New(domainErr.ErrNotFound, "login code not found", nil)
		}
		return 0, domainErr.New(domainErr.ErrInternal, "failed to increment login code attempts", err)
	}
	return attempts, nil
}

func (r *LoginCodeRepo) MarkConsumed(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE login_codes SET consumed_at = $1 WHERE id = $2 AND consumed_at IS NULL`, at, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to consume login code", err)
	}
	return nil
}

func (r *LoginCodeRepo) InvalidateActive(ctx context.Context, email string, purpose model.LoginCodePurpose, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE login_codes SET consumed_at = $1
		 WHERE email = $2 AND purpose = $3 AND consumed_at IS NULL AND expires_at > $1`,
		at, email, purpose,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to invalidate login codes", err)
	}
	return nil
}

func (r *LoginCodeRepo) CountSince(ctx context.Context, email string, purpose model.LoginCodePurpose, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM login_codes WHERE email = $1 AND purpose = $2 AND created_at >= $3`,
		email, purpose, since,
	).Scan(&count)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to count login codes", err)
	}
	return count, nil
}

func (r *LoginCodeRepo) DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM login_codes WHERE expires_at < $1`, cutoff)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to delete expired login codes", err)
	}
	return tag.RowsAffected(), nil
}

// rowScanner is satisfied by pgx.Row.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanLoginCode(row rowScanner, failMsg string) (*model.LoginCode, error) {
	var (
		c  model.LoginCode
		ip *string
	)
	err := row.Scan(&c.ID, &c.Email, &c.CodeDigest, &c.Purpose, &c.Attempts, &c.MaxAttempts,
		&c.ExpiresAt, &c.ConsumedAt, &ip, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "login code not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, failMsg, err)
	}
	if ip != nil {
		c.RequestIP = *ip
	}
	return &c, nil
}

// nullableIP keeps request_ip NULL for callers with no address, since INET
// rejects the empty string.
func nullableIP(ip string) any {
	if ip == "" {
		return nil
	}
	return ip
}
