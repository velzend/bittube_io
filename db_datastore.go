package bittube

import (
"fmt"

"cloud.google.com/go/datastore"

"golang.org/x/net/context"
)

// datastoreDB persists videos to Cloud Datastore.
// https://cloud.google.com/datastore/docs/concepts/overview
type datastoreDB struct {
	client *datastore.Client
}

// Ensure datastoreDB conforms to the VideoDatabase interface.
var _ VideoDatabase = &datastoreDB{}

// newDatastoreDB creates a new VideoDatabase backed by Cloud Datastore.
// See the datastore and google packages for details on creating a suitable Client:
// https://godoc.org/cloud.google.com/go/datastore
func newDatastoreDB(client *datastore.Client) (VideoDatabase, error) {
	ctx := context.Background()
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := client.NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	return &datastoreDB{
		client: client,
	}, nil
}

// Close closes the database.
func (db *datastoreDB) Close() {
	// No op.
}

func (db *datastoreDB) datastoreKey(id int64) *datastore.Key {
	return datastore.IDKey("Video", id, nil)
}

// GetVideo retrieves a video by its ID.
func (db *datastoreDB) GetVideo(id int64) (*Video, error) {
	ctx := context.Background()
	k := db.datastoreKey(id)
	video := &Video{}
	if err := db.client.Get(ctx, k, video); err != nil {
		return nil, fmt.Errorf("datastoredb: could not get Video: %v", err)
	}
	video.ID = id
	return video, nil
}

// AddVideo saves a given video, assigning it a new ID.
func (db *datastoreDB) AddVideo(b *Video) (id int64, err error) {
	ctx := context.Background()
	k := datastore.IncompleteKey("Video", nil)
	k, err = db.client.Put(ctx, k, b)
	if err != nil {
		return 0, fmt.Errorf("datastoredb: could not put Video: %v", err)
	}
	return k.ID, nil
}

// DeleteVideo removes a given video by its ID.
func (db *datastoreDB) DeleteVideo(id int64) error {
	ctx := context.Background()
	k := db.datastoreKey(id)
	if err := db.client.Delete(ctx, k); err != nil {
		return fmt.Errorf("datastoredb: could not delete Video: %v", err)
	}
	return nil
}

// UpdateVideo updates the entry for a given video.
func (db *datastoreDB) UpdateVideo(b *Video) error {
	ctx := context.Background()
	k := db.datastoreKey(b.ID)
	if _, err := db.client.Put(ctx, k, b); err != nil {
		return fmt.Errorf("datastoredb: could not update Video: %v", err)
	}
	return nil
}

// ListVideos returns a list of videos, ordered by title.
func (db *datastoreDB) ListVideos() ([]*Video, error) {
	ctx := context.Background()
	videos := make([]*Video, 0)
	q := datastore.NewQuery("Video").
		Order("Title")

	keys, err := db.client.GetAll(ctx, q, &videos)

	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not list videos: %v", err)
	}

	for i, k := range keys {
		videos[i].ID = k.ID
	}

	return videos, nil
}

// ListVideosCreatedBy returns a list of videos, ordered by title, filtered by
// the user who created the video entry.
func (db *datastoreDB) ListVideosCreatedBy(userID string) ([]*Video, error) {
	ctx := context.Background()
	if userID == "" {
		return db.ListVideos()
	}

	videos := make([]*Video, 0)
	q := datastore.NewQuery("Video").
		Filter("CreatedByID =", userID).
		Order("Title")

	keys, err := db.client.GetAll(ctx, q, &videos)

	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not list videos: %v", err)
	}

	for i, k := range keys {
		videos[i].ID = k.ID
	}

	return videos, nil
}

