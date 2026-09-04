// Package model holds the training documents: scenarios, characters, rubrics
// and the sessions played against them.
//
// Every document here is immutable and versioned. Editing a scenario does not
// change a row; it publishes a new version, and the old one stays exactly as it
// was. This is not tidiness — it is the product's core claim. A certificate has
// to be defensible years later, which means the exact scenario, character and
// rubric it was awarded under must still be readable in their original form.
package model

import (
	"time"

	"github.com/google/uuid"
)

// DocumentStatus is the publication state of a version.
type DocumentStatus string

const (
	// DocumentStatusDraft is editable in place and can never be referenced by a
	// session. Drafts are the one exception to immutability, and the exception
	// holds precisely because nothing can point at them.
	DocumentStatusDraft DocumentStatus = "draft"
	// DocumentStatusPublished is frozen and may be played.
	DocumentStatusPublished DocumentStatus = "published"
	// DocumentStatusArchived is frozen and withdrawn from new sessions.
	// Existing references stay valid: archiving must never invalidate a
	// certificate already awarded.
	DocumentStatusArchived DocumentStatus = "archived"
)

// Document is the envelope shared by every versioned training record.
//
// It is embedded rather than inherited so that each concrete type keeps its own
// free-form body — a "reluctant customer" character and an "auditor" character
// have almost no fields in common, which is the whole reason this data lives in
// an object database.
type Document struct {
	// ID identifies this exact version.
	ID uuid.UUID `bson:"_id" json:"id"`

	// OrgID is the tenant boundary. It is present on every document in every
	// collection without exception, and every query filters on it. In a
	// multi-tenant system this is the one structural defence against cross-org
	// leakage; anything enforced only in application code is one forgotten
	// filter away from failing.
	OrgID uuid.UUID `bson:"org_id" json:"org_id"`

	// LineageID is stable across all versions of the same logical record. "The
	// scenario" is a lineage; "version 3 of the scenario" is a document.
	LineageID uuid.UUID `bson:"lineage_id" json:"lineage_id"`

	// Version counts up from 1 within a lineage.
	Version int `bson:"version" json:"version"`

	// Status controls whether this version may be played.
	Status DocumentStatus `bson:"status" json:"status"`

	// SupersedesID points at the version this one was derived from, giving the
	// lineage a readable edit history rather than a bag of versions.
	SupersedesID *uuid.UUID `bson:"supersedes_id,omitempty" json:"supersedes_id,omitempty"`

	CreatedBy   uuid.UUID  `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	PublishedAt *time.Time `bson:"published_at,omitempty" json:"published_at,omitempty"`
	ArchivedAt  *time.Time `bson:"archived_at,omitempty" json:"archived_at,omitempty"`
}

// NewLineage starts a fresh draft as version 1 of a new lineage.
func NewLineage(orgID, createdBy uuid.UUID, now time.Time) Document {
	id := uuid.New()
	return Document{
		ID:        id,
		OrgID:     orgID,
		LineageID: id,
		Version:   1,
		Status:    DocumentStatusDraft,
		CreatedBy: createdBy,
		CreatedAt: now.UTC(),
	}
}

// NextVersion derives the envelope for the successor of a published document.
//
// The new version starts as a draft: an edit becomes playable when its author
// publishes it, not the moment they start typing.
func (d Document) NextVersion(createdBy uuid.UUID, now time.Time) Document {
	previous := d.ID
	return Document{
		ID:           uuid.New(),
		OrgID:        d.OrgID,
		LineageID:    d.LineageID,
		Version:      d.Version + 1,
		Status:       DocumentStatusDraft,
		SupersedesID: &previous,
		CreatedBy:    createdBy,
		CreatedAt:    now.UTC(),
	}
}

// IsFrozen reports whether the document may no longer be edited in place.
func (d Document) IsFrozen() bool {
	return d.Status != DocumentStatusDraft
}

// IsPlayable reports whether a session may reference this version.
func (d Document) IsPlayable() bool {
	return d.Status == DocumentStatusPublished
}

// Reference is a pointer from a session to the exact document version it was
// played or scored against.
//
// It stores the version identifier, never the lineage alone. A reference that
// only named the lineage would silently re-point at whatever was published
// later, which is the precise failure this whole model exists to prevent.
type Reference struct {
	LineageID uuid.UUID `bson:"lineage_id" json:"lineage_id"`
	VersionID uuid.UUID `bson:"version_id" json:"version_id"`
	Version   int       `bson:"version" json:"version"`
}

// Ref builds a Reference to this document version.
func (d Document) Ref() Reference {
	return Reference{LineageID: d.LineageID, VersionID: d.ID, Version: d.Version}
}
