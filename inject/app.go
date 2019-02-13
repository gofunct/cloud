package inject

import (
	"cloud.google.com/go/firestore"
	"fmt"
	"github.com/google/wire"
	"github.com/gorilla/mux"
	"go.opencensus.io/trace"
	"gocloud.dev/blob"
	"gocloud.dev/health"
	"gocloud.dev/server"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	// TODO: randomize it
	oauthStateString = "pseudo-random"
	googleOauthConfig *oauth2.Config
)

func init() {
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  Configuration.Redirect,
		ClientID:    Configuration.ClientId,
		ClientSecret: Configuration.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

var Configuration = &Config{}

type Config struct {
	Redirect string 		`json:"redirect"`
	ClientId string 		`json:"clientid"`
	ClientSecret string 	`json:"clientsecret"`
	Project   string 		 `json:"project"`
	Bucket     string        `json:"bucket"`
	Endpoint oauth2.Endpoint
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
	Fire 	*firestore.Client
	Bucket *blob.Bucket

}

// of the day variable.
func NewApplication(srv *server.Server, bucket *blob.Bucket, fire *firestore.Client) *Application {
	app := &Application{
		Server: srv,
		Fire:     fire,
		Bucket: bucket,
	}
	return app
}

// appHealthChecks returns a health check for the database. This will signal
// to Kubernetes or other orchestrators that the server should not receive
// traffic until the server is able to connect to its database.
func AppHealthChecks() ([]health.Checker, func()) {
	list := []health.Checker{nil}
	return list, func() {
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


func (app *Application) Index(w http.ResponseWriter, r *http.Request) {
	var htmlIndex = `<html>
<body>
	<a href="/login">Google Log In</a>
</body>
</html>`
	fmt.Fprintf(w, htmlIndex)
}


func (app *Application) Login(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (app *Application) CallBack(w http.ResponseWriter, r *http.Request) {
	content, err := app.getUserInfo(r.FormValue("state"), r.FormValue("code"))
	if err != nil {
		fmt.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	fmt.Fprintf(w, "Content: %s\n", content)
}

func (app *Application) getUserInfo(state string, code string) ([]byte, error) {
	if state != oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}
	return contents, nil
}