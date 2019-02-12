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

// This file wires the generic interfaces up to Google Cloud Platform (GCP). It
// won't be directly included in the final binary, since it includes a Wire
// injector template function (setupGCP), but the declarations will be copied
// into wire_gen.go when Wire is run.

// setupGCP is a Wire injector function that sets up the Application using GCP.
func setupGCP(ctx context.Context, flags *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		gcpcloud.GCP,
		cloudmysql.Open,
		ApplicationSet,
		GcpBucket,
		GcpMOTDVar,
		GcpSQLParams,
	)
	return nil, nil, nil
}

// gcpBucket is a Wire provider function that returns the GCS bucket based on
// the command-line flags.
func GcpBucket(ctx context.Context, flags *Config, client *gcp.HTTPClient) (*blob.Bucket, error) {
	return gcsblob.OpenBucket(ctx, client, flags.Bucket, nil)
}

// gcpSQLParams is a Wire provider function that returns the Cloud SQL
// connection parameters based on the command-line flags. Other providers inside
// gcpcloud.GCP use the parameters to construct a *sql.DB.
func GcpSQLParams(id gcp.ProjectID, flags *Config) *cloudmysql.Params {
	return &cloudmysql.Params{
		ProjectID: string(id),
		Region:    flags.CloudSQLRegion,
		Instance:  flags.DbHost,
		Database:  flags.DbName,
		User:      flags.DbUser,
		Password:  flags.DbPassword,
	}
}

// gcpMOTDVar is a Wire provider function that returns the Message of the Day
// variable from Runtime Configurator.
func GcpMOTDVar(ctx context.Context, client pb.RuntimeConfigManagerClient, project gcp.ProjectID, flags *Config) (*runtimevar.Variable, func(), error) {
	name := runtimeconfigurator.ResourceName{
		ProjectID: string(project),
		Config:    flags.RuntimeConfigName,
		Variable:  flags.RunVar,
	}
	v, err := runtimeconfigurator.NewVariable(client, name, runtimevar.StringDecoder, &runtimeconfigurator.Options{
		WaitDuration: flags.RunVarWaitTime,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}
