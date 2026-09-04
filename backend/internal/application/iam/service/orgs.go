package service

import (
	"context"

	"github.com/google/uuid"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
)

// OrgLookup, kullanıcının üye olduğu organizasyonları listeler.
//
// Dışa aktarma ve kalıcı silme bunu kullanır: bir kullanıcı birden fazla
// organizasyonda oturum oynamış olabilir, ve tek bir kiracıya bakan bir
// dışa aktarma eksik olurdu.
type OrgLookup struct {
	orgUsers iamRepo.OrgUserRepository
}

// NewOrgLookup wires the lookup.
func NewOrgLookup(orgUsers iamRepo.OrgUserRepository) *OrgLookup {
	return &OrgLookup{orgUsers: orgUsers}
}

// OrgsForUser returns every organisation the user belongs to.
//
// Kaldırılmış üyelikler de dâhil: kullanıcı bir organizasyondan çıkarılmış
// olsa bile orada oynadığı oturumlar onun verisidir ve dışa aktarma hakkı
// üyelikle birlikte sona ermez.
func (l *OrgLookup) OrgsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	memberships, err := l.orgUsers.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]bool, len(memberships))
	orgs := make([]uuid.UUID, 0, len(memberships))
	for _, m := range memberships {
		if m == nil || seen[m.OrganizationID] {
			continue
		}
		seen[m.OrganizationID] = true
		orgs = append(orgs, m.OrganizationID)
	}
	return orgs, nil
}
