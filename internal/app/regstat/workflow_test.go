package regstat

import (
	"github.com/docker/distribution/notifications"
	"testing"
		"encoding/json"
	"time"
	"fmt"
	"github.com/vleurgat/regstat/internal/app/registry"
	"errors"
	"net/http"
	"io/ioutil"
	"strings"
	"github.com/vleurgat/regstat/internal/app/database/mock"
)

type MockWorkflow struct {
	receivedEvents *[]*notifications.Event
}

func createMockWorkflow() MockWorkflow {
	return MockWorkflow{receivedEvents: &[]*notifications.Event{}}
}

func (wf MockWorkflow) processDelete(event *notifications.Event) {
	*wf.receivedEvents = append(*wf.receivedEvents, event)
}

func (wf MockWorkflow) processPush(event *notifications.Event) {
	*wf.receivedEvents = append(*wf.receivedEvents, event)
}

func (wf MockWorkflow) processPull(event *notifications.Event) {
	*wf.receivedEvents = append(*wf.receivedEvents, event)
}

func createEvent(t *testing.T, body string) *notifications.Event {
	var event notifications.Event
	err := json.Unmarshal([]byte(body), &event)
	if err != nil {
		t.Fatal("failed to create event", err)
	}
	return &event
}

func TestProcessDelete(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		db := mock.CreateDatabase()
		wf := WorkflowImpl{db: db}
		event := createEvent(t, "{}")
		wf.processDelete(event)
		if len(*db.DeletedManifests) != 0 || len(*db.DeletedBlobs) != 0 {
			t.Fatal("expected no deletions")
		}
	})

	t.Run("manifest", func(t *testing.T) {
		db := mock.CreateDatabase()
		db.IsManifestRetValue = true
		wf := WorkflowImpl{db: db}
		event := createEvent(t, "{\"target\":{\"digest\":\"boo\"}}")
		wf.processDelete(event)
		if len(*db.DeletedManifests) != 1 || len(*db.DeletedBlobs) != 0 {
			t.Fatal("expected 1 manifest and no blob deletions")
		}
		if (*db.DeletedManifests)[0] != "boo" {
			t.Error("unexpected deleted manifest digest")
		}
	})

	t.Run("blob", func(t *testing.T) {
		db := mock.CreateDatabase()
		db.IsBlobRetValue = true
		wf := WorkflowImpl{db: db}
		event := createEvent(t, "{\"target\":{\"digest\":\"boo\"}}")
		wf.processDelete(event)
		if len(*db.DeletedManifests) != 0 || len(*db.DeletedBlobs) != 1 {
			t.Fatal("expected 1 blob and no manifest deletions")
		}
		if (*db.DeletedBlobs)[0] != "boo" {
			t.Error("unexpected deleted blob digest")
		}
	})
}

func TestProcessPull(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowStr := now.Format("2006-01-02T15:04:05Z07:00")

	t.Run("unknown", func(t *testing.T) {
		db := mock.CreateDatabase()
		wf := WorkflowImpl{db: db}
		event := createEvent(t, "{}")
		wf.processPull(event)
		if len(*db.PulledManifests) != 0 || len(*db.PulledBlobs) != 0 || len(*db.PulledTags) != 0 {
			t.Fatal("expected no pulls")
		}
	})

	t.Run("blob", func(t *testing.T) {
		db := mock.CreateDatabase()
		wf := WorkflowImpl{db: db}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"digest\":\"boo\", \"mediaType\":\"application/octet-stream\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPull(event)
		if len(*db.PulledManifests) != 0 || len(*db.PulledBlobs) != 1 || len(*db.PulledTags) != 0 {
			t.Fatal("expected 1 blob pull only")
		}
		if (*db.PulledBlobs)[0].Digest != "boo" {
			t.Error("unexpected pulled blob digest")
		}
		if !now.Equal((*db.PulledBlobs)[0].Pulled) {
			t.Error("unexpected pulled blob timestamp")
		}
	})

	t.Run("manifest no tag", func(t *testing.T) {
		db := mock.CreateDatabase()
		eqr := registry.EquivRegistries{}
		wf := WorkflowImpl{db: db, eqr: &eqr}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"digest\":\"boo\", \"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPull(event)
		if len(*db.PulledManifests) != 0 || len(*db.PulledBlobs) != 0 || len(*db.PulledTags) != 0 {
			t.Fatal("expected no pulls")
		}
	})

	t.Run("manifest with tag", func(t *testing.T) {
		db := mock.CreateDatabase()
		eqr := registry.EquivRegistries{}
		wf := WorkflowImpl{db: db, eqr: &eqr}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"tag\":\"hoo\", \"digest\":\"boo\", \"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPull(event)
		if len(*db.PulledManifests) != 1 || len(*db.PulledBlobs) != 0 || len(*db.PulledTags) != 1 {
			t.Fatal("expected 1 manifest and 1 tag pull")
		}
		if (*db.PulledManifests)[0].Digest != "boo" {
			t.Error("unexpected pulled manifest digest")
		}
		if !now.Equal((*db.PulledManifests)[0].Pulled) {
			t.Error("unexpected pulled manifest timestamp")
		}
		if (*db.PulledTags)[0].Tag != "hoo" && (*db.PulledTags)[0].Name != "/:hoo" {
			t.Error("unexpected pulled tag name")
		}
		if !now.Equal((*db.PulledTags)[0].Pulled) {
			t.Error("unexpected pulled tag timestamp")
		}
	})
}

func TestProcessPush(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowStr := now.Format("2006-01-02T15:04:05Z07:00")

	t.Run("unknown", func(t *testing.T) {
		db := mock.CreateDatabase()
		wf := WorkflowImpl{db: db}
		event := createEvent(t, "{}")
		wf.processPush(event)
		if len(*db.PushedManifests) != 0 || len(*db.PushedBlobs) != 0 || len(*db.PushedTags) != 0 {
			t.Fatal("expected no pushes")
		}
	})

	t.Run("blob", func(t *testing.T) {
		db := mock.CreateDatabase()
		wf := WorkflowImpl{db: db}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"digest\":\"boo\", \"mediaType\":\"application/octet-stream\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPush(event)
		if len(*db.PushedManifests) != 0 || len(*db.PushedBlobs) != 1 || len(*db.PushedTags) != 0 {
			t.Fatal("expected 1 blob push only")
		}
		if (*db.PushedBlobs)[0].Digest != "boo" {
			t.Error("unexpected pushed blob digest")
		}
		if !now.Equal((*db.PushedBlobs)[0].Pushed) {
			t.Error("unexpected pushed blob timestamp")
		}
	})

	t.Run("manifest no enrichment", func(t *testing.T) {
		db := mock.CreateDatabase()
		eqr := registry.EquivRegistries{}
		httpClient := registry.CreateMockHttpClientErr(errors.New("oops"))
		wf := WorkflowImpl{
			db:     db,
			eqr:    &eqr,
			client: registry.CreateClientProvidingHttpClient(httpClient, nil),
		}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"tag\":\"hoo\", \"digest\":\"boo\", \"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPush(event)
		if len(*db.PushedManifests) != 1 || len(*db.PushedBlobs) != 0 || len(*db.PushedTags) != 1 {
			t.Fatal("expected 1 manifest and 1 tag push")
		}
		if (*db.PushedManifests)[0].Digest != "boo" {
			t.Error("unexpected pushed manifest digest")
		}
		if !now.Equal((*db.PushedManifests)[0].Pushed) {
			t.Error("unexpected pushed manifest timestamp")
		}
		if len((*db.PushedManifests)[0].Blobs) != 0 {
			t.Error("expected no associated blobs")
		}
		if (*db.PushedTags)[0].Tag != "hoo" && (*db.PushedTags)[0].Name != "/:hoo" {
			t.Error("unexpected pushed tag name")
		}
		if !now.Equal((*db.PushedTags)[0].Pushed) {
			t.Error("unexpected pushed tag timestamp")
		}
	})

	t.Run("manifest with enrichment", func(t *testing.T) {
		db := mock.CreateDatabase()
		eqr := registry.EquivRegistries{}
		httpClient := registry.CreateMockHttpClient(http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("{\"config\":{\"digest\": \"123456\"}}")),
		})
		wf := WorkflowImpl{
			db:     db,
			eqr:    &eqr,
			client: registry.CreateClientProvidingHttpClient(httpClient, nil),
		}
		event := createEvent(t, fmt.Sprintf(
			"{\"target\":{\"tag\":\"hoo\", \"url\":\"http://hello\", \"digest\":\"boo\", \"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\"}, \"timestamp\":\"%s\"}",
			nowStr))
		wf.processPush(event)
		if len(*db.PushedManifests) != 1 || len(*db.PushedBlobs) != 0 || len(*db.PushedTags) != 1 {
			t.Fatal("expected 1 manifest and 1 tag push")
		}
		if (*db.PushedManifests)[0].Digest != "boo" {
			t.Error("unexpected pushed manifest digest")
		}
		if !now.Equal((*db.PushedManifests)[0].Pushed) {
			t.Error("unexpected pushed manifest timestamp")
		}
		if len((*db.PushedManifests)[0].Blobs) != 1 {
			t.Error("expected 1 associated blob")
		}
		if (*db.PushedManifests)[0].Blobs[0].Digest != "123456" {
			t.Error("unexpected digest of associated blob")
		}
		if (*db.PushedTags)[0].Tag != "hoo" && (*db.PushedTags)[0].Name != "/:hoo" {
			t.Error("unexpected pushed tag name")
		}
		if !now.Equal((*db.PushedTags)[0].Pushed) {
			t.Error("unexpected pushed tag timestamp")
		}
	})
}
