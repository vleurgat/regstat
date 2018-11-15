// +build database
//
// to run this test start up a test Postgres db in a Docker container, e.g. ...
//
//   docker run --rm --name=db-test -e POSTGRES_PASSWORD='' -p 5432:5432 postgres:10
//
// and then use ... "go test -tags database" ... to run the test
//
// finally kill the Postgres container once the test has completed

package postgres

import (
	"testing"
	"time"

	"github.com/vleurgat/regstat/internal/app/database"
)

var (
	db            database.Database
	dbInitialised = false
)

func createTestDatabase() {
	if !dbInitialised {
		db = CreateDatabase("host=spike port=5432 user=postgres password=\"\" sslmode=disable")
		dbInitialised = true
	}
}

func TestCreateSchemaIfNecessary(t *testing.T) {
	createTestDatabase()
	db.CreateSchemaIfNecessary()

	conn := db.GetConnection()
	var schemaExists bool
	conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM information_schema.schemata "+
		"WHERE schema_name = $1"+
		")",
		"regstat").Scan(&schemaExists)
	if !schemaExists {
		t.Error("expected schema to exist")
	}
	var blobTableExists bool
	conn.QueryRow("SELECT EXISTS("+
		"SELECT 1 FROM information_schema.tables "+
		"WHERE table_schema = $1 and table_name = $2"+
		")",
		"regstat", "blobs").Scan(&blobTableExists)
	if !blobTableExists {
		t.Error("expected blobs table to exist")
	}
}

func TestPushPullDelete(t *testing.T) {
	createTestDatabase()
	conn := db.GetConnection()

	blobPushTime := time.Now()
	testBlob := database.Blob{Digest: "blob1234", Pushed: blobPushTime}

	t.Run("push blob", func(t *testing.T) {
		db.PushBlob(&testBlob)
		var blobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.blobs "+
			"WHERE digest = $1"+
			")",
			"blob1234").Scan(&blobExists)
		if !blobExists {
			t.Fatal("expected blob to exist")
		}
	})

	t.Run("is blob", func(t *testing.T) {
		isBlob := db.IsBlob(testBlob.Digest)
		if !isBlob {
			t.Error("expected pushed blob to be a blob")
		}
		isBlob = db.IsBlob("fake1234")
		if isBlob {
			t.Error("expected fake blob to not be a blob")
		}
	})

	t.Run("pull blob", func(t *testing.T) {
		testBlob.Pulled = time.Now()
		db.PullBlob(&testBlob)
		var blobHasBeenPulled bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.blobs "+
			"WHERE digest = $1 AND pulled IS NOT NULL AND pulled > pushed"+
			")",
			"blob1234").Scan(&blobHasBeenPulled)
		if !blobHasBeenPulled {
			t.Fatal("expected blob to have been pulled")
		}
	})

	manifestPushTime := time.Now()
	testManifest := database.Manifest{Digest: "man1234", Pushed: manifestPushTime}
	testManifest.Blobs = append(testManifest.Blobs, testBlob)

	t.Run("push manifest", func(t *testing.T) {
		db.PushManifest(&testManifest)
		var manifestExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifests "+
			"WHERE digest = $1"+
			")",
			"man1234").Scan(&manifestExists)
		if !manifestExists {
			t.Fatal("expected manifest to exist")
		}
		var manifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifest_blob "+
			"WHERE manifest_digest = $1 AND blob_digest = $2"+
			")",
			"man1234", "blob1234").Scan(&manifestBlobExists)
		if !manifestBlobExists {
			t.Fatal("expected manifest_blob to exist")
		}
	})

	t.Run("is manifest", func(t *testing.T) {
		isManifest := db.IsManifest(testManifest.Digest)
		if !isManifest {
			t.Error("expected pushed manifest to be a manifest")
		}
		isManifest = db.IsManifest("fake1234")
		if isManifest {
			t.Error("expected fake manifest to not be a manifest")
		}
	})

	t.Run("pull manifest", func(t *testing.T) {
		testManifest.Pulled = time.Now().Truncate(time.Second)
		db.PullManifest(&testManifest)
		var manifestHasBeenPulled bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifests "+
			"WHERE digest = $1 AND pulled = $2"+
			")",
			"man1234", testManifest.Pulled).Scan(&manifestHasBeenPulled)
		if !manifestHasBeenPulled {
			t.Fatal("expected manifest to have been pulled")
		}
		var blobHasBeenPulled bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.blobs "+
			"WHERE digest = $1 AND pulled = $2"+
			")",
			"blob1234", testManifest.Pulled).Scan(&blobHasBeenPulled)
		if !blobHasBeenPulled {
			t.Fatal("expected blob to have been pulled")
		}
	})

	tagPushTime := time.Now()
	testTag := database.Tag{Name: "tag1234", Registry: "reg1", Repository: "rep1", Tag: "tag1", Manifest: testManifest, Pushed: tagPushTime}

	t.Run("push tag", func(t *testing.T) {
		db.PushTag(&testTag)
		var tagExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.tags "+
			"WHERE name = $1"+
			")",
			"tag1234").Scan(&tagExists)
		if !tagExists {
			t.Fatal("expected tag to exist")
		}
	})

	t.Run("pull tag", func(t *testing.T) {
		testTag.Pulled = time.Now()
		db.PullTag(&testTag)
		var tagHasBeenPulled bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.tags "+
			"WHERE name = $1 AND pulled IS NOT NULL AND pulled > pushed"+
			")",
			"tag1234").Scan(&tagHasBeenPulled)
		if !tagHasBeenPulled {
			t.Fatal("expected tag to have been pulled")
		}
	})

	t.Run("delete blob", func(t *testing.T) {
		db.DeleteBlob(testBlob.Digest)
		if db.IsBlob(testBlob.Digest) {
			t.Error("expected blob to have been deleted")
		}
		var deletedBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.deleted_blobs "+
			"WHERE digest = $1"+
			")",
			"blob1234").Scan(&deletedBlobExists)
		if !deletedBlobExists {
			t.Fatal("expected blob to have been written to deleted table")
		}
		var deletedManifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.deleted_manifest_blob "+
			"WHERE blob_digest = $1"+
			")",
			"blob1234").Scan(&deletedManifestBlobExists)
		if !deletedManifestBlobExists {
			t.Fatal("expected manifest_blob to have been written to deleted table")
		}
		var blobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.blobs "+
			"WHERE digest = $1"+
			")",
			"blob1234").Scan(&blobExists)
		if blobExists {
			t.Fatal("expected blob to not exist")
		}
		var manifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifest_blob "+
			"WHERE blob_digest = $1"+
			")",
			"blob1234").Scan(&manifestBlobExists)
		if manifestBlobExists {
			t.Fatal("expected manifest_blob to not exist")
		}
	})

	t.Run("push manifest #2", func(t *testing.T) {
		db.PushManifest(&testManifest)
		var manifestExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifests "+
			"WHERE digest = $1"+
			")",
			"man1234").Scan(&manifestExists)
		if !manifestExists {
			t.Fatal("expected manifest to exist")
		}
		var manifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifest_blob "+
			"WHERE manifest_digest = $1 AND blob_digest = $2"+
			")",
			"man1234", "blob1234").Scan(&manifestBlobExists)
		if !manifestBlobExists {
			t.Fatal("expected manifest_blob to have been recreated")
		}
		var blobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.blobs "+
			"WHERE digest = $1"+
			")",
			"blob1234").Scan(&blobExists)
		if !blobExists {
			t.Fatal("expected blob to have been recreated")
		}
	})

	t.Run("delete manifest", func(t *testing.T) {
		db.DeleteManifest(testManifest.Digest)
		if db.IsManifest(testManifest.Digest) {
			t.Error("expected manifest to have been deleted")
		}
		var deletedManifestExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.deleted_manifests "+
			"WHERE digest = $1"+
			")",
			"man1234").Scan(&deletedManifestExists)
		if !deletedManifestExists {
			t.Fatal("expected manifest to have been written to deleted table")
		}
		var deletedTagExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.deleted_tags "+
			"WHERE manifest_digest = $1"+
			")",
			"man1234").Scan(&deletedTagExists)
		if !deletedTagExists {
			t.Fatal("expected tag to have been written to deleted table")
		}
		var deletedManifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.deleted_manifest_blob "+
			"WHERE manifest_digest = $1"+
			")",
			"man1234").Scan(&deletedManifestBlobExists)
		if !deletedManifestBlobExists {
			t.Fatal("expected manifest_blob to have been written to deleted table")
		}
		var manifestExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifests "+
			"WHERE digest = $1"+
			")",
			"man1234").Scan(&manifestExists)
		if manifestExists {
			t.Fatal("expected manifest to not exist")
		}
		var tagExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.tags "+
			"WHERE manifest_digest = $1"+
			")",
			"man1234").Scan(&tagExists)
		if tagExists {
			t.Fatal("expected tag to not exist")
		}
		var manifestBlobExists bool
		conn.QueryRow("SELECT EXISTS("+
			"SELECT 1 FROM regstat.manifest_blob "+
			"WHERE manifest_digest = $1"+
			")",
			"man1234").Scan(&manifestBlobExists)
		if manifestBlobExists {
			t.Fatal("expected manifest_blob to not exist")
		}
	})
}
