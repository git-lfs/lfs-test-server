package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetAuthed(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.SetBasicAuth(testUser, testPass)
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 302, got %d", res.StatusCode)
	}

	by, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected response to contain content, got error: %s", err)
	}

	if string(by) != "content" {
		t.Fatalf("expected content to be `content`, got: %s", string(by))
	}
}

func TestGetUnauthed(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/user/repo/objects/"+authedOid, nil)
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

func TestGetMetaAuthed(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/bilbo/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.SetBasicAuth(testUser, testPass)
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d %s", res.StatusCode, req.URL)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != authedOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", authedOid, meta.Oid)
	}

	if meta.Size != 1234 {
		t.Fatalf("expected to see a size of `1234`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/bilbo/repo/objects/"+authedOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}
}

func TestGetMetaUnauthed(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", res.StatusCode)
	}
}

func TestPostAuthedNewObject(t *testing.T) {
	req, err := http.NewRequest("POST", lfsServer.URL+"/bilbo/repo/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.SetBasicAuth(testUser, testPass)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, nonexistingOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != nonexistingOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", nonexistingOid, meta.Oid)
	}

	if meta.Size != 1234 {
		t.Fatalf("expected to see a size of `1234`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/bilbo/repo/objects/"+nonexistingOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}

	upload, ok := meta.Links["upload"]
	if !ok {
		t.Fatal("expected upload link to be present")
	}

	if upload.Href != "http://localhost:8080/bilbo/repo/objects/"+nonexistingOid {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}
}

func TestPostAuthedExistingObject(t *testing.T) {
	req, err := http.NewRequest("POST", lfsServer.URL+"/bilbo/repo/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.SetBasicAuth(testUser, testPass)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != authedOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", authedOid, meta.Oid)
	}

	if meta.Size != 1234 {
		t.Fatalf("expected to see a size of `1234`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/bilbo/repo/objects/"+authedOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}

	upload, ok := meta.Links["upload"]
	if !ok {
		t.Fatalf("expected upload link to be present")
	}

	if upload.Href != "http://localhost:8080/bilbo/repo/objects/"+authedOid {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}
}

func TestPostUnauthed(t *testing.T) {
	req, err := http.NewRequest("POST", lfsServer.URL+"/bilbo/readonly/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", res.StatusCode)
	}
}

func TestPut(t *testing.T) {
}

func TestMediaTypesRequired(t *testing.T) {
	m := []string{"GET", "PUT", "POST", "HEAD"}
	for _, method := range m {
		req, err := http.NewRequest(method, lfsServer.URL+"/user/repo/objects/"+authedOid, nil)
		if err != nil {
			t.Fatalf("request error: %s", err)
		}
		req.SetBasicAuth(testUser, testPass)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("response error: %s", err)
		}

		if res.StatusCode != 404 {
			t.Fatalf("expected status 404, got %d", res.StatusCode)
		}
	}
}

func TestMediaTypesParsed(t *testing.T) {
	req, err := http.NewRequest("GET", lfsServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.SetBasicAuth(testUser, testPass)
	req.Header.Set("Accept", contentMediaType+"; charset=utf-8")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
}

var (
	lfsServer        *httptest.Server
	testMetaStore    *MetaStore
	testContentStore *ContentStore
	testUser         = "bilbo"
	testPass         = "baggins"
	authedOid        = "44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072"
	nonexistingOid   = "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f"
)

func TestMain(m *testing.M) {
	Config.Scheme = "http"

	os.Remove("lfs-test.db")

	var err error
	testMetaStore, err = NewMetaStore("lfs-test.db")
	if err != nil {
		fmt.Printf("Error creating meta store: %s", err)
		os.Exit(1)
	}

	testContentStore, err = NewContentStore("lfs-content-test")
	if err != nil {
		fmt.Printf("Error creating content store: %s", err)
		os.Exit(1)
	}

	if err := seedMetaStore(); err != nil {
		fmt.Printf("Error seeding meta store: %s", err)
		os.Exit(1)
	}

	if err := seedContentStore(); err != nil {
		fmt.Printf("Error seeding content store: %s", err)
		os.Exit(1)
	}

	app := NewApp(testContentStore, testMetaStore)
	lfsServer = httptest.NewServer(app.router)

	logger = NewKVLogger(ioutil.Discard)

	ret := m.Run()

	lfsServer.Close()
	testMetaStore.Close()
	os.Remove("lfs-test.db")
	os.RemoveAll("lfs-content-test")

	os.Exit(ret)
}

func seedMetaStore() error {
	if err := testMetaStore.AddUser(testUser, testPass); err != nil {
		return err
	}

	rv := &RequestVars{User: testUser, Password: testPass, Oid: authedOid, Size: 1234}
	if _, err := testMetaStore.Put(rv); err != nil {
		return err
	}

	return nil
}

func seedContentStore() error {
	meta := &Meta{Oid: authedOid}
	buf := bytes.NewBuffer([]byte("content"))
	if err := testContentStore.Put(meta, buf); err != nil {
		return err
	}

	return nil
}
