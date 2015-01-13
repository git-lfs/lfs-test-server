package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetJsonAuthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d %s", res.StatusCode, req.URL)
	}

	var meta Meta
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != authedOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", authedOid, meta.Oid)
	}

	if meta.Size != 42 {
		t.Fatalf("expected to see a size of `42`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "https://examplebucket.s3.amazonaws.com"+oidPath(authedOid) {
		t.Fatalf("expected download link, got %s", download.Href)
	}

	upload := meta.Links["upload"]
	if upload.Href != "https://examplebucket.s3.amazonaws.com"+oidPath(authedOid) {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}
}

func TestGetJsonUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+unauthedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d %s", res.StatusCode, req.URL)
	}
}

func TestGetJsonReadOnly(t *testing.T) {

}

func TestGetAuthed(t *testing.T) {

}

func TestGetUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+unauthedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d %s", res.StatusCode, req.URL)
	}
}

var (
	now         time.Time
	mediaServer *httptest.Server
	metaServer  *httptest.Server

	authedOid   = "44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072"
	unauthedOid = "5be476dce1f7c1fbaf41bf9c0097e1725d7d26b74ea93543989d1a2b76fef4a5"
)

func testSetup() {
	Config.AwsKey = "AKIAIOSFODNN7EXAMPLE"
	Config.AwsSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	Config.AwsBucket = "examplebucket"

	contentSha = sha256Hex([]byte(content))
	now, _ = time.Parse(time.RFC822, "24 May 13 00:00 GMT")

	mediaServer = httptest.NewServer(newServer())
	metaServer = httptest.NewServer(newMetaServer())
	Config.MetaEndpoint = metaServer.URL
}

func testTeardown() {
	mediaServer.Close()
	metaServer.Close()
}

func newMetaServer() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/{user}/{repo}/{oid}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		oid := vars["oid"]
		if oid == authedOid {
			fmt.Fprintf(w, `{"oid":"%s","size":42}`, oid)
		} else {
			w.WriteHeader(404)
		}
	})
	return router
}
