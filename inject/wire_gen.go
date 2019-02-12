// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package inject

import (
	"context"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"go.opencensus.io/trace"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/gcp"
	"gocloud.dev/gcp/cloudsql"
	"gocloud.dev/mysql/cloudmysql"
	"gocloud.dev/runtimevar"
	"gocloud.dev/runtimevar/runtimeconfigurator"
	"gocloud.dev/server"
	"gocloud.dev/server/sdserver"
	"google.golang.org/genproto/googleapis/cloud/runtimeconfig/v1beta1"
)

// Injectors from gcp.go:

func SetupGCP(ctx context.Context, flags *Config) (*Application, func(), error) {
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
	variable, cleanup4, err := GcpRunVar(ctx, runtimeConfigManagerClient, projectID, flags)
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

var (
	_wireDefaultDriverValue = &server.DefaultDriver{}
)

// gcp.go:

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

func GcpRunVar(ctx context.Context, client runtimeconfig.RuntimeConfigManagerClient, project gcp.ProjectID, flags *Config) (*runtimevar.Variable, func(), error) {
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
