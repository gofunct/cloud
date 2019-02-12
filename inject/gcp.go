//+build wireinject

package inject

import (
	"context"

	"github.com/google/wire"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/gcp"
	"gocloud.dev/gcp/gcpcloud"
	"gocloud.dev/mysql/cloudmysql"
	"gocloud.dev/runtimevar"
	"gocloud.dev/runtimevar/runtimeconfigurator"
	pb "google.golang.org/genproto/googleapis/cloud/runtimeconfig/v1beta1"
)

// setupGCP is a Wire injector function that sets up the Application using GCP.
func SetupGCP(ctx context.Context, flags *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		gcpcloud.GCP,
		cloudmysql.Open,
		ApplicationSet,
		GcpBucket,
		GcpRunVar,
		GcpSQLParams,
	)
	return nil, nil, nil
}

func GcpBucket(ctx context.Context, flags *Config, client *gcp.HTTPClient) (*blob.Bucket, error) {
	return gcsblob.OpenBucket(ctx, client, flags.Bucket, nil)
}

func GcpSQLParams(id gcp.ProjectID, flags *Config) *cloudmysql.Params {
	return &cloudmysql.Params{
		ProjectID: string(id),
		Region:    flags.SQLRegion,
		Instance:  flags.DbHost,
		Database:  flags.DbName,
		User:      flags.DbUser,
		Password:  flags.DbPass,
	}
}

func GcpRunVar(ctx context.Context, client pb.RuntimeConfigManagerClient, project gcp.ProjectID, flags *Config) (*runtimevar.Variable, func(), error) {
	name := runtimeconfigurator.ResourceName{
		ProjectID: string(project),
		Config:    flags.RunVarName,
		Variable:  flags.RunVar,
	}
	v, err := runtimeconfigurator.NewVariable(client, name, runtimevar.StringDecoder, &runtimeconfigurator.Options{
		WaitDuration: flags.RunVarWait,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}
