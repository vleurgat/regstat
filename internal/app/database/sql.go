package database

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// Database connection details - must be a Postgres database.
type Database struct {
	PgConnStr string
	conn      *sqlx.DB
}

// ConnectToDb makes a connection to a Postgres database.
func (db *Database) ConnectToDb() {
	conn, err := sqlx.Connect("postgres", db.PgConnStr)
	if err != nil {
		log.Fatalln(err)
	}
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(time.Minute * 5)
	db.conn = conn
}

// IsBlob determines whether the given digest belongs to a persisted blob.
func (db *Database) IsBlob(digest string) bool {
	var exists bool
	db.conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM blobs "+
		"WHERE digest = $1"+
		")",
		digest).Scan(&exists)
	return exists
}

// PushBlob writes a blob to the database, or updates the pushed time of an existing one.
func (db *Database) PushBlob(blob *Blob) {
	tx := db.conn.MustBegin()
	tx.MustExec(
		"INSERT INTO blobs "+
			"(digest, pushed) "+
			"VALUES ($1, $2) "+
			"ON CONFLICT (digest) "+
			"DO UPDATE SET "+
			"pushed = $2",
		blob.Digest, blob.Pushed)
	tx.Commit()
	log.Println("push blob", blob.Digest)
}

// PullBlob writes a blob to the database, or updates the pulled time of an existing one.
func (db *Database) PullBlob(blob *Blob) {
	tx := db.conn.MustBegin()
	db.pullBlob(blob, tx)
	tx.Commit()
	log.Println("pull blob", blob.Digest)
}

func (db *Database) pullBlob(blob *Blob, tx *sqlx.Tx) {
	tx.MustExec(
		"INSERT INTO blobs "+
			"(digest, pushed, pulled) "+
			"VALUES ($1, $2, $3) "+
			"ON CONFLICT (digest) "+
			"DO UPDATE SET "+
			"pulled = $3",
		blob.Digest, blob.Pushed, blob.Pulled)
}

// DeleteBlob deletes a blob from the database, moving the existing entry to the deleted_blobs table.
func (db *Database) DeleteBlob(digest string) {
	tx := db.conn.MustBegin()
	tx.MustExec(
		"INSERT INTO deleted_blobs "+
			"SELECT digest, pushed, pulled, NOW() FROM blobs "+
			"WHERE digest = $1 "+
			"ON CONFLICT (digest) "+
			"DO UPDATE SET "+
			"deleted = NOW()",
		digest)
	tx.MustExec(
		"INSERT INTO deleted_manifest_blob "+
			"SELECT manifest_digest, blob_digest FROM manifest_blob "+
			"WHERE blob_digest = $1 "+
			"ON CONFLICT (manifest_digest, blob_digest) "+
			"DO NOTHING",
		digest)
	tx.MustExec(
		"DELETE FROM manifest_blob "+
			"WHERE blob_digest = $1",
		digest)
	tx.MustExec(
		"DELETE FROM blobs "+
			"WHERE digest = $1",
		digest)
	tx.Commit()
	log.Println("delete blob", digest)
}

// IsManifest determines whether the given digest belongs to a persisted manifest.
func (db *Database) IsManifest(digest string) bool {
	var exists bool
	db.conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM manifests "+
		"WHERE digest = $1"+
		")",
		digest).Scan(&exists)
	return exists
}

// PushManifest writes a manifest to the database, or updates the pushed time of an existing one.
func (db *Database) PushManifest(manifest *Manifest) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO manifests "+
		"(digest, pushed)"+
		"VALUES ($1, $2) "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"pushed = $2",
		manifest.Digest, manifest.Pushed)
	for _, blob := range manifest.Blobs {
		db.pullBlob(&blob, tx)
		tx.MustExec("INSERT INTO manifest_blob "+
			"(manifest_digest, blob_digest)"+
			"VALUES ($1, $2) "+
			"ON CONFLICT (manifest_digest, blob_digest) "+
			"DO NOTHING",
			manifest.Digest, blob.Digest)
	}
	tx.Commit()
	log.Println("push manifest", manifest.Digest, len(manifest.Blobs))
}

// PullManifest writes a manifest to the database, or updates the pulled time of an existing one.
func (db *Database) PullManifest(manifest *Manifest) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO manifests "+
		"(digest, pushed, pulled)"+
		"VALUES ($1, $2, $3) "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"pulled = $3",
		manifest.Digest, manifest.Pushed, manifest.Pulled)
	tx.MustExec("UPDATE blobs b "+
		"SET pulled = $1 "+
		"FROM manifest_blob mb "+
		"WHERE b.digest = mb.blob_digest AND mb.manifest_digest = $2",
		manifest.Pulled, manifest.Digest)
	tx.Commit()
	log.Println("pull manifest", manifest.Digest)
}

// DeleteManifest deletes a manifest and associated tag from the database, moving the
// existing entries to the deleted_manifests and deleted_tags tables.
func (db *Database) DeleteManifest(digest string) {
	tx := db.conn.MustBegin()
	tx.MustExec(
		"INSERT INTO deleted_manifests "+
			"SELECT digest, pushed, pulled, NOW() FROM manifests "+
			"WHERE digest = $1 "+
			"ON CONFLICT (digest) "+
			"DO UPDATE SET "+
			"deleted = NOW()",
		digest)
	tx.MustExec(
		"INSERT INTO deleted_tags "+
			"SELECT name, registry, repository, tag, manifest_digest, pushed, pulled, NOW() FROM tags "+
			"WHERE manifest_digest = $1 "+
			"ON CONFLICT (name) "+
			"DO UPDATE SET "+
			"deleted = NOW()",
		digest)
	tx.MustExec(
		"INSERT INTO deleted_manifest_blob "+
			"SELECT manifest_digest, blob_digest FROM manifest_blob "+
			"WHERE manifest_digest = $1 "+
			"ON CONFLICT (manifest_digest, blob_digest) "+
			"DO NOTHING",
		digest)
	tx.MustExec(
		"DELETE FROM tags "+
			"WHERE manifest_digest = $1",
		digest)
	tx.MustExec(
		"DELETE FROM manifest_blob "+
			"WHERE manifest_digest = $1",
		digest)
	tx.MustExec(
		"DELETE FROM manifests "+
			"WHERE digest = $1",
		digest)
	tx.Commit()
	log.Println("delete manifest", digest)
}

// PushTag writes a tag to the database, or updates the pushed time of an existing one.
func (db *Database) PushTag(tag *Tag) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO tags "+
		"(name, registry, repository, tag, manifest_digest, pushed) "+
		"VALUES ($1, $2, $3, $4, $5, $6) "+
		"ON CONFLICT (name) "+
		"DO UPDATE SET "+
		"manifest_digest = $5, "+
		"pushed = $6",
		tag.Name, tag.Registry, tag.Repository, tag.Tag, tag.Manifest.Digest, tag.Pushed)
	tx.Commit()
	log.Println("push tag", tag.Name)
}

// PullTag writes a tag to the database, or updates the pulled time of an existing one.
func (db *Database) PullTag(tag *Tag) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO tags "+
		"(name, registry, repository, tag, manifest_digest, pushed, pulled) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7) "+
		"ON CONFLICT (name) "+
		"DO UPDATE SET "+
		"pulled = $7",
		tag.Name, tag.Registry, tag.Repository, tag.Tag, tag.Manifest.Digest, tag.Pushed, tag.Pulled)
	tx.Commit()
	log.Println("pull tag", tag.Name)
}
