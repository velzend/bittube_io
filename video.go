package bittube

// Video holds metadata about the video.
type Video struct {
	ID            int64
	Title         string
	Author        string
	PublishedDate string
	VideoURL      string
	Description   string
	CreatedBy     string
	CreatedByID   string
}

// CreatedByDisplayName returns a string appropriate for displaying the name of
// the user who created this video object.
func (b *Video) CreatedByDisplayName() string {
	if b.CreatedByID == "anonymous" {
		return "Anonymous"
	}
	return b.CreatedBy
}

// SetCreatorAnonymous sets the CreatedByID field to the "anonymous" ID.
func (b *Video) SetCreatorAnonymous() {
	b.CreatedBy = ""
	b.CreatedByID = "anonymous"
}

// VideoDatabase provides thread-safe access to a database of movies.
type VideoDatabase interface {
	// ListVideos returns a list of videos, ordered by title.
	ListVideos() ([]*Video, error)

	// ListVideosCreatedBy returns a list of videos, ordered by title, filtered by
	// the user who created the video entry.
	ListVideosCreatedBy(userID string) ([]*Video, error)

	// GetVideo retrieves a video by its ID.
	GetVideo(id int64) (*Video, error)

	// AddVideo saves a given video, assigning it a new ID.
	AddVideo(b *Video) (id int64, err error)

	// Close closes the database, freeing up any available resources.
	Close()
}
