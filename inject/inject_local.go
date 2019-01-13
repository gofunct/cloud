//+build wireinject

package inject

import (
	"context"
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"go.opencensus.io/trace"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/requestlog"
	"gocloud.dev/runtimevar"
	"gocloud.dev/runtimevar/filevar"
	"gocloud.dev/server"
)

func Local(ctx context.Context, c *Config) (*Application, func(), error) {
	// This will be filled in by Wire with providers from the provider sets in
	// wire.Build.
	wire.Build(
		wire.InterfaceValue(new(requestlog.Logger), requestlog.Logger(nil)),
		wire.InterfaceValue(new(trace.Exporter), trace.Exporter(nil)),
		server.Set,
		ApplicationSet,
		dialLocalSQL,
		localBucket,
		localRunVar,
	)
	return nil, nil, nil
}

// localBucket is a Wire provider function that returns a directory-based bucket
// based on the command-line c.
func localBucket(c *Config) (*blob.Bucket, error) {
	return fileblob.OpenBucket(c.Bucket, nil)
}

// dialLocalSQL is a Wire provider function that connects to a MySQL database
// (usually on localhost).
func dialLocalSQL(c *Config) (*sql.DB, error) {
	cfg := &mysql.Config{
		Net:                  "tcp",
		Addr:                 c.DbHost,
		DBName:               c.DbName,
		User:                 c.DbUser,
		Passwd:               c.DbPassword,
		AllowNativePasswords: true,
	}
	return sql.Open("mysql", cfg.FormatDSN())
}

// localRuntimeVar is a Wire provider function that returns the Message of the
// Day variable based on a local file.
func localRunVar(c *Config) (*runtimevar.Variable, func(), error) {
	v, err := filevar.New(c.RunVar, runtimevar.StringDecoder, &filevar.Options{
		WaitDuration: c.RunVarWait,
	})
	if err != nil {
		return nil, nil, err
	}
	return v, func() { v.Close() }, nil
}
