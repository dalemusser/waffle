// toolkit/db/mongodb/db.go
package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const mongoConnectTimeout = 10 * time.Second

// Connect opens a Mongo connection with a bounded timeout derived from the
// provided parent context. The returned client must be disconnected by the caller.
func Connect(ctx context.Context, uri string, dbName string) (*mongo.Client, error) {
	// Derive a timeout from the parent context so connection attempts
	// do not hang indefinitely.
	ctx, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}
