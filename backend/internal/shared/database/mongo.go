package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/masterfabric-go/masterfabric/internal/shared/config"
)

// MongoDB holds the object-database client and the application database handle.
//
// PostgreSQL answers "who is this?"; MongoDB answers "what was played and how
// was it scored?". Keeping that line sharp is what stops the two stores from
// drifting into a single confused schema.
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoClient connects to MongoDB and verifies the connection.
func NewMongoClient(ctx context.Context, cfg config.MongoConfig) (*MongoDB, error) {
	opts := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return &MongoDB{Client: client, Database: client.Database(cfg.Database)}, nil
}

// Close disconnects the client.
func (m *MongoDB) Close(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return nil
	}
	return m.Client.Disconnect(ctx)
}
