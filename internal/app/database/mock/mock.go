package mock

import (
	"github.com/jmoiron/sqlx"
	"github.com/vleurgat/regstat/internal/app/database"
)

// Database is a mock implementation of database.Database
type Database struct {
	IsBlobRetValue     bool
	IsManifestRetValue bool
	PushedBlobs        *[]*database.Blob
	PushedManifests    *[]*database.Manifest
	PushedTags         *[]*database.Tag
	PulledBlobs        *[]*database.Blob
	PulledManifests    *[]*database.Manifest
	PulledTags         *[]*database.Tag
	DeletedBlobs       *[]string
	DeletedManifests   *[]string
}

// CreateDatabase creates a mock Database implementation
func CreateDatabase() Database {
	return Database{
		PushedBlobs:      &[]*database.Blob{},
		PushedManifests:  &[]*database.Manifest{},
		PushedTags:       &[]*database.Tag{},
		PulledBlobs:      &[]*database.Blob{},
		PulledManifests:  &[]*database.Manifest{},
		PulledTags:       &[]*database.Tag{},
		DeletedBlobs:     &[]string{},
		DeletedManifests: &[]string{},
	}
}

// GetConnection always returns nil.
func (db Database) GetConnection() *sqlx.DB {
	return nil
}

// CreateSchemaIfNecessary does what it says on the tin.
func (db Database) CreateSchemaIfNecessary() {
	// no op
}

// IsBlob determines whether the given digest belongs to a persisted blob.
func (db Database) IsBlob(digest string) bool {
	return db.IsBlobRetValue
}

// PushBlob writes a blob to the database, or updates the pushed time of an existing one.
func (db Database) PushBlob(blob *database.Blob) {
	*db.PushedBlobs = append(*db.PushedBlobs, blob)
}

// PullBlob writes a blob to the database, or updates the pulled time of an existing one.
func (db Database) PullBlob(blob *database.Blob) {
	*db.PulledBlobs = append(*db.PulledBlobs, blob)
}

// DeleteBlob deletes a blob from the database, moving the existing entry to the deleted_blobs table.
func (db Database) DeleteBlob(digest string) {
	*db.DeletedBlobs = append(*db.DeletedBlobs, digest)
}

// IsManifest determines whether the given digest belongs to a persisted manifest.
func (db Database) IsManifest(digest string) bool {
	return db.IsManifestRetValue
}

// PushManifest writes a manifest to the database, or updates the pushed time of an existing one.
func (db Database) PushManifest(manifest *database.Manifest) {
	*db.PushedManifests = append(*db.PushedManifests, manifest)
}

// PullManifest writes a manifest to the database, or updates the pulled time of an existing one.
func (db Database) PullManifest(manifest *database.Manifest) {
	*db.PulledManifests = append(*db.PulledManifests, manifest)
}

// DeleteManifest deletes a manifest and associated tag from the database, moving the
// existing entries to the deleted_manifests and deleted_tags tables.
func (db Database) DeleteManifest(digest string) {
	*db.DeletedManifests = append(*db.DeletedManifests, digest)
}

// PushTag writes a tag to the database, or updates the pushed time of an existing one.
func (db Database) PushTag(tag *database.Tag) {
	*db.PushedTags = append(*db.PushedTags, tag)
}

// PullTag writes a tag to the database, or updates the pulled time of an existing one.
func (db Database) PullTag(tag *database.Tag) {
	*db.PulledTags = append(*db.PulledTags, tag)
}
