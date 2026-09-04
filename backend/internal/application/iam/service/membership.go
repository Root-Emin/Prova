// Package service, birden fazla domain'e dokunan kimlik yardımcılarını barındırır.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	iamModel "github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	tenantModel "github.com/masterfabric-go/masterfabric/internal/domain/tenant/model"
	tenantRepo "github.com/masterfabric-go/masterfabric/internal/domain/tenant/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// PersonalOwnerRole, kişisel organizasyonun sahibine verilen rolün adı.
const PersonalOwnerRole = "owner"

// personalOwnerPermissions, kişisel organizasyonda sahibin hakları.
//
// Joker "*" bilerek seçildi: kişisel organizasyonda kullanıcı tek üyedir, ve
// kendi verisi üzerinde kısıtlanması anlamsız olurdu. Kurumsal
// organizasyonlarda roller tohum verisinden gelir ve joker kullanılmaz.
var personalOwnerPermissions = []string{"*"}

// MembershipService, MembershipResolver'ı PostgreSQL depoları üzerinde uygular.
type MembershipService struct {
	orgs     tenantRepo.OrgRepository
	orgUsers iamRepo.OrgUserRepository
	roles    iamRepo.RoleRepository
	log      *slog.Logger
	now      func() time.Time
}

// NewMembershipService wires the resolver.
func NewMembershipService(
	orgs tenantRepo.OrgRepository,
	orgUsers iamRepo.OrgUserRepository,
	roles iamRepo.RoleRepository,
	log *slog.Logger,
) *MembershipService {
	return &MembershipService{orgs: orgs, orgUsers: orgUsers, roles: roles, log: log, now: time.Now}
}

// ResolveActiveOrg implements service.MembershipResolver.
func (s *MembershipService) ResolveActiveOrg(ctx context.Context, userID uuid.UUID, email string) (uuid.UUID, error) {
	memberships, err := s.orgUsers.ListByUser(ctx, userID)
	if err != nil {
		return uuid.Nil, err
	}

	// Etkin üyelik varsa en eskisi seçilir. "En eski" belirlenimci bir
	// kuraldır: aynı kullanıcı iki kez giriş yaptığında iki farklı
	// organizasyonda token almasın diye sıralama rastgele bırakılamaz.
	var chosen *iamModel.OrganizationUser
	for _, m := range memberships {
		if m.Status != iamModel.OrgUserStatusActive {
			continue
		}
		if chosen == nil || m.CreatedAt.Before(chosen.CreatedAt) {
			chosen = m
		}
	}
	if chosen != nil {
		return chosen.OrganizationID, nil
	}

	return s.provisionPersonalOrg(ctx, userID, email)
}

// provisionPersonalOrg, üyeliği olmayan kullanıcıya kendi organizasyonunu açar.
//
// Alternatif, organizasyonsuz bir token üretmekti; o token'la kullanıcı giriş
// yapmış ama hiçbir şey yapamaz durumda kalıyordu. Kişisel organizasyon, tek
// kişilik de olsa, kiracı sınırının her zaman var olmasını sağlar — ve daha
// sonra kurumsal bir organizasyona davet edildiğinde eski oturumları kendi
// sınırında kalır.
func (s *MembershipService) provisionPersonalOrg(ctx context.Context, userID uuid.UUID, email string) (uuid.UUID, error) {
	now := s.now().UTC()

	org := &tenantModel.Organization{
		Name:   personalOrgName(email),
		Slug:   personalOrgSlug(email, userID),
		Status: tenantModel.OrgStatusActive,
	}
	if err := s.orgs.Create(ctx, org); err != nil {
		return uuid.Nil, fmt.Errorf("kişisel organizasyon oluşturulamadı: %w", err)
	}

	membership := &iamModel.OrganizationUser{
		OrganizationID: org.ID,
		UserID:         userID,
		Status:         iamModel.OrgUserStatusActive,
	}
	if err := s.orgUsers.Add(ctx, membership); err != nil {
		return uuid.Nil, fmt.Errorf("kişisel organizasyon üyeliği eklenemedi: %w", err)
	}

	// Rol ataması başarısız olursa organizasyon yine de kullanılabilir olmalı:
	// kullanıcı içeri girer, yalnızca yetkileri eksiktir. Bunu ölümcül hataya
	// çevirmek, düzeltilebilir bir durumu giriş engeline dönüştürürdü.
	if err := s.grantOwnerRole(ctx, userID, org.ID, now); err != nil {
		s.log.ErrorContext(ctx, "kişisel organizasyonda sahip rolü atanamadı",
			"user_id", userID, "org_id", org.ID, "error", err)
	}

	s.log.InfoContext(ctx, "kişisel organizasyon oluşturuldu", "user_id", userID, "org_id", org.ID)
	return org.ID, nil
}

func (s *MembershipService) grantOwnerRole(ctx context.Context, userID, orgID uuid.UUID, now time.Time) error {
	role := &iamModel.Role{
		ScopeType:   iamModel.ScopeTypeOrganization,
		ScopeID:     orgID,
		Name:        PersonalOwnerRole,
		Description: "Kişisel organizasyonun sahibi",
	}
	if err := s.roles.Create(ctx, role); err != nil {
		return err
	}
	for _, permission := range personalOwnerPermissions {
		if err := s.roles.AddPermission(ctx, role.ID, permission); err != nil {
			return err
		}
	}
	return s.roles.AssignRoleToUser(ctx, &iamModel.UserRole{
		UserID:         userID,
		RoleID:         role.ID,
		OrganizationID: orgID,
		CreatedAt:      now,
	})
}

// personalOrgName, adresin yerel kısmından okunabilir bir ad üretir.
func personalOrgName(email string) string {
	local, _, found := strings.Cut(email, "@")
	if !found || local == "" {
		return "Kişisel çalışma alanı"
	}
	return local + " (kişisel)"
}

// personalOrgSlug, çakışmayan bir slug üretir.
//
// Kullanıcı kimliğinin ilk sekiz hanesi eklenir: iki farklı alan adındaki aynı
// yerel kısım ("ali@a.com", "ali@b.com") aksi hâlde aynı slug'ı isterdi ve
// ikinci kayıt benzersizlik kısıtına takılırdı.
func personalOrgSlug(email string, userID uuid.UUID) string {
	local, _, _ := strings.Cut(email, "@")
	local = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, local)
	local = strings.Trim(local, "-")
	if local == "" {
		local = "user"
	}
	return fmt.Sprintf("%s-%s", local, userID.String()[:8])
}

// IsNotFound, depo katmanının "yok" cevabını sadeleştirir.
func IsNotFound(err error) bool { return errors.Is(err, domainErr.ErrNotFound) }
