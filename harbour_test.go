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

func TestGetAuthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultTransport.RoundTrip(req) // Do not follow the redirect
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 302 {
		t.Fatalf("expected status 302, got %d", res.StatusCode)
	}
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

func TestGetMetaAuthedReadWrite(t *testing.T) {
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

	upload, ok := meta.Links["upload"]
	if !ok {
		t.Fatalf("expected upload link to be present")
	}

	if upload.Href != "https://examplebucket.s3.amazonaws.com"+oidPath(authedOid) {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}
}

func TestGetMetaAuthedReadOnly(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/readonly/objects/"+authedOid, nil)
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

	if _, ok := meta.Links["upload"]; ok {
		t.Fatal("expected upload link to not be present")
	}
}

func TestGetMetaUnauthed(t *testing.T) {
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
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}

	// Check unauthorized json message
	var msg map[string]string
	dec := json.NewDecoder(res.Body)
	dec.Decode(&msg)

	if m := msg["message"]; m != "Not Found" {
		t.Fatalf("expected a message in the 404 json response")
	}
}

func TestOptions(t *testing.T) {

}

func TestPut(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("PUT", mediaServer.URL+"/user/repo/objects/"+unauthedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", contentMediaType)
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 405 {
		t.Fatalf("expected status 405, got %d", res.StatusCode)
	}
}

func TestMediaTypesRequired(t *testing.T) {
	testSetup()
	defer testTeardown()

	m := []string{"GET", "PUT", "OPTIONS"}
	for _, method := range m {
		req, err := http.NewRequest(method, mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
		if err != nil {
			t.Fatalf("request error: %s", err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("response error: %s", err)
		}

		if res.StatusCode != 404 {
			t.Fatalf("expected status 404, got %d", res.StatusCode)
		}
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
		repo := vars["repo"]

		if oid == authedOid {
			if repo == "readonly" {
				fmt.Fprintf(w, `{"oid":"%s","size":42,"writeable":false}`, oid)
			} else {
				fmt.Fprintf(w, `{"oid":"%s","size":42,"writeable":true}`, oid)
			}
		} else {
			w.WriteHeader(404)
		}
	})
	return router
}
