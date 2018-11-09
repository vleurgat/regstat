package mock

import "github.com/vleurgat/regstat/internal/app/database"

// Database: mock implementation of Database
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

func (db Database) CreateSchemaIfNecessary() {
	// no op
}

func (db Database) IsBlob(digest string) bool {
	return db.IsBlobRetValue
}

func (db Database) PushBlob(blob *database.Blob) {
	*db.PushedBlobs = append(*db.PushedBlobs, blob)
}

func (db Database) PullBlob(blob *database.Blob) {
	*db.PulledBlobs = append(*db.PulledBlobs, blob)
}

func (db Database) DeleteBlob(digest string) {
	*db.DeletedBlobs = append(*db.DeletedBlobs, digest)
}

func (db Database) IsManifest(digest string) bool {
	return db.IsManifestRetValue
}

func (db Database) PushManifest(manifest *database.Manifest) {
	*db.PushedManifests = append(*db.PushedManifests, manifest)
}

func (db Database) PullManifest(manifest *database.Manifest) {
	*db.PulledManifests = append(*db.PulledManifests, manifest)
}

func (db Database) DeleteManifest(digest string) {
	*db.DeletedManifests = append(*db.DeletedManifests, digest)
}

func (db Database) PushTag(tag *database.Tag) {
	*db.PushedTags = append(*db.PushedTags, tag)
}

func (db Database) PullTag(tag *database.Tag) {
	*db.PulledTags = append(*db.PulledTags, tag)
}
