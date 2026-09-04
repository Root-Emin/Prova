package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLineage_StartsAsDraftVersionOne(t *testing.T) {
	orgID, author := uuid.New(), uuid.New()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

	doc := NewLineage(orgID, author, now)

	assert.Equal(t, orgID, doc.OrgID)
	assert.Equal(t, doc.ID, doc.LineageID, "the first version names its own lineage")
	assert.Equal(t, 1, doc.Version)
	assert.Equal(t, DocumentStatusDraft, doc.Status)
	assert.Equal(t, author, doc.CreatedBy)
	assert.Nil(t, doc.SupersedesID)
}

// An edit publishes a new version; it never mutates the old one. That is what
// keeps a years-old certificate defensible.
func TestNextVersion_PreservesLineageAndLinksBack(t *testing.T) {
	orgID, author, editor := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

	v1 := NewLineage(orgID, author, now)
	v1.Status = DocumentStatusPublished

	v2 := v1.NextVersion(editor, now.Add(time.Hour))

	assert.Equal(t, v1.LineageID, v2.LineageID)
	assert.Equal(t, orgID, v2.OrgID)
	assert.Equal(t, 2, v2.Version)
	assert.NotEqual(t, v1.ID, v2.ID)
	require.NotNil(t, v2.SupersedesID)
	assert.Equal(t, v1.ID, *v2.SupersedesID)
	assert.Equal(t, editor, v2.CreatedBy)

	// The predecessor is untouched.
	assert.Equal(t, DocumentStatusPublished, v1.Status)
	assert.Equal(t, 1, v1.Version)
}

// An edit becomes playable when its author publishes it, not when they start
// typing.
func TestNextVersion_StartsAsDraft(t *testing.T) {
	v1 := NewLineage(uuid.New(), uuid.New(), time.Now())
	v1.Status = DocumentStatusPublished

	v2 := v1.NextVersion(uuid.New(), time.Now())

	assert.Equal(t, DocumentStatusDraft, v2.Status)
	assert.False(t, v2.IsPlayable())
	assert.False(t, v2.IsFrozen())
}

func TestDocument_StatusPredicates(t *testing.T) {
	doc := NewLineage(uuid.New(), uuid.New(), time.Now())

	assert.False(t, doc.IsFrozen())
	assert.False(t, doc.IsPlayable())

	doc.Status = DocumentStatusPublished
	assert.True(t, doc.IsFrozen())
	assert.True(t, doc.IsPlayable())

	// Archiving withdraws a document from new sessions but must never
	// invalidate a certificate already awarded under it.
	doc.Status = DocumentStatusArchived
	assert.True(t, doc.IsFrozen())
	assert.False(t, doc.IsPlayable())
}

// A reference that named only the lineage would silently re-point at whatever
// was published later — the exact failure this model exists to prevent.
func TestRef_PinsTheExactVersion(t *testing.T) {
	v1 := NewLineage(uuid.New(), uuid.New(), time.Now())
	v1.Status = DocumentStatusPublished
	v2 := v1.NextVersion(uuid.New(), time.Now())

	ref := v1.Ref()

	assert.Equal(t, v1.ID, ref.VersionID)
	assert.Equal(t, v1.LineageID, ref.LineageID)
	assert.Equal(t, 1, ref.Version)
	assert.NotEqual(t, v2.ID, ref.VersionID)
}
