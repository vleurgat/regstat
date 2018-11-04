ALTER TABLE public.tags
	DROP CONSTRAINT manifests_fkey CASCADE
;
ALTER TABLE public.manifest_blob
	DROP CONSTRAINT manifests_fkey CASCADE
;
ALTER TABLE public.manifest_blob
	DROP CONSTRAINT blobs_fkey CASCADE
;
DROP INDEX public.deleted_blob_digest
;
DROP INDEX public.blob_digest
;
DROP TABLE public.tags
;
DROP TABLE public.manifests
;
DROP TABLE public.manifest_blob
;
DROP TABLE public.deleted_tags
;
DROP TABLE public.deleted_manifests
;
DROP TABLE public.deleted_manifest_blob
;
DROP TABLE public.deleted_blobs
;
DROP TABLE public.blobs
;

CREATE TABLE public.blobs  (
	digest	text NOT NULL,
	pushed	timestamp NOT NULL,
	pulled	timestamp NULL,
	PRIMARY KEY(digest)
)
;
CREATE TABLE public.deleted_blobs  (
	digest 	text NOT NULL,
	pushed 	timestamp NOT NULL,
	pulled 	timestamp NULL,
	deleted	timestamp NOT NULL,
	PRIMARY KEY(digest)
)
;
CREATE TABLE public.deleted_manifest_blob  (
	manifest_digest	text NOT NULL,
	blob_digest    	text NOT NULL,
	PRIMARY KEY(manifest_digest,blob_digest)
)
;
CREATE TABLE public.deleted_manifests  (
	digest 	text NOT NULL,
	pushed 	timestamp NOT NULL,
	pulled 	timestamp NULL,
	deleted	timestamp NOT NULL,
	PRIMARY KEY(digest)
)
;
CREATE TABLE public.deleted_tags  (
	name           	text NOT NULL,
	registry       	text NOT NULL,
	repository     	text NOT NULL,
	tag            	text NULL,
	manifest_digest	text NOT NULL,
	pushed         	timestamp NOT NULL,
	pulled         	timestamp NULL,
	deleted        	timestamp NOT NULL,
	PRIMARY KEY(name)
)
;
CREATE TABLE public.manifest_blob  (
	manifest_digest	text NOT NULL,
	blob_digest    	text NOT NULL,
	PRIMARY KEY(manifest_digest,blob_digest)
)
;
CREATE TABLE public.manifests  (
	digest	text NOT NULL,
	pushed	timestamp NOT NULL,
	pulled	timestamp NULL,
	PRIMARY KEY(digest)
)
;
CREATE TABLE public.tags  (
	name           	text NOT NULL,
	registry       	text NOT NULL,
	repository     	text NOT NULL,
	tag            	text NULL,
	manifest_digest	text NOT NULL,
	pushed         	timestamp NOT NULL,
	pulled         	timestamp NULL,
	PRIMARY KEY(name)
)
;
CREATE INDEX blob_digest
	ON public.manifest_blob USING btree (blob_digest)
;
CREATE INDEX deleted_blob_digest
	ON public.deleted_manifest_blob USING btree (blob_digest)
;

ALTER TABLE public.manifest_blob
	ADD CONSTRAINT manifests_fkey
	FOREIGN KEY(manifest_digest)
	REFERENCES public.manifests(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
;
ALTER TABLE public.manifest_blob
	ADD CONSTRAINT blobs_fkey
	FOREIGN KEY(blob_digest)
	REFERENCES public.blobs(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
;
ALTER TABLE public.tags
	ADD CONSTRAINT manifests_fkey
	FOREIGN KEY(manifest_digest)
	REFERENCES public.manifests(digest)
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
;
