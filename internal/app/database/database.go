package database

import "time"

// Blob representation in the database.
type Blob struct {
	Digest string
	Pushed time.Time
	Pulled time.Time
}

// Manifest representation in the database.
//
// A manifest is linked to one or more blobs.
type Manifest struct {
	Digest string
	Pushed time.Time
	Pulled time.Time
	Blobs  []Blob
}

// Tag representation in the database.
//
// A tag is linked to one manifest.
type Tag struct {
	Name       string
	Registry   string
	Repository string
	Tag        string
	Manifest   Manifest
	Pushed     time.Time
	Pulled     time.Time
}

// Database operations.
type Database interface {
	CreateSchemaIfNecessary()
	IsBlob(digest string) bool
	PushBlob(blob *Blob)
	PullBlob(blob *Blob)
	DeleteBlob(digest string)
	IsManifest(digest string) bool
	PushManifest(manifest *Manifest)
	PullManifest(manifest *Manifest)
	DeleteManifest(digest string)
	PushTag(tag *Tag)
	PullTag(tag *Tag)
}
