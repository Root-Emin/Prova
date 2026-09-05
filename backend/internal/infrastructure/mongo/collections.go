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
	CollectionScenarios     = "scenarios"
	CollectionCharacters    = "characters"
	CollectionRubrics       = "rubrics"
	CollectionLLMProfiles   = "llm_profiles"
	CollectionSessions      = "sessions"
	CollectionTurns         = "turns"
	CollectionScores        = "scores"
	CollectionRoutingRecord = "routing_records"
)

// versionedCollections are the collections holding model.Document envelopes.
var versionedCollections = []string{
	CollectionScenarios,
	CollectionCharacters,
	CollectionRubrics,
	CollectionLLMProfiles,
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

	// Kademe araması yalnızca LLM profillerinde anlamlı: hangi modelin şu an
	// yürürlükte olduğu her konuşma sırasında sorulur.
	if _, err := db.Collection(CollectionLLMProfiles).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "org_id", Value: 1},
			{Key: "tier", Value: 1},
			{Key: "status", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().SetName("idx_org_tier_status"),
	}); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionLLMProfiles, err)
	}

	sessionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "employee_id", Value: 1}, {Key: "started_at", Value: -1}},
			Options: options.Index().SetName("idx_org_employee_started"),
		},
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "scenario.lineage_id", Value: 1}},
			Options: options.Index().SetName("idx_org_scenario_lineage"),
		},
		{
			// Kimliksizleştirme işi çalışanın tüm oturumlarını bulur;
			// kiracı sınırı olmadan sorgulanan tek index budur, çünkü silme
			// talebi kullanıcıya aittir ve kullanıcı birden fazla
			// organizasyonda oturum oynamış olabilir.
			Keys:    bson.D{{Key: "employee_id", Value: 1}},
			Options: options.Index().SetName("idx_employee"),
		},
	}
	if _, err := db.Collection(CollectionSessions).Indexes().CreateMany(ctx, sessionIndexes); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionSessions, err)
	}

	// Konuşma sıralarında (org, session, index) benzersiz: iki eşzamanlı
	// gönderim aynı sıra numarasını alamaz. Alıntı doğrulaması sıra
	// numarasına göre metin aradığı için, çakışan bir numara alıntıyı
	// yanlış metinde aratırdı.
	turnIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "org_id", Value: 1},
				{Key: "session_id", Value: 1},
				{Key: "index", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("uq_org_session_index"),
		},
		{
			Keys:    bson.D{{Key: "session_id", Value: 1}},
			Options: options.Index().SetName("idx_session"),
		},
		{
			// Yeni SubmitTurn çağrılarında aynı request_id ve role ikinci kez
			// yazılamaz. Partial index eski turn'lerde olmayan alanın unique
			// kısıtına takılmasını önler.
			Keys: bson.D{
				{Key: "org_id", Value: 1},
				{Key: "session_id", Value: 1},
				{Key: "request_id", Value: 1},
				{Key: "role", Value: 1},
			},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"request_id": bson.M{"$exists": true}}).
				SetName("uq_org_session_request_role"),
		},
	}
	if _, err := db.Collection(CollectionTurns).Indexes().CreateMany(ctx, turnIndexes); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionTurns, err)
	}

	// Oturum başına en fazla bir puan. Benzersizlik, iki eşzamanlı bitirme
	// isteğinin aynı oturuma iki farklı puan yazmasını engeller.
	if _, err := db.Collection(CollectionScores).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "session_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uq_org_session"),
	}); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionScores, err)
	}

	routingIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_org_created"),
		},
		{
			Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "session_id", Value: 1}, {Key: "created_at", Value: 1}},
			Options: options.Index().SetName("idx_org_session_created"),
		},
	}
	if _, err := db.Collection(CollectionRoutingRecord).Indexes().CreateMany(ctx, routingIndexes); err != nil {
		return fmt.Errorf("ensure indexes on %s: %w", CollectionRoutingRecord, err)
	}

	return nil
}
