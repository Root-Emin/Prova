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

// userColumns, her okuma sorgusunun kolon listesi. Tek yerde tutuluyor:
// kolon eklemek dört ayrı SELECT'i aynı anda değiştirmek zorunda kalmasın.
const userColumns = `id, email, first_name, last_name, status, email_verified_at, deletion_requested_at, deletion_scheduled_at, deleted_at, locked_until, failed_attempts, created_at, updated_at`

// scanner, pgx.Row ve pgx.Rows'un ortak yüzü.
type scanner interface{ Scan(dest ...any) error }

// scanUser, satırı modele çevirir.
//
// E-posta ve isim alanları nullable okunur: kalıcı silme onları null'a çeker
// ama satırı bırakır, çünkü denetim kaydı silinmez ve bir kullanıcı kimliğine
// bağlanabilmelidir. Doğrudan string'e okumak, silinmiş bir hesabı listeleyen
// her sorguyu hataya çevirirdi.
func scanUser(row scanner) (*model.User, error) {
	var (
		u                          model.User
		email, firstName, lastName *string
	)
	if err := row.Scan(&u.ID, &email, &firstName, &lastName, &u.Status, &u.EmailVerifiedAt,
		&u.DeletionRequestedAt, &u.DeletionScheduledAt, &u.DeletedAt,
		&u.LockedUntil, &u.FailedAttempts, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.Email = deref(email)
	u.FirstName = deref(firstName)
	u.LastName = deref(lastName)
	return &u, nil
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// nullIfEmpty, boş metni NULL olarak yazar. Silinen hesapta e-postanın
// gerçekten null olması gerekir; boş metin "silindi" demek değildir.
func nullIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// UserRepo implements repository.UserRepository with PostgreSQL.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, first_name, last_name, status, email_verified_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, nullIfEmpty(user.Email), nullIfEmpty(user.FirstName), nullIfEmpty(user.LastName),
		user.Status, user.EmailVerifiedAt, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create user", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get user", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get user by email", err)
	}
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE users SET email=$1, first_name=$2, last_name=$3, status=$4, email_verified_at=$5,
		        deletion_requested_at=$6, deletion_scheduled_at=$7, deleted_at=$8,
		        locked_until=$9, failed_attempts=$10, updated_at=$11
		  WHERE id=$12`,
		nullIfEmpty(user.Email), nullIfEmpty(user.FirstName), nullIfEmpty(user.LastName),
		user.Status, user.EmailVerifiedAt,
		user.DeletionRequestedAt, user.DeletionScheduledAt, user.DeletedAt,
		user.LockedUntil, user.FailedAttempts, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update user", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete user", err)
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]*model.User, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count users", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list users", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to scan user", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list users", err)
	}
	return users, total, nil
}

// ListDuePurge, geri alma penceresi dolmuş hesapları döndürür.
func (r *UserRepo) ListDuePurge(ctx context.Context, now time.Time, limit int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+`
		   FROM users
		  WHERE deletion_scheduled_at IS NOT NULL
		    AND deletion_scheduled_at <= $1
		    AND deleted_at IS NULL
		  ORDER BY deletion_scheduled_at
		  LIMIT $2`, now, limit,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "silinecek hesaplar okunamadı", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "hesap çözümlenemedi", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "silinecek hesaplar okunamadı", err)
	}
	return users, nil
}

// Purge, kişisel veriyi siler ve satırı kimliksiz bir kabuk olarak bırakır.
//
// deleted_at koşulu sorgunun içinde: iş iki kez çalışırsa ikinci koşu hiçbir
// şey yapmamalı, ve "önce oku sonra sil" biçiminde yazılsaydı aynı hesap iki
// kez silinmiş sayılıp iki denetim kaydı üretirdi.
func (r *UserRepo) Purge(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users
		    SET email = NULL,
		        first_name = NULL,
		        last_name = NULL,
		        email_verified_at = NULL,
		        status = $1,
		        deleted_at = $2,
		        locked_until = NULL,
		        failed_attempts = 0,
		        updated_at = $2
		  WHERE id = $3 AND deleted_at IS NULL`,
		model.UserStatusDeleted, at, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "hesap silinemedi", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "silinecek hesap bulunamadı", nil)
	}
	return nil
}

// RecordFailedAttempt, sayacı artırır ve eşik aşılırsa kilitler.
//
// Artırma ve kilitleme tek ifadede: iki ayrı sorgu olsaydı, eşzamanlı
// denemeler sayacı aynı değerde okuyup kilidi hiç kurmayabilirdi.
func (r *UserRepo) RecordFailedAttempt(ctx context.Context, id uuid.UUID, threshold int, lockUntil time.Time) (int, error) {
	var attempts int
	err := r.db.QueryRow(ctx,
		`UPDATE users
		    SET failed_attempts = failed_attempts + 1,
		        locked_until = CASE
		            WHEN failed_attempts + 1 >= $1 THEN $2
		            ELSE locked_until
		        END,
		        updated_at = NOW()
		  WHERE id = $3
		  RETURNING failed_attempts`,
		threshold, lockUntil, id,
	).Scan(&attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domainErr.New(domainErr.ErrNotFound, "kullanıcı bulunamadı", nil)
		}
		return 0, domainErr.New(domainErr.ErrInternal, "başarısız deneme kaydedilemedi", err)
	}
	return attempts, nil
}

// ClearFailedAttempts, başarılı girişten sonra sayacı ve kilidi sıfırlar.
func (r *UserRepo) ClearFailedAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET failed_attempts = 0, locked_until = NULL, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "deneme sayacı sıfırlanamadı", err)
	}
	return nil
}
