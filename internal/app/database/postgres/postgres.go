package postgres

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/vleurgat/regstat/internal/app/database"
)

// Database is an implementation of database.Database for Postgres.
type Database struct {
	conn *sqlx.DB
}

// CreateDatabase creates a PostgresDatabase which contains a connection to a Postgres database.
func CreateDatabase(pgConnStr string) database.Database {
	conn, err := sqlx.Connect("postgres", pgConnStr)
	if err != nil {
		log.Fatalln(err)
	}
	conn.SetMaxOpenConns(100)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(time.Minute * 5)
	return Database{
		conn: conn,
	}
}

// CreateSchemaIfNecessary does what it says on the tin.
func (db Database) CreateSchemaIfNecessary() {
	var schemaExists bool
	var tableExists bool
	db.conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM information_schema.schemata "+
		"WHERE schema_name = $1"+
		")",
		"regstat").Scan(&schemaExists)
	if schemaExists {
		db.conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM information_schema.tables "+
			"WHERE table_schema = $1 "+
			"AND table_name = $2"+
			")",
			"regstat", "blobs").Scan(&tableExists)
	}
	if !schemaExists || !tableExists {
		log.Println("creating regstat schema")
		db.conn.MustExec(postgresSchema)
	} else {
		log.Println("regstat schema already exists")
	}
}

// IsBlob determines whether the given digest belongs to a persisted blob.
func (db Database) IsBlob(digest string) bool {
	var exists bool
	db.conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM regstat.blobs "+
		"WHERE digest = $1"+
		")",
		digest).Scan(&exists)
	return exists
}

// PushBlob writes a blob to the database, or updates the pushed time of an existing one.
func (db Database) PushBlob(blob *database.Blob) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.blobs "+
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
func (db Database) PullBlob(blob *database.Blob) {
	tx := db.conn.MustBegin()
	pullBlob(blob, tx)
	tx.Commit()
	log.Println("pull blob", blob.Digest)
}

func pullBlob(blob *database.Blob, tx *sqlx.Tx) {
	tx.MustExec("INSERT INTO regstat.blobs "+
		"(digest, pushed, pulled) "+
		"VALUES ($1, $2, $3) "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"pulled = $3",
		blob.Digest, blob.Pushed, blob.Pulled)
}

// DeleteBlob deletes a blob from the database, moving the existing entry to the deleted_blobs table.
func (db Database) DeleteBlob(digest string) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.deleted_blobs "+
		"SELECT digest, pushed, pulled, NOW() FROM regstat.blobs "+
		"WHERE digest = $1 "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"deleted = NOW()",
		digest)
	tx.MustExec("INSERT INTO regstat.deleted_manifest_blob "+
		"SELECT manifest_digest, blob_digest FROM regstat.manifest_blob "+
		"WHERE blob_digest = $1 "+
		"ON CONFLICT (manifest_digest, blob_digest) "+
		"DO NOTHING",
		digest)
	tx.MustExec("DELETE FROM regstat.manifest_blob "+
		"WHERE blob_digest = $1",
		digest)
	tx.MustExec("DELETE FROM regstat.blobs "+
		"WHERE digest = $1",
		digest)
	tx.Commit()
	log.Println("delete blob", digest)
}

// IsManifest determines whether the given digest belongs to a persisted manifest.
func (db Database) IsManifest(digest string) bool {
	var exists bool
	db.conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM regstat.manifests "+
		"WHERE digest = $1"+
		")",
		digest).Scan(&exists)
	return exists
}

// PushManifest writes a manifest to the database, or updates the pushed time of an existing one.
func (db Database) PushManifest(manifest *database.Manifest) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.manifests "+
		"(digest, pushed)"+
		"VALUES ($1, $2) "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"pushed = $2",
		manifest.Digest, manifest.Pushed)
	for _, blob := range manifest.Blobs {
		pullBlob(&blob, tx)
		tx.MustExec("INSERT INTO regstat.manifest_blob "+
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
func (db Database) PullManifest(manifest *database.Manifest) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.manifests "+
		"(digest, pushed, pulled)"+
		"VALUES ($1, $2, $3) "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"pulled = $3",
		manifest.Digest, manifest.Pushed, manifest.Pulled)
	tx.MustExec("UPDATE regstat.blobs b "+
		"SET pulled = $1 "+
		"FROM regstat.manifest_blob mb "+
		"WHERE b.digest = mb.blob_digest AND mb.manifest_digest = $2",
		manifest.Pulled, manifest.Digest)
	tx.Commit()
	log.Println("pull manifest", manifest.Digest)
}

// DeleteManifest deletes a manifest and associated tag from the database, moving the
// existing entries to the deleted_manifests and deleted_tags tables.
func (db Database) DeleteManifest(digest string) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.deleted_manifests "+
		"SELECT digest, pushed, pulled, NOW() FROM regstat.manifests "+
		"WHERE digest = $1 "+
		"ON CONFLICT (digest) "+
		"DO UPDATE SET "+
		"deleted = NOW()",
		digest)
	tx.MustExec("INSERT INTO regstat.deleted_tags "+
		"SELECT name, registry, repository, tag, manifest_digest, pushed, pulled, NOW() FROM regstat.tags "+
		"WHERE manifest_digest = $1 "+
		"ON CONFLICT (name) "+
		"DO UPDATE SET "+
		"deleted = NOW()",
		digest)
	tx.MustExec("INSERT INTO regstat.deleted_manifest_blob "+
		"SELECT manifest_digest, blob_digest FROM regstat.manifest_blob "+
		"WHERE manifest_digest = $1 "+
		"ON CONFLICT (manifest_digest, blob_digest) "+
		"DO NOTHING",
		digest)
	tx.MustExec("DELETE FROM regstat.tags "+
		"WHERE manifest_digest = $1",
		digest)
	tx.MustExec("DELETE FROM regstat.manifest_blob "+
		"WHERE manifest_digest = $1",
		digest)
	tx.MustExec("DELETE FROM regstat.manifests "+
		"WHERE digest = $1",
		digest)
	tx.Commit()
	log.Println("delete manifest", digest)
}

// PushTag writes a tag to the database, or updates the pushed time of an existing one.
func (db Database) PushTag(tag *database.Tag) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.tags "+
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
func (db Database) PullTag(tag *database.Tag) {
	tx := db.conn.MustBegin()
	tx.MustExec("INSERT INTO regstat.tags "+
		"(name, registry, repository, tag, manifest_digest, pushed, pulled) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7) "+
		"ON CONFLICT (name) "+
		"DO UPDATE SET "+
		"pulled = $7",
		tag.Name, tag.Registry, tag.Repository, tag.Tag, tag.Manifest.Digest, tag.Pushed, tag.Pulled)
	tx.Commit()
	log.Println("pull tag", tag.Name)
}
