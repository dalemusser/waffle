// db/mongo/mongo.go
package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Connect opens a Mongo/DocumentDB connection using the given URI and timeout.
// It performs a Ping to ensure the connection is usable before returning.
//
// The caller is responsible for calling client.Disconnect(...) when done.
func Connect(uri string, timeout time.Duration) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Configure connection pool for better stability
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMinPoolSize(2).                          // Keep minimum connections ready
		SetMaxPoolSize(50).                         // Limit max connections (default 100)
		SetMaxConnIdleTime(5 * time.Minute).        // Close idle connections after 5 min
		SetServerSelectionTimeout(10 * time.Second) // Fail fast if server unavailable

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return client, nil
}
