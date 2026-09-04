package mongo

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Scope is a tenant-bound query builder.
//
// Repositories take a Scope instead of a raw filter so that org_id cannot be
// left out of a query by accident. Forgetting it once, in one method, is a
// cross-tenant data leak; making it impossible to express an unscoped filter is
// cheaper than remembering it everywhere.
type Scope struct {
	orgID uuid.UUID
}

// NewScope binds queries to an organisation.
func NewScope(orgID uuid.UUID) Scope {
	return Scope{orgID: orgID}
}

// OrgID returns the bound organisation.
func (s Scope) OrgID() uuid.UUID { return s.orgID }

// Filter returns the given conditions with the tenant predicate applied.
//
// org_id is written last, so a caller that passes its own "org_id" — whether by
// mistake or because the value arrived from a client — cannot widen the scope.
func (s Scope) Filter(conditions bson.M) bson.M {
	filter := bson.M{}
	for k, v := range conditions {
		filter[k] = v
	}
	filter["org_id"] = s.orgID
	return filter
}

// ByID scopes a lookup of a single document version.
func (s Scope) ByID(id uuid.UUID) bson.M {
	return s.Filter(bson.M{"_id": id})
}

// ByLineage scopes a lookup of every version of one logical record.
func (s Scope) ByLineage(lineageID uuid.UUID) bson.M {
	return s.Filter(bson.M{"lineage_id": lineageID})
}
