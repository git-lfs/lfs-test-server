package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/rubyist/circuitbreaker"
	"net/http"
	"path"
)

var (
	Client = &httpClient{
		cb: circuit.NewRateBreaker(0.20, 100),
	}
)

type httpClient struct {
	cb *circuit.Breaker
	http.Client
}

func (c *httpClient) Do(r *http.Request) (resp *http.Response, err error) {
	if c.cb.Ready() {
		resp, err = c.Client.Do(r)
		if err != nil || resp.StatusCode >= 500 {
			c.cb.Fail()
			return
		}

		c.cb.Success()
	} else {
		logger.Log(D{"fn": "Client.Do", "err": "tripped"})
	}

	return
}

// S3Redirector implements the ContentStorer interface to serve content via redirecting to S3.
type S3Redirector struct {
}

// Get will use the provided object Meta data to write a redirect Location and status to
// the Response Writer. It generates a signed S3 URL that is valid for 5 minutes.
func (s *S3Redirector) Get(meta *Meta, w http.ResponseWriter, r *http.Request) int {
	token := S3SignQuery("GET", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 300)
	w.Header().Set("Location", token.Location)
	w.WriteHeader(302)
	return 302
}

// PutLink generates an signed S3 link that will allow the client to PUT data into S3. This
// link includes the x-amz-content-sha256 header which will ensure that the client uploads only
// data that will match the OID.
func (s *S3Redirector) PutLink(meta *Meta) *link {
	token := S3SignHeader("PUT", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), meta.Oid)
	header := make(map[string]string)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = meta.Oid
	header["x-amz-date"] = token.Time.Format(isoLayout)

	return &link{Href: token.Location, Header: header}
}

// Exists checks to see if an object exists on S3.
func (s *S3Redirector) Exists(meta *Meta) (bool, error) {
	token := S3SignQuery("HEAD", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 30)
	req, err := http.NewRequest("HEAD", token.Location, nil)
	if err != nil {
		return false, err
	}

	res, err := Client.Do(req)
	if err != nil {
		return false, err
	}

	if res.StatusCode != 200 {
		return false, err
	}

	return true, nil
}

func oidPath(oid string) string {
	dir := path.Join(oid[0:2], oid[2:4])

	return path.Join("/", dir, oid)
}

// MetaStore implements the MetaStorer interface to provide metadata from an external HTTP API.
type MetaStore struct {
}

// MetaLink generates a URI path using the configured MetaEndpoint.
func (s *MetaStore) MetaLink(v *RequestVars) string {
	return Config.MetaEndpoint + "/" + path.Join(v.User, v.Repo, "media", "blobs", v.Oid)
}

// Get retrieves metadata from the backend API.
func (s *MetaStore) Get(v *RequestVars) (*Meta, error) {
	req, err := http.NewRequest("GET", s.MetaLink(v), nil)
	if err != nil {
		logger.Log(D{"fn": "meta.Get", "err": err})
		return nil, err
	}
	req.Header.Set("Accept", Config.ApiMediaType)
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
	}
	if v.RequestId != "" {
		req.Header.Set("X-GitHub-Request-Id", v.RequestId)
	}

	res, err := Client.Do(req)
	if err != nil {
		logger.Log(D{"fn": "meta.Get", "err": err})
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode == 200 {
		var m Meta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Log(D{"fn": "meta.Get", "err": err})
			return nil, err
		}

		logger.Log(D{"fn": "meta.Get", "status": res.StatusCode})
		return &m, nil
	}

	if res.StatusCode == 204 {
		return &Meta{Oid: v.Oid, Size: v.Size, PathPrefix: v.PathPrefix}, nil
	}

	logger.Log(D{"fn": "meta.Get", "status": res.StatusCode})
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

// Send POSTs metadata to the backend API.
func (s *MetaStore) Send(v *RequestVars) (*Meta, error) {
	req, err := signedApiPost(s.MetaLink(v), v)

	if v.RequestId != "" {
		req.Header.Set("X-GitHub-Request-Id", v.RequestId)
	}

	res, err := Client.Do(req)
	if err != nil {
		logger.Log(D{"fn": "meta.Send", "err": err})
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 403 {
		logger.Log(D{"fn": "meta.Send", "status": 403})
		return nil, apiAuthError
	}

	if res.StatusCode == 200 || res.StatusCode == 201 {
		var m Meta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Log(D{"fn": "meta.Send", "err": err})
			return nil, err
		}
		m.existing = res.StatusCode == 200
		logger.Log(D{"fn": "meta.Send", "status": res.StatusCode})
		return &m, nil
	}

	logger.Log(D{"fn": "meta.Send", "status": res.StatusCode})
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

// Verify is used during the callback phase to indicate to the backend API that the
// object has been received.
func (s *MetaStore) Verify(v *RequestVars) error {
	url := Config.MetaEndpoint + "/" + path.Join(v.User, v.Repo, "media", "blobs", v.Oid, "verify")

	req, err := signedApiPost(url, v)
	if err != nil {
		return err
	}
	if v.RequestId != "" {
		req.Header.Set("X-GitHub-Request-Id", v.RequestId)
	}

	res, err := Client.Do(req)
	if err != nil {
		logger.Log(D{"fn": "meta.Verify", "err": err})
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == 403 {
		logger.Log(D{"fn": "meta.Verify", "err": 403})
		return apiAuthError
	}

	logger.Log(D{"fn": "meta.Verify", "status": res.StatusCode})
	if res.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("status: %d", res.StatusCode)
}

func signedApiPost(url string, v *RequestVars) (*http.Request, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(&Meta{Oid: v.Oid, Size: v.Size})

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", Config.ApiMediaType)
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
	}

	if Config.HmacKey != "" {
		mac := hmac.New(sha256.New, []byte(Config.HmacKey))
		mac.Write(buf.Bytes())
		req.Header.Set("Content-Hmac", "sha256 "+hex.EncodeToString(mac.Sum(nil)))
	}

	return req, nil
}
