// Package mongo holds the object-database wiring for the training documents.
package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection names. Every one of these stores documents carrying org_id.
const (
	CollectionScenarios  = "scenarios"
	CollectionCharacters = "characters"
	CollectionRubrics    = "rubrics"
	CollectionSessions   = "sessions"
)

// versionedCollections are the collections holding model.Document envelopes.
var versionedCollections = []string{
	CollectionScenarios,
	CollectionCharacters,
	CollectionRubrics,
}

// EnsureIndexes creates the indexes the training collections rely on.
//
// Two of them are correctness, not performance. The unique (org_id, lineage_id,
// version) index makes a duplicate version number impossible even when two
// editors publish at the same instant — without it, "version 3" could name two
// different documents and every certificate referring to it becomes ambiguous.
// Leading every index with org_id keeps tenant-scoped queries on an index path,
// so the cheap query and the correct query are the same query.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	for _, name := range versionedCollections {
		models := []mongo.IndexModel{
			{
				Keys: bson.D{
					{Key: "org_id", Value: 1},
					{Key: "lineage_id", Value: 1},
					{Key: "version", Value: 1},
				},
				Options: options.Index().SetUnique(true).SetName("uq_org_lineage_version"),
			},
			{
				Keys: bson.D{
					{Key: "org_id", Value: 1},
					{Key: "status", Value: 1},
					{Key: "created_at", Value: -1},
				},
				Options: options.Index().SetName("idx_org_status_created"),
			},
		}
		if _, err := db.Collection(name).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("ensure indexes on %s: %w", name, err)
		}
	}

	sessionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "started_at", Value: -1}},
			Options: options.Index().SetName("idx_org_user_started"),
		},
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "scenario.lineage_id", Value: 1}},
			Options: options.Index().SetName("idx_org_scenario_lineage"),
		},
	}
	if _, err := db.Collection(CollectionSessions).Indexes().CreateMany(ctx, sessionIndexes); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionSessions, err)
	}

	return nil
}
