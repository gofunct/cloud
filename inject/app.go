package inject

import (
	"database/sql"
	"github.com/google/wire"
	"go.opencensus.io/trace"
	"log"
	"sync"
	"time"
	"gocloud.dev/blob"
	"gocloud.dev/health"
	"gocloud.dev/health/sqlhealth"
	"gocloud.dev/runtimevar"
	"gocloud.dev/server"
	"context"
)

type Config struct {
	Bucket          string
	DbHost          string
	DbName          string
	DbUser          string
	DbPassword      string
	RunVar         string
	RunVarWaitTime time.Duration

	CloudSQLRegion    string
	RuntimeConfigName string
	EnvFlag string
}

// ApplicationSet is the Wire provider set for the Guestbook Application that
// does not depend on the underlying platform.
var ApplicationSet = wire.NewSet(
	NewApplication,
	AppHealthChecks,
	trace.AlwaysSample,
)

// Application is the main server struct for Guestbook. It contains the state of
// the most recently read message of the day.
type Application struct {
	srv    *server.Server
	db     *sql.DB
	bucket *blob.Bucket

	// The following fields are protected by mu:
	mu   sync.RWMutex
	motd string // message of the day
}
// of the day variable.
func NewApplication(srv *server.Server, db *sql.DB, bucket *blob.Bucket, motdVar *runtimevar.Variable) *Application {
	app := &Application{
		srv:    srv,
		db:     db,
		bucket: bucket,
	}
	go app.WatchRunVar(motdVar)
	return app
}

// watchMOTDVar listens for changes in v and updates the app's message of the
// day. It is run in a separate goroutine.
func (app *Application) WatchRunVar(v *runtimevar.Variable) {
	ctx := context.Background()
	for {
		snap, err := v.Watch(ctx)
		if err != nil {
			log.Printf("watch MOTD variable: %v", err)
			continue
		}
		log.Println("updated MOTD to", snap.Value)
		app.mu.Lock()
		app.motd = snap.Value.(string)
		app.mu.Unlock()
	}
}

// appHealthChecks returns a health check for the database. This will signal
// to Kubernetes or other orchestrators that the server should not receive
// traffic until the server is able to connect to its database.
func AppHealthChecks(db *sql.DB) ([]health.Checker, func()) {
	dbCheck := sqlhealth.New(db)
	list := []health.Checker{dbCheck}
	return list, func() {
		dbCheck.Stop()
	}
}