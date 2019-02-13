//+build wireinject

package inject

import (
	"cloud.google.com/go/firestore"
	"context"
	"github.com/google/wire"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/gcp"
	"gocloud.dev/gcp/gcpcloud"
)

// setupGCP is a Wire injector function that sets up the Application using GCP.
func SetupGCP(ctx context.Context, flags *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		gcpcloud.GCP,
		ApplicationSet,
		GcpBucket,
		NewFireStoreClient,
	)
	return nil, nil, nil
}

func GcpBucket(ctx context.Context, flags *Config, client *gcp.HTTPClient) (*blob.Bucket, error) {
	return gcsblob.OpenBucket(ctx, client, flags.Bucket, nil)
}

func NewFireStoreClient(ctx context.Context, c *Config) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, c.Project)
	if err != nil {
		return nil, err
	}
	return client, nil
}