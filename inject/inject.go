package inject

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"github.com/gorilla/mux"
	"go.opencensus.io/trace"
	"gocloud.dev/blob"
	"gocloud.dev/health"
	"gocloud.dev/health/sqlhealth"
	"gocloud.dev/runtimevar"
	"gocloud.dev/server"
)

type Config struct {
	Bucket          string `mapstructure:"bucket"`
	DbHost          string `mapstructure:"dbHost"`
	DbName          string `mapstructure:"dbName"`
	DbUser          string `mapstructure:"dbUser"`
	DbPassword      string `mapstructure:"dbPassword"`
	RunVar         string `mapstructure:"runVar"`
	RunVarWait time.Duration `mapstructure:"runVarWait"`
	SqlRegion    string `mapstructure:"sqlRegion"`
	RunVarConfigName string `mapstructure:"runVarConfigName"`
	Env string `mapstructure:"env"`
}

// applicationSet is the Wire provider set for the Guestbook Application that
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

// newApplication creates a new Application struct based on the backends and the message
// of the day variable.
func NewApplication(srv *server.Server, db *sql.DB, bucket *blob.Bucket, RunVar *runtimevar.Variable) *Application {
	app := &Application{
		srv:    srv,
		db:     db,
		bucket: bucket,
	}
	go app.WatchRunVar(RunVar)
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

// serveBlob handles a request for a static asset by retrieving it from a bucket.
func (app *Application) ServeBlob(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	blobRead, err := app.bucket.NewReader(r.Context(), key, nil)
	if err != nil {
		log.Println("serve blob:", err)
		http.Error(w, "blob read error", http.StatusInternalServerError)
		return
	}
	switch {
	case strings.HasSuffix(key, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(key, ".jpg"):
		w.Header().Set("Content-Type", "image/jpeg")
	default:
		w.Header().Set("Content-Type", "Application/octet-stream")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(blobRead.Size(), 10))
	if _, err = io.Copy(w, blobRead); err != nil {
		log.Println("Copying blob:", err)
	}
}

func AppHealthChecks(db *sql.DB) ([]health.Checker, func()) {
	dbCheck := sqlhealth.New(db)
	list := []health.Checker{dbCheck}
	return list, func() {
		dbCheck.Stop()
	}
}
