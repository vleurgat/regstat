package regstat

import (
	"log"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/notifications"
	"github.com/vleurgat/dockerclient/pkg/dockerclient"
	"github.com/vleurgat/regstat/internal/app/database"
	"github.com/vleurgat/regstat/internal/app/registry"
)

// Workflow defines the main registry notification event processing methods.
type Workflow interface {
	processDelete(event *notifications.Event)
	processPush(event *notifications.Event)
	processPull(event *notifications.Event)
}

// WorkflowImpl encapsulates the business logic of how Docker registry
// notifications of tag, manifest and blob pulls, pushes and deletes
// should be intrepreted and persisted. It implements the Workflow interface.
type WorkflowImpl struct {
	db     database.Database
	client dockerclient.Client
	eqr    *registry.EquivRegistries
}

func createBlob(event *notifications.Event) database.Blob {
	return database.Blob{
		Digest: event.Target.Digest.String(),
		Pushed: event.Timestamp,
		Pulled: event.Timestamp,
	}
}

func createManifest(event *notifications.Event) database.Manifest {
	manifest := database.Manifest{
		Digest: event.Target.Digest.String(),
		Pushed: event.Timestamp,
		Pulled: event.Timestamp,
	}
	return manifest
}

func appendBlob(manifest *database.Manifest, digest string, timestamp time.Time) {
	manifest.Blobs = append(manifest.Blobs,
		database.Blob{
			Digest: digest,
			Pushed: timestamp,
			Pulled: timestamp,
		})
}

func createTag(event *notifications.Event, manifest *database.Manifest, eqr *registry.EquivRegistries) database.Tag {
	name := eqr.FindEquivalent(event.Request.Host) + "/" + event.Target.Repository
	if event.Target.Tag != "" {
		name += ":" + event.Target.Tag
	}
	return database.Tag{
		Name:       name,
		Registry:   event.Request.Host,
		Repository: event.Target.Repository,
		Tag:        event.Target.Tag,
		Manifest:   *manifest,
		Pushed:     event.Timestamp,
		Pulled:     event.Timestamp,
	}
}

func enrichManifest(manifest *database.Manifest, v2Manifest *schema2.Manifest, timestamp time.Time) {
	if v2Manifest.Config.Digest != "" {
		appendBlob(manifest, v2Manifest.Config.Digest.String(), timestamp)
	}
	for _, layer := range v2Manifest.Layers {
		appendBlob(manifest, layer.Digest.String(), timestamp)
	}
}

func (wf WorkflowImpl) processDelete(event *notifications.Event) {
	// for delete events we need to lookup whether the digest refers to a blob or a manifest
	if wf.db.IsManifest(event.Target.Digest.String()) {
		wf.db.DeleteManifest(event.Target.Digest.String())
	} else if wf.db.IsBlob(event.Target.Digest.String()) {
		wf.db.DeleteBlob(event.Target.Digest.String())
	} else {
		log.Println("unknown delete event", event)
	}
}

func (wf WorkflowImpl) processPull(event *notifications.Event) {
	switch event.Target.MediaType {
	case "application/octet-stream",
		"application/vnd.docker.image.rootfs.diff.tar.gzip":
		// blob
		blob := createBlob(event)
		wf.db.PullBlob(&blob)
	case "application/vnd.docker.distribution.manifest.v2+json":
		// manifest
		manifest := createManifest(event)
		tag := createTag(event, &manifest, wf.eqr)
		if tag.Tag == "" {
			// the tag will be missing on the response to a pull of a manifest by
			// digest; that usually only happens for technical reasons - e.g. when
			// we want to discover the blobs that are associated with a tag - and
			// so we skip it in order to avoid creating an empty tag entry
			log.Println("ignoring pull of manifest with no tag")
		} else {
			wf.db.PullManifest(&manifest)
			wf.db.PullTag(&tag)
		}
	default:
		log.Println("unknown event media type", event.Target.MediaType)
	}
}

func (wf WorkflowImpl) processPush(event *notifications.Event) {
	switch event.Target.MediaType {
	case "application/octet-stream",
		"application/vnd.docker.image.rootfs.diff.tar.gzip":
		// blob
		blob := createBlob(event)
		wf.db.PushBlob(&blob)
	case "application/vnd.docker.distribution.manifest.v2+json":
		// manifest
		manifest := createManifest(event)
		tag := createTag(event, &manifest, wf.eqr)
		manifestJSON, err := wf.client.GetV2Manifest(event.Target.URL)
		if err == nil {
			enrichManifest(&manifest, &manifestJSON, event.Timestamp)
		}
		wf.db.PushManifest(&manifest)
		wf.db.PushTag(&tag)
	default:
		log.Println("unknown event media type", event.Target.MediaType)
	}
}
