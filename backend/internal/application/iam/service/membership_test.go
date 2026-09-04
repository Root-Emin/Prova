package service

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	iamModel "github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	tenantModel "github.com/masterfabric-go/masterfabric/internal/domain/tenant/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// --- sahte depolar ---

type fakeOrgRepo struct {
	mu    sync.Mutex
	orgs  map[uuid.UUID]*tenantModel.Organization
	slugs map[string]bool
}

func newFakeOrgRepo() *fakeOrgRepo {
	return &fakeOrgRepo{orgs: map[uuid.UUID]*tenantModel.Organization{}, slugs: map[string]bool{}}
}

func (r *fakeOrgRepo) Create(_ context.Context, org *tenantModel.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.slugs[org.Slug] {
		return domainErr.New(domainErr.ErrAlreadyExists, "slug taken", nil)
	}
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	r.slugs[org.Slug] = true
	r.orgs[org.ID] = org
	return nil
}

func (r *fakeOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*tenantModel.Organization, error) {
	if o, ok := r.orgs[id]; ok {
		return o, nil
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "org not found", nil)
}
func (r *fakeOrgRepo) GetBySlug(context.Context, string) (*tenantModel.Organization, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "org not found", nil)
}
func (r *fakeOrgRepo) Update(context.Context, *tenantModel.Organization) error { return nil }
func (r *fakeOrgRepo) Delete(context.Context, uuid.UUID) error                 { return nil }
func (r *fakeOrgRepo) List(context.Context, int, int) ([]*tenantModel.Organization, int, error) {
	return nil, 0, nil
}

type fakeOrgUserRepo struct {
	rows []*iamModel.OrganizationUser
}

func (r *fakeOrgUserRepo) Add(_ context.Context, ou *iamModel.OrganizationUser) error {
	if ou.CreatedAt.IsZero() {
		ou.CreatedAt = time.Now().UTC()
	}
	r.rows = append(r.rows, ou)
	return nil
}
func (r *fakeOrgUserRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *fakeOrgUserRepo) GetByOrgAndUser(context.Context, uuid.UUID, uuid.UUID) (*iamModel.OrganizationUser, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "not found", nil)
}
func (r *fakeOrgUserRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*iamModel.OrganizationUser, int, error) {
	return nil, 0, nil
}
func (r *fakeOrgUserRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*iamModel.OrganizationUser, error) {
	var out []*iamModel.OrganizationUser
	for _, row := range r.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

type fakeRoleRepo struct {
	permissions map[uuid.UUID][]string
	assignments []*iamModel.UserRole
}

func newFakeRoleRepo() *fakeRoleRepo {
	return &fakeRoleRepo{permissions: map[uuid.UUID][]string{}}
}

func (r *fakeRoleRepo) Create(_ context.Context, role *iamModel.Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	return nil
}
func (r *fakeRoleRepo) GetByID(context.Context, uuid.UUID) (*iamModel.Role, error) { return nil, nil }
func (r *fakeRoleRepo) ListByScope(context.Context, iamModel.ScopeType, uuid.UUID) ([]*iamModel.Role, error) {
	return nil, nil
}
func (r *fakeRoleRepo) Update(context.Context, *iamModel.Role) error { return nil }
func (r *fakeRoleRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (r *fakeRoleRepo) AddPermission(_ context.Context, roleID uuid.UUID, permission string) error {
	r.permissions[roleID] = append(r.permissions[roleID], permission)
	return nil
}
func (r *fakeRoleRepo) RemovePermission(context.Context, uuid.UUID, string) error { return nil }
func (r *fakeRoleRepo) GetPermissions(_ context.Context, roleID uuid.UUID) ([]string, error) {
	return r.permissions[roleID], nil
}
func (r *fakeRoleRepo) AssignRoleToUser(_ context.Context, ur *iamModel.UserRole) error {
	r.assignments = append(r.assignments, ur)
	return nil
}
func (r *fakeRoleRepo) RemoveRoleFromUser(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *fakeRoleRepo) GetUserRoles(context.Context, uuid.UUID, uuid.UUID) ([]*iamModel.UserRole, error) {
	return nil, nil
}
func (r *fakeRoleRepo) GetUserPermissions(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return nil, nil
}

// --- testler ---

func newService() (*MembershipService, *fakeOrgRepo, *fakeOrgUserRepo, *fakeRoleRepo) {
	orgs, orgUsers, roles := newFakeOrgRepo(), &fakeOrgUserRepo{}, newFakeRoleRepo()
	return NewMembershipService(orgs, orgUsers, roles, discardLogger()), orgs, orgUsers, roles
}

// Bu, Faz 0'ın düzelttiği asıl hata: token'a sıfır UUID yazıldığında RBAC
// sıfır organizasyonda izin arıyor ve her korumalı alan kapanıyordu.
func TestResolveActiveOrg_NeverReturnsNilOrg(t *testing.T) {
	svc, _, _, _ := newService()

	orgID, err := svc.ResolveActiveOrg(context.Background(), uuid.New(), "ali@corp.com")

	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if orgID == uuid.Nil {
		t.Fatal("organizasyon kimliği sıfır UUID olamaz")
	}
}

func TestResolveActiveOrg_UsesExistingMembership(t *testing.T) {
	svc, orgs, orgUsers, _ := newService()
	userID, existing := uuid.New(), uuid.New()
	orgUsers.rows = append(orgUsers.rows, &iamModel.OrganizationUser{
		OrganizationID: existing, UserID: userID,
		Status: iamModel.OrgUserStatusActive, CreatedAt: time.Now(),
	})

	orgID, err := svc.ResolveActiveOrg(context.Background(), userID, "ali@corp.com")

	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if orgID != existing {
		t.Fatalf("mevcut üyelik kullanılmalıydı: %s != %s", orgID, existing)
	}
	if len(orgs.orgs) != 0 {
		t.Fatal("üyeliği olan kullanıcıya yeni organizasyon açılmamalı")
	}
}

// Birden fazla üyelikte seçim belirlenimci olmalı: aynı kullanıcı iki kez
// giriş yaptığında iki farklı organizasyonda token almamalı.
func TestResolveActiveOrg_PicksOldestActiveMembershipDeterministically(t *testing.T) {
	svc, _, orgUsers, _ := newService()
	userID := uuid.New()
	oldest, newer := uuid.New(), uuid.New()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	orgUsers.rows = append(orgUsers.rows,
		&iamModel.OrganizationUser{OrganizationID: newer, UserID: userID, Status: iamModel.OrgUserStatusActive, CreatedAt: base.Add(time.Hour)},
		&iamModel.OrganizationUser{OrganizationID: oldest, UserID: userID, Status: iamModel.OrgUserStatusActive, CreatedAt: base},
	)

	for i := 0; i < 5; i++ {
		orgID, err := svc.ResolveActiveOrg(context.Background(), userID, "ali@corp.com")
		if err != nil {
			t.Fatalf("beklenmeyen hata: %v", err)
		}
		if orgID != oldest {
			t.Fatalf("en eski etkin üyelik seçilmeliydi, %s geldi", orgID)
		}
	}
}

// Davet edilmiş ama kabul etmemiş üyelik bir kiracı hakkı vermez.
func TestResolveActiveOrg_IgnoresNonActiveMemberships(t *testing.T) {
	svc, orgs, orgUsers, _ := newService()
	userID, invited := uuid.New(), uuid.New()
	orgUsers.rows = append(orgUsers.rows, &iamModel.OrganizationUser{
		OrganizationID: invited, UserID: userID, Status: iamModel.OrgUserStatusInvited,
	})

	orgID, err := svc.ResolveActiveOrg(context.Background(), userID, "ali@corp.com")

	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if orgID == invited {
		t.Fatal("etkin olmayan üyelik seçilmemeli")
	}
	if len(orgs.orgs) != 1 {
		t.Fatal("kişisel organizasyon açılmalıydı")
	}
}

func TestResolveActiveOrg_GrantsOwnerRoleInPersonalOrg(t *testing.T) {
	svc, _, _, roles := newService()

	orgID, err := svc.ResolveActiveOrg(context.Background(), uuid.New(), "ali@corp.com")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}

	if len(roles.assignments) != 1 || roles.assignments[0].OrganizationID != orgID {
		t.Fatalf("kişisel organizasyonda sahip rolü atanmalıydı: %+v", roles.assignments)
	}
}

// Aynı yerel kısma sahip iki farklı adres aynı slug'ı isterse ikinci kayıt
// benzersizlik kısıtına takılır; kullanıcı kimliği eki bunu engeller.
func TestPersonalOrgSlug_IsUniquePerUser(t *testing.T) {
	svc, orgs, _, _ := newService()

	first, err := svc.ResolveActiveOrg(context.Background(), uuid.New(), "ali@a.com")
	if err != nil {
		t.Fatalf("ilk kullanıcı: %v", err)
	}
	second, err := svc.ResolveActiveOrg(context.Background(), uuid.New(), "ali@b.com")
	if err != nil {
		t.Fatalf("ikinci kullanıcı: %v", err)
	}

	if first == second {
		t.Fatal("iki kullanıcı aynı organizasyonu paylaşmamalı")
	}
	if len(orgs.orgs) != 2 {
		t.Fatalf("iki ayrı organizasyon beklenirdi, %d var", len(orgs.orgs))
	}
}
