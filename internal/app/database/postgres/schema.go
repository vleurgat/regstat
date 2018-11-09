package postgres

var postgresSchema = `
CREATE SCHEMA IF NOT EXISTS regstat;

CREATE TABLE IF NOT EXISTS regstat.blobs  (
	digest	text NOT NULL,
	pushed	timestamp NOT NULL,
	pulled	timestamp NULL,
	PRIMARY KEY(digest)
);

CREATE TABLE IF NOT EXISTS regstat.deleted_blobs  (
	digest 	text NOT NULL,
	pushed 	timestamp NOT NULL,
	pulled 	timestamp NULL,
	deleted	timestamp NOT NULL,
	PRIMARY KEY(digest)
);

CREATE TABLE IF NOT EXISTS regstat.deleted_manifest_blob  (
	manifest_digest	text NOT NULL,
	blob_digest    	text NOT NULL,
	PRIMARY KEY(manifest_digest,blob_digest)
);

CREATE TABLE IF NOT EXISTS regstat.deleted_manifests  (
	digest 	text NOT NULL,
	pushed 	timestamp NOT NULL,
	pulled 	timestamp NULL,
	deleted	timestamp NOT NULL,
	PRIMARY KEY(digest)
);

CREATE TABLE IF NOT EXISTS regstat.deleted_tags  (
	name           	text NOT NULL,
	registry       	text NOT NULL,
	repository     	text NOT NULL,
	tag            	text NULL,
	manifest_digest	text NOT NULL,
	pushed         	timestamp NOT NULL,
	pulled         	timestamp NULL,
	deleted        	timestamp NOT NULL,
	PRIMARY KEY(name)
);

CREATE TABLE IF NOT EXISTS regstat.manifest_blob  (
	manifest_digest	text NOT NULL,
	blob_digest    	text NOT NULL,
	PRIMARY KEY(manifest_digest,blob_digest)
);

CREATE TABLE IF NOT EXISTS regstat.manifests  (
	digest	text NOT NULL,
	pushed	timestamp NOT NULL,
	pulled	timestamp NULL,
	PRIMARY KEY(digest)
);

CREATE TABLE IF NOT EXISTS regstat.tags  (
	name           	text NOT NULL,
	registry       	text NOT NULL,
	repository     	text NOT NULL,
	tag            	text NULL,
	manifest_digest	text NOT NULL,
	pushed         	timestamp NOT NULL,
	pulled         	timestamp NULL,
	PRIMARY KEY(name)
);

CREATE INDEX IF NOT EXISTS blob_digest
	ON regstat.manifest_blob USING btree (blob_digest);

CREATE INDEX IF NOT EXISTS deleted_blob_digest
	ON regstat.deleted_manifest_blob USING btree (blob_digest);

ALTER TABLE regstat.manifest_blob
  DROP CONSTRAINT IF EXISTS manifests_fkey;

ALTER TABLE regstat.manifest_blob
	ADD CONSTRAINT manifests_fkey
	FOREIGN KEY(manifest_digest)
	REFERENCES regstat.manifests(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION;

ALTER TABLE regstat.manifest_blob
  DROP CONSTRAINT IF EXISTS blobs_fkey;

ALTER TABLE regstat.manifest_blob
	ADD CONSTRAINT blobs_fkey
	FOREIGN KEY(blob_digest)
	REFERENCES regstat.blobs(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION;

ALTER TABLE regstat.tags
  DROP CONSTRAINT IF EXISTS manifests_fkey;

ALTER TABLE regstat.tags
	ADD CONSTRAINT manifests_fkey
	FOREIGN KEY(manifest_digest)
	REFERENCES regstat.manifests(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION;
`
