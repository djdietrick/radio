package models

import "time"

// Track is a single audio file in the library.
type Track struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	ArtistID    string    `json:"artistId"`
	ArtistName  string    `json:"artistName"`
	AlbumID     string    `json:"albumId"`
	AlbumTitle  string    `json:"albumTitle"`
	TrackNumber int       `json:"trackNumber"`
	DiscNumber  int       `json:"discNumber"`
	DurationMs  int64     `json:"durationMs"`
	Genre       string    `json:"genre"`
	Year        int       `json:"year"`
	// Codec/format info used to decide whether the browser can play directly.
	Codec      string    `json:"codec"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	ModifiedAt time.Time `json:"modifiedAt"`
	AddedAt    time.Time `json:"addedAt"`
}

// Album groups tracks. AlbumArtist is the album-level artist (may differ from
// per-track artists for compilations).
type Album struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	AlbumArtist string    `json:"albumArtist"`
	Year        int       `json:"year"`
	TrackCount  int       `json:"trackCount"`
	DurationMs  int64     `json:"durationMs"`
	HasArt      bool      `json:"hasArt"`
	AddedAt     time.Time `json:"addedAt"`
}

// Artist is a distinct performing artist.
type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	TrackCount int    `json:"trackCount"`
}

// PlaylistItemKind distinguishes a single-track entry from a whole-album entry.
// This is the core of the album-aware shuffle: in album-shuffle mode an album
// item plays its tracks in order as one indivisible unit.
type PlaylistItemKind string

const (
	ItemKindTrack PlaylistItemKind = "track"
	ItemKindAlbum PlaylistItemKind = "album"
)

// PlaylistItem is one ordered entry in a playlist. Exactly one of TrackID or
// AlbumID is set, per Kind.
type PlaylistItem struct {
	ID       string           `json:"id"`
	Kind     PlaylistItemKind `json:"kind"`
	Position int              `json:"position"`
	TrackID  string           `json:"trackId,omitempty"`
	AlbumID  string           `json:"albumId,omitempty"`
}

// Playlist is an ordered list of typed items owned by a user.
type Playlist struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	Name      string         `json:"name"`
	Items     []PlaylistItem `json:"items,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// Station is a "radio station": a playlist that began playing at StartedAt and
// is treated as continuously playing. The current position is computed, not
// stored. ShuffleSeed makes the (shuffled) ordering deterministic and shared
// across all listeners.
type Station struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	PlaylistID  string    `json:"playlistId"`
	StartedAt   time.Time `json:"startedAt"`
	AlbumShuffle bool     `json:"albumShuffle"`
	ShuffleSeed int64     `json:"shuffleSeed"`
	// Loop controls behaviour once the resolved tracklist is exhausted.
	Loop      bool      `json:"loop"`
	CreatedAt time.Time `json:"createdAt"`
}

// NowPlaying is the computed live state of a station at a given instant.
type NowPlaying struct {
	StationID    string `json:"stationId"`
	Track        Track  `json:"track"`
	// OffsetMs is how far into the current track the station is right now.
	OffsetMs     int64  `json:"offsetMs"`
	// IndexInQueue is the position of the current track in the resolved queue.
	IndexInQueue int    `json:"indexInQueue"`
	QueueLength  int    `json:"queueLength"`
	// ServerTime is the instant the computation was made (for client clock sync).
	ServerTime   time.Time `json:"serverTime"`
	// MsUntilNextTrack is how long, from ServerTime, until the current track
	// ends and the station advances. 0 for an ended (non-looping) station.
	// The live-sync layer uses this to schedule the next push.
	MsUntilNextTrack int64 `json:"msUntilNextTrack"`
	// Ended is true for non-looping stations whose program has finished.
	Ended        bool   `json:"ended"`
}
