package main

import (
	// "encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"

	"google.golang.org/appengine"
	"bittube.io"

)

var (
	// See template.go
	listTmpl   = parseTemplate("list.html")
	editTmpl   = parseTemplate("edit.html")
	detailTmpl = parseTemplate("detail.html")

)

func main() {
	registerHandlers()
	appengine.Main()
}

func registerHandlers() {
	// Use gorilla/mux for rich routing.
	// See http://www.gorillatoolkit.org/pkg/mux
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/videos", http.StatusFound))

	r.Methods("GET").Path("/videos").
		Handler(appHandler(listHandler))
	//r.Methods("GET").Path("/videos/mine").
	//	Handler(appHandler(listMineHandler))
	r.Methods("GET").Path("/videos/{id:[0-9]+}").
		Handler(appHandler(detailHandler))
	r.Methods("GET").Path("/videos/add").
		Handler(appHandler(addFormHandler))
	//r.Methods("GET").Path("/videos/{id:[0-9]+}/edit").
	//	Handler(appHandler(editFormHandler))

	r.Methods("POST").Path("/videos").
		Handler(appHandler(createHandler))
	//r.Methods("POST", "PUT").Path("/videos/{id:[0-9]+}").
	//	Handler(appHandler(updateHandler))
	//r.Methods("POST").Path("/videos/{id:[0-9]+}:delete").
	//	Handler(appHandler(deleteHandler)).Name("delete")
	//
	// The following handlers are defined in auth.go and used in the
	// "Authenticating Users" part of the Getting Started guide.
	//r.Methods("GET").Path("/login").
	//	Handler(appHandler(loginHandler))
	//r.Methods("POST").Path("/logout").
	//	Handler(appHandler(logoutHandler))
	//r.Methods("GET").Path("/oauth2callback").
	//	Handler(appHandler(oauthCallbackHandler))

	r.Methods("GET").PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	// Respond to App Engine and Compute Engine health checks.
	// Indicate the server is healthy.
	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	// [START request_logging]
	// Delegate all of the HTTP routing and serving to the gorilla/mux router.
	// Log all requests using the standard Apache format.
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
	// [END request_logging]
}

// listHandler displays a list with summaries of videos in the database.
func listHandler(w http.ResponseWriter, r *http.Request) *appError {
	videos, err := bittube.DB.ListVideos()
	if err != nil {
		return appErrorf(err, "could not list videos: %v", err)
	}

	return listTmpl.Execute(w, r, videos)
}

// listMineHandler displays a list of videos created by the currently
// authenticated user.
/*func listMineHandler(w http.ResponseWriter, r *http.Request) *appError {
	user := profileFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/videos/mine", http.StatusFound)
		return nil
	}

	videos, err := bittube.DB.ListVideosCreatebittube.DBy(user.ID)
	if err != nil {
		return appErrorf(err, "could not list videos: %v", err)
	}

	return listTmpl.Execute(w, r, videos)
}*/

// videoFromRequest retrieves a video from the database given a video ID in the
// URL's path.
func videoFromRequest(r *http.Request) (*bittube.Video, error) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad video id: %v", err)
	}
	video, err := bittube.DB.GetVideo(id)
	if err != nil {
		return nil, fmt.Errorf("could not find video: %v", err)
	}
	return video, nil
}

// detailHandler displays the details of a given video.
func detailHandler(w http.ResponseWriter, r *http.Request) *appError {
	video, err := videoFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}

	return detailTmpl.Execute(w, r, video)
}

// addFormHandler displays a form that captures details of a new video to add to
// the database.
func addFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	return editTmpl.Execute(w, r, nil)
}

// editFormHandler displays a form that allows the user to edit the details of
// a given video.
func editFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	video, err := videoFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}

	return editTmpl.Execute(w, r, video)
}

// videoFromForm populates the fields of a Video from form values
// (see templates/edit.html).
func videoFromForm(r *http.Request) (*bittube.Video, error) {
	videoURL, err := uploadFileFromForm(r)
	if err != nil {
		return nil, fmt.Errorf("could not upload file: %v", err)
	}
	if videoURL == "" {
		videoURL = r.FormValue("videoURL")
	}

	video := &bittube.Video{
		Title:         r.FormValue("title"),
		Author:        r.FormValue("author"),
		PublishedDate: r.FormValue("publishedDate"),
		VideoURL:      videoURL,
		Description:   r.FormValue("description"),
		CreatedBy:     r.FormValue("createbittube.DBy"),
		CreatedByID:   r.FormValue("createbittube.DByID"),
	}

	// If the form didn't carry the user information for the creator, populate it
	// from the currently logged in user (or mark as anonymous).
	if video.CreatedByID == "" {
		user := profileFromSession(r)
		if user != nil {
			// Logged in.
			video.CreatedBy = user.DisplayName
			video.CreatedByID = user.ID
		} else {
			// Not logged in.
			video.SetCreatorAnonymous()
		}
	}

	return video, nil
}

// uploadFileFromForm uploads a file if it's present in the "video" form field.
func uploadFileFromForm(r *http.Request) (url string, err error) {
	f, fh, err := r.FormFile("video")
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if bittube.StorageBucket == nil {
		return "", errors.New("storage bucket is missing - check config.go")
	}

	// random filename, retaining existing extension.
	name := uuid.NewV4().String() + path.Ext(fh.Filename)

	ctx := context.Background()
	w := bittube.StorageBucket.Object(name).NewWriter(ctx)
	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	w.ContentType = fh.Header.Get("Content-Type")

	// Entries are immutable, be aggressive about caching (1 day).
	w.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(w, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	const publicURL = "https://storage.googleapis.com/%s/%s"
	return fmt.Sprintf(publicURL, bittube.StorageBucketName, name), nil
}

// createHandler adds a video to the database.
func createHandler(w http.ResponseWriter, r *http.Request) *appError {
	video, err := videoFromForm(r)
	if err != nil {
		return appErrorf(err, "could not parse video from form: %v", err)
	}
	id, err := bittube.DB.AddVideo(video)
	if err != nil {
		return appErrorf(err, "could not save video: %v", err)
	}
	// go publishUpdate(id)
	http.Redirect(w, r, fmt.Sprintf("/videos/%d", id), http.StatusFound)
	return nil
}

// http://blog.golang.org/error-handling-and-go
type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error.
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)

		http.Error(w, e.Message, e.Code)
	}
}

func appErrorf(err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
	}
}
