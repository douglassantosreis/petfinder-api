package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	raw *mongo.Client
	db  *mongo.Database
}

func New(ctx context.Context, uri, database string) (*Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, err
	}
	return &Client{
		raw: client,
		db:  client.Database(database),
	}, nil
}

func (c *Client) DB() *mongo.Database {
	return c.db
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.raw.Disconnect(ctx)
}
