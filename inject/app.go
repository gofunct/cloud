package inject

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/google/wire"
	"github.com/gorilla/mux"
	"go.opencensus.io/trace"
	"gocloud.dev/blob"
	"gocloud.dev/health"
	"gocloud.dev/health/sqlhealth"
	"gocloud.dev/runtimevar"
	"gocloud.dev/server"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Bucket     string        `json:"bucket"`
	DbHost     string        `json:"dbhost"`
	DbName     string        `json:"dbname"`
	DbUser     string        `json:"dbuser"`
	DbPass     string        `json:"dbpass"`
	SQLRegion  string        `json:"sqlregion"`
	RunVar     string        `json:"runvar"`
	RunVarWait time.Duration `json:"runvarwait"`
	RunVarName string        `json:"runvarname"`
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
	Server *server.Server
	Db     *sql.DB
	Bucket *blob.Bucket
	// The following fields are protected by mu:
	Mutex  sync.RWMutex
	Runvar string
}

// of the day variable.
func NewApplication(srv *server.Server, db *sql.DB, bucket *blob.Bucket, runvar *runtimevar.Variable) *Application {
	app := &Application{
		Server: srv,
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
func ServeBlob(app *Application, c *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
}

// index serves the server's landing page. It lists the 100 most recent
// greetings, shows a cloud environment banner, and displays the message of the
// day.
func Index(app *Application, c *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			MOTD      string
			Env       string
			BannerSrc string
			Greetings []greeting
		}
		app.Mutex.RLock()
		data.MOTD = app.Runvar
		app.Mutex.RUnlock()
		data.Env = "GCP"
		data.BannerSrc = "/blob/gcp.png"

		const query = "SELECT content FROM (SELECT content, post_date FROM greetings ORDER BY post_date DESC LIMIT 100) AS recent_greetings ORDER BY post_date ASC;"
		q, err := app.Db.QueryContext(r.Context(), query)
		if err != nil {
			log.Println("main page SQL error:", err)
			http.Error(w, "could not load greetings", http.StatusInternalServerError)
			return
		}
		defer q.Close()
		for q.Next() {
			var g greeting
			if err := q.Scan(&g.Content); err != nil {
				log.Println("main page SQL error:", err)
				http.Error(w, "could not load greetings", http.StatusInternalServerError)
				return
			}
			data.Greetings = append(data.Greetings, g)
		}
		if err := q.Err(); err != nil {
			log.Println("main page SQL error:", err)
			http.Error(w, "could not load greetings", http.StatusInternalServerError)
			return
		}
		buf := new(bytes.Buffer)
		if err := tmpl.Execute(buf, data); err != nil {
			log.Println("template error:", err)
			http.Error(w, "could not render page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Println("writing response:", err)
		}
	}

}

type greeting struct {
	Content string
}

var tmpl = template.Must(template.New("index.html").Parse(`<!DOCTYPE html>
<title>Guestbook - {{.Env}}</title>
<style type="text/css">
html, body {
	font-family: Helvetica, sans-serif;
}
blockquote {
	font-family: cursive, Helvetica, sans-serif;
}
.banner {
	height: 125px;
	width: 250px;
}
.greeting {
	font-size: 85%;
}
.motd {
	font-weight: bold;
}
</style>
<h1>Guestbook</h1>
<div><img class="banner" src="{{.BannerSrc}}"></div>
{{with .MOTD}}<p class="motd">Admin says: {{.}}</p>{{end}}
{{range .Greetings}}
<div class="greeting">
	Someone wrote:
	<blockquote>{{.Content}}</blockquote>
</div>
{{end}}
<form action="/sign" method="POST">
	<div><textarea name="content" rows="3"></textarea></div>
	<div><input type="submit" value="Sign"></div>
</form>
`))

// sign is a form action handler for adding a greeting.
func Sign(app *Application, c *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		content := r.FormValue("content")
		if content == "" {
			http.Error(w, "content must not be empty", http.StatusBadRequest)
			return
		}
		const sqlStmt = "INSERT INTO greetings (content) VALUES (?);"
		_, err := app.Db.ExecContext(r.Context(), sqlStmt, content)
		if err != nil {
			log.Println("sign SQL error:", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
