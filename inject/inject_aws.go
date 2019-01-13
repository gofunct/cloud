//+build wireinject

package inject

import (
	"context"

	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/google/wire"
	"gocloud.dev/aws/awscloud"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/mysql/rdsmysql"
	"gocloud.dev/runtimevar"
	"gocloud.dev/runtimevar/paramstore"
)

func Aws(ctx context.Context, c *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		awscloud.AWS,
		rdsmysql.Open,
		ApplicationSet,
		awsBucket,
		awsRunVar,
		awsSQLParams,
	)
	return nil, nil, nil
}

func awsBucket(ctx context.Context, cp awsclient.ConfigProvider, c *Config) (*blob.Bucket, error) {
	return s3blob.OpenBucket(ctx, cp, c.Bucket, nil)
}

// awsSQLParams is a Wire provider function that returns the RDS SQL connection
// parameters based on the command-line c. Other providers inside
// awscloud.AWS use the parameters to construct a *sql.DB.
func awsSQLParams(c *Config) *rdsmysql.Params {
	return &rdsmysql.Params{
		Endpoint: c.DbHost,
		Database: c.DbName,
		User:     c.DbUser,
		Password: c.DbPassword,
	}
}

// awsMOTDVar is a Wire provider function that returns the Message of the Day
// variable from SSM Parameter Store.
func awsRunVar(ctx context.Context, sess awsclient.ConfigProvider, c *Config) (*runtimevar.Variable, error) {
	return paramstore.NewVariable(sess, c.RunVar, runtimevar.StringDecoder, &paramstore.Options{
		WaitDuration: c.RunVarWait,
	})
}
