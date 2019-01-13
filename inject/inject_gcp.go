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

func Gcp(ctx context.Context, c *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		gcpcloud.GCP,
		cloudmysql.Open,
		ApplicationSet,
		gcpBucket,
		gcpRunVar,
		gcpSQLParams,
	)
	return nil, nil, nil
}


func gcpBucket(ctx context.Context, c *Config, client *gcp.HTTPClient) (*blob.Bucket, error) {
	return gcsblob.OpenBucket(ctx, client, c.Bucket, nil)
}

// gcpSQLParams is a Wire provider function that returns the Cloud SQL
// connection parameters based on the command-line c. Other providers inside
// gcpcloud.GCP use the parameters to construct a *sql.DB.
func gcpSQLParams(id gcp.ProjectID, c *Config) *cloudmysql.Params {
	return &cloudmysql.Params{
		ProjectID: string(id),
		Region:    c.SqlRegion,
		Instance:  c.DbHost,
		Database:  c.DbName,
		User:      c.DbUser,
		Password:  c.DbPassword,
	}
}

// gcpMOTDVar is a Wire provider function that returns the Message of the Day
// variable from Runtime Configurator.
func gcpRunVar(ctx context.Context, client pb.RuntimeConfigManagerClient, project gcp.ProjectID, c *Config) (*runtimevar.Variable, func(), error) {
	name := runtimeconfigurator.ResourceName{
		ProjectID: string(project),
		Config:    c.RunVarConfigName,
		Variable:  c.RunVar,
	}
	v, err := runtimeconfigurator.NewVariable(client, name, runtimevar.StringDecoder, &runtimeconfigurator.Options{
		WaitDuration: c.RunVarWait,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}
