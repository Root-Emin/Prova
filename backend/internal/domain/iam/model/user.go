package model

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	// UserStatusPendingDeletion, silme talebi verilmiş ama geri alma
	// penceresi dolmamış hesap. Hesap bu durumda çalışmaya devam eder:
	// kullanıcı fikrini değiştirebilmeli.
	UserStatusPendingDeletion UserStatus = "pending_deletion"
	// UserStatusDeleted, kalıcı silme işi çalışmış hesap. Satır durur ama
	// kişisel veri taşımaz; denetim kaydının bağlanacağı bir kimlik gerekir.
	UserStatusDeleted UserStatus = "deleted"
)

// User represents a platform user entity.
//
// PasswordHash is optional. Production accounts may continue to use the
// passwordless mailbox flow; seeded development accounts also support the
// explicit password login used by local test environments.
type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Status          UserStatus `json:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`

	// DeletionRequestedAt, silme talebinin verildiği an.
	DeletionRequestedAt *time.Time `json:"deletion_requested_at,omitempty"`
	// DeletionScheduledAt, geri alma penceresinin bittiği ve kalıcı silmenin
	// çalışabileceği an.
	DeletionScheduledAt *time.Time `json:"deletion_scheduled_at,omitempty"`
	// DeletedAt, kalıcı silmenin çalıştığı an.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// LockedUntil, kaba kuvvet koruması nedeniyle kilidin bittiği an.
	LockedUntil *time.Time `json:"-"`
	// FailedAttempts, ardışık başarısız giriş sayacı. Başarılı girişte sıfırlanır.
	FailedAttempts int `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsLocked reports whether brute-force protection is currently holding the
// account shut.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// IsPendingDeletion reports whether the account is inside its undo window.
func (u *User) IsPendingDeletion() bool {
	return u.Status == UserStatusPendingDeletion
}

// IsDeleted reports whether the purge job has already run.
func (u *User) IsDeleted() bool {
	return u.Status == UserStatusDeleted
}

// IsEmailVerified reports whether the address behind this account has ever been
// proven reachable.
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// FullName returns the user's full name.
func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Email
	}
	return u.FirstName + " " + u.LastName
}

// IsActive checks if the user account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}
