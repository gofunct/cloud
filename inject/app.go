package inject

import (
	"database/sql"
	"github.com/google/wire"
	"go.opencensus.io/trace"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"gocloud.dev/blob"
	"gocloud.dev/health"
	"gocloud.dev/health/sqlhealth"
	"gocloud.dev/runtimevar"
	"gocloud.dev/server"
	"context"
	"github.com/gorilla/mux"
)

type Config struct {
	Env 	string `json:"env"`
	Bucket          string `json:"bucket"`
	DbHost          string `json:"dbhost"`
	DbName          string `json:"dbname"`
	DbUser          string	`json:"dbuser"`
	DbPass     string	`json:"dbpass"`
	SQLRegion    	string	`json:"sqlregion"`
	RunVar         	string 	`json:"runvar"`
	RunVarWait 		time.Duration `json:"runvarwait"`
	RunVarName 		string `json:"runvarname"`

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
	Server    *server.Server
	Db     *sql.DB
	Bucket *blob.Bucket
	// The following fields are protected by mu:
	Mutex   sync.RWMutex
	Runvar string
}
// of the day variable.
func NewApplication(srv *server.Server, db *sql.DB, bucket *blob.Bucket, runvar *runtimevar.Variable) *Application {
	app := &Application{
		Server:    srv,
		Db:     db,
		Bucket: bucket,
	}
	go app.WatchRunVar(runvar)
	return app
}

// WatchRunVar listens for changes in v and updates the app's message of the
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
		app.Mutex.Lock()
		app.Runvar = snap.Value.(string)
		app.Mutex.Unlock()
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


// serveBlob handles a request for a static asset by retrieving it from a bucket.
func (app *Application) ServeBlob(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	blobRead, err := app.Bucket.NewReader(r.Context(), key, nil)
	if err != nil {
		// TODO: Distinguish 404.
		log.Println("serve blob:", err)
		http.Error(w, "blob read error", http.StatusInternalServerError)
		return
	}
	// TODO: Get content type from blob storage.
	switch {
	case strings.HasSuffix(key, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(key, ".jpg"):
		w.Header().Set("Content-Type", "image/jpeg")
	case strings.HasSuffix(key, ".html"):
		w.Header().Set("Content-Type", "text/html")
	case strings.HasSuffix(key, ".mpeg"):
		w.Header().Set("Content-Type", "video/mpeg")
	case strings.HasSuffix(key, ".mp4"):
		w.Header().Set("Content-Type", "video/mp4")
	case strings.HasSuffix(key, ".csv"):
		w.Header().Set("Content-Type", "text/csv")


	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(blobRead.Size(), 10))
	if _, err = io.Copy(w, blobRead); err != nil {
		log.Println("Copying blob:", err)
	}
}
