// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package inject

import (
	"context"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"database/sql"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-sql-driver/mysql"
	"go.opencensus.io/trace"
	"gocloud.dev/aws/rds"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"gocloud.dev/gcp/cloudsql"
	"gocloud.dev/mysql/cloudmysql"
	"gocloud.dev/mysql/rdsmysql"
	"gocloud.dev/requestlog"
	"gocloud.dev/runtimevar"
	"gocloud.dev/runtimevar/filevar"
	"gocloud.dev/runtimevar/paramstore"
	"gocloud.dev/runtimevar/runtimeconfigurator"
	"gocloud.dev/server"
	"gocloud.dev/server/sdserver"
	"gocloud.dev/server/xrayserver"
	"google.golang.org/genproto/googleapis/cloud/runtimeconfig/v1beta1"
	"net/http"
)

// Injectors from aws.go:

func SetupAWS(ctx context.Context, flags *Config) (*Application, func(), error) {
	ncsaLogger := xrayserver.NewRequestLogger()
	client := _wireClientValue
	certFetcher := &rds.CertFetcher{
		Client: client,
	}
	params := AwsSQLParams(flags)
	db, cleanup, err := rdsmysql.Open(ctx, certFetcher, params)
	if err != nil {
		return nil, nil, err
	}
	v, cleanup2 := AppHealthChecks(db)
	options := _wireOptionsValue
	sessionSession, err := session.NewSessionWithOptions(options)
	if err != nil {
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	xRay := xrayserver.NewXRayClient(sessionSession)
	exporter, cleanup3, err := xrayserver.NewExporter(xRay)
	if err != nil {
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	sampler := trace.AlwaysSample()
	defaultDriver := _wireDefaultDriverValue
	serverOptions := &server.Options{
		RequestLogger:         ncsaLogger,
		HealthChecks:          v,
		TraceExporter:         exporter,
		DefaultSamplingPolicy: sampler,
		Driver:                defaultDriver,
	}
	serverServer := server.New(serverOptions)
	bucket, err := AwsBucket(ctx, sessionSession, flags)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	variable, err := AwsMOTDVar(ctx, sessionSession, flags)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	application := NewApplication(serverServer, db, bucket, variable)
	return application, func() {
		cleanup3()
		cleanup2()
		cleanup()
	}, nil
}

var (
	_wireClientValue        = http.DefaultClient
	_wireOptionsValue       = session.Options{}
	_wireDefaultDriverValue = &server.DefaultDriver{}
)

// Injectors from gcp.go:

func setupGCP(ctx context.Context, flags *Config) (*Application, func(), error) {
	stackdriverLogger := sdserver.NewRequestLogger()
	roundTripper := gcp.DefaultTransport()
	credentials, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, nil, err
	}
	tokenSource := gcp.CredentialsTokenSource(credentials)
	httpClient, err := gcp.NewHTTPClient(roundTripper, tokenSource)
	if err != nil {
		return nil, nil, err
	}
	remoteCertSource := cloudsql.NewCertSource(httpClient)
	projectID, err := gcp.DefaultProjectID(credentials)
	if err != nil {
		return nil, nil, err
	}
	params := GcpSQLParams(projectID, flags)
	db, err := cloudmysql.Open(ctx, remoteCertSource, params)
	if err != nil {
		return nil, nil, err
	}
	v, cleanup := AppHealthChecks(db)
	monitoredresourceInterface := monitoredresource.Autodetect()
	exporter, cleanup2, err := sdserver.NewExporter(projectID, tokenSource, monitoredresourceInterface)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	sampler := trace.AlwaysSample()
	defaultDriver := _wireDefaultDriverValue
	options := &server.Options{
		RequestLogger:         stackdriverLogger,
		HealthChecks:          v,
		TraceExporter:         exporter,
		DefaultSamplingPolicy: sampler,
		Driver:                defaultDriver,
	}
	serverServer := server.New(options)
	bucket, err := GcpBucket(ctx, flags, httpClient)
	if err != nil {
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	runtimeConfigManagerClient, cleanup3, err := runtimeconfigurator.Dial(ctx, tokenSource)
	if err != nil {
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	variable, cleanup4, err := GcpMOTDVar(ctx, runtimeConfigManagerClient, projectID, flags)
	if err != nil {
		cleanup3()
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	application := NewApplication(serverServer, db, bucket, variable)
	return application, func() {
		cleanup4()
		cleanup3()
		cleanup2()
		cleanup()
	}, nil
}

// Injectors from local.go:

func SetupLocal(ctx context.Context, flags *Config) (*Application, func(), error) {
	logger := _wireLoggerValue
	db, err := DialLocalSQL(flags)
	if err != nil {
		return nil, nil, err
	}
	v, cleanup := AppHealthChecks(db)
	exporter := _wireExporterValue
	sampler := trace.AlwaysSample()
	defaultDriver := _wireDefaultDriverValue
	options := &server.Options{
		RequestLogger:         logger,
		HealthChecks:          v,
		TraceExporter:         exporter,
		DefaultSamplingPolicy: sampler,
		Driver:                defaultDriver,
	}
	serverServer := server.New(options)
	bucket, err := LocalBucket(flags)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	variable, cleanup2, err := LocalRuntimeVar(flags)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	application := NewApplication(serverServer, db, bucket, variable)
	return application, func() {
		cleanup2()
		cleanup()
	}, nil
}

var (
	_wireLoggerValue   = requestlog.Logger(nil)
	_wireExporterValue = trace.Exporter(nil)
)

// aws.go:

// awsBucket is a Wire provider function that returns the S3 bucket based on the
// command-line flags.
func AwsBucket(ctx context.Context, cp client.ConfigProvider, flags *Config) (*blob.Bucket, error) {
	return s3blob.OpenBucket(ctx, cp, flags.Bucket, nil)
}

// awsSQLParams is a Wire provider function that returns the RDS SQL connection
// parameters based on the command-line flags. Other providers inside
// awscloud.AWS use the parameters to construct a *sql.DB.
func AwsSQLParams(flags *Config) *rdsmysql.Params {
	return &rdsmysql.Params{
		Endpoint: flags.DbHost,
		Database: flags.DbName,
		User:     flags.DbUser,
		Password: flags.DbPassword,
	}
}

// awsMOTDVar is a Wire provider function that returns the Message of the Day
// variable from SSM Parameter Store.
func AwsMOTDVar(ctx context.Context, sess client.ConfigProvider, flags *Config) (*runtimevar.Variable, error) {
	return paramstore.NewVariable(sess, flags.RunVar, runtimevar.StringDecoder, &paramstore.Options{
		WaitDuration: flags.RunVarWaitTime,
	})
}

// gcp.go:

// gcpBucket is a Wire provider function that returns the GCS bucket based on
// the command-line flags.
func GcpBucket(ctx context.Context, flags *Config, client2 *gcp.HTTPClient) (*blob.Bucket, error) {
	return gcsblob.OpenBucket(ctx, client2, flags.Bucket, nil)
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
func GcpMOTDVar(ctx context.Context, client2 runtimeconfig.RuntimeConfigManagerClient, project gcp.ProjectID, flags *Config) (*runtimevar.Variable, func(), error) {
	name := runtimeconfigurator.ResourceName{
		ProjectID: string(project),
		Config:    flags.RuntimeConfigName,
		Variable:  flags.RunVar,
	}
	v, err := runtimeconfigurator.NewVariable(client2, name, runtimevar.StringDecoder, &runtimeconfigurator.Options{
		WaitDuration: flags.RunVarWaitTime,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}

// local.go:

// localBucket is a Wire provider function that returns a directory-based bucket
// based on the command-line flags.
func LocalBucket(flags *Config) (*blob.Bucket, error) {
	return fileblob.OpenBucket(flags.Bucket, nil)
}

// dialLocalSQL is a Wire provider function that connects to a MySQL database
// (usually on localhost).
func DialLocalSQL(flags *Config) (*sql.DB, error) {
	cfg := &mysql.Config{
		Net:                  "tcp",
		Addr:                 flags.DbHost,
		DBName:               flags.DbName,
		User:                 flags.DbUser,
		Passwd:               flags.DbPassword,
		AllowNativePasswords: true,
	}
	return sql.Open("mysql", cfg.FormatDSN())
}

// localRuntimeVar is a Wire provider function that returns the Message of the
// Day variable based on a local file.
func LocalRuntimeVar(flags *Config) (*runtimevar.Variable, func(), error) {
	v, err := filevar.New(flags.RunVar, runtimevar.StringDecoder, &filevar.Options{
		WaitDuration: flags.RunVarWaitTime,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}