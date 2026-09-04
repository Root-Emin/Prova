package mongo

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestScope_AlwaysAppliesTheTenantPredicate(t *testing.T) {
	orgID := uuid.New()
	scope := NewScope(orgID)

	filter := scope.Filter(bson.M{"status": "published"})

	assert.Equal(t, orgID, filter["org_id"])
	assert.Equal(t, "published", filter["status"])
}

// A caller passing its own org_id — by mistake, or because the value arrived
// from a client — must not be able to widen the scope.
func TestScope_CallerCannotOverrideTheTenant(t *testing.T) {
	orgID := uuid.New()
	otherOrg := uuid.New()

	filter := NewScope(orgID).Filter(bson.M{"org_id": otherOrg})

	assert.Equal(t, orgID, filter["org_id"])
}

func TestScope_DoesNotMutateCallerConditions(t *testing.T) {
	conditions := bson.M{"status": "draft"}

	NewScope(uuid.New()).Filter(conditions)

	assert.NotContains(t, conditions, "org_id")
}

func TestScope_ByIDAndByLineageAreScoped(t *testing.T) {
	orgID := uuid.New()
	docID := uuid.New()
	lineageID := uuid.New()
	scope := NewScope(orgID)

	byID := scope.ByID(docID)
	assert.Equal(t, docID, byID["_id"])
	assert.Equal(t, orgID, byID["org_id"])

	byLineage := scope.ByLineage(lineageID)
	assert.Equal(t, lineageID, byLineage["lineage_id"])
	assert.Equal(t, orgID, byLineage["org_id"])
}
