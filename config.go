package bittube

import (
	// "errors"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	// "cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"

	"gopkg.in/mgo.v2"

	"github.com/gorilla/sessions"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	DB          VideoDatabase
	OAuthConfig *oauth2.Config

	StorageBucket     *storage.BucketHandle
	StorageBucketName string

	SessionStore sessions.Store

	// PubsubClient *pubsub.Client

	// Force import of mgo library.
	_ mgo.Session
)

// const PubsubTopicID = "fill-book-details"

func init() {
	var err error

	DB, err = configureDatastoreDB("bittube-io")

	if err != nil {
		log.Fatal(err)
	}

	// [START storage]
	// To configure Cloud Storage, uncomment the following lines and update the
	// bucket name.
	//
	StorageBucketName = "bittube-io.appspot.com"
	StorageBucket, err = configureStorage(StorageBucketName)
	// [END storage]

	if err != nil {
		log.Fatal(err)
	}

	// [START auth]
	// To enable user sign-in, uncomment the following lines and update the
	// Client ID and Client Secret.
	// You will also need to update OAUTH2_CALLBACK in app.yaml when pushing to
	// production.
	//
	// OAuthConfig = configureOAuthClient("clientid", "clientsecret")
	// [END auth]

	// [START sessions]
	// Configure storage method for session-wide information.
	// Update "something-very-secret" with a hard to guess string or byte sequence.
	cookieStore := sessions.NewCookieStore([]byte("seCRet$-@Re-leAkeD-s0mEt!me$"))
	cookieStore.Options = &sessions.Options{
		HttpOnly: true,
	}
	SessionStore = cookieStore

	if err != nil {
		log.Fatal(err)
	}
}

func configureDatastoreDB(projectID string) (VideoDatabase, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return newDatastoreDB(client)
}

func configureStorage(bucketID string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucketID), nil
}

func configureOAuthClient(clientID, clientSecret string) *oauth2.Config {
	redirectURL := os.Getenv("OAUTH2_CALLBACK")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth2callback"
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}
