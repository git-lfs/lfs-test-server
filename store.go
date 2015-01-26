package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

type S3Redirector struct {
}

func (s *S3Redirector) Get(meta *Meta, w http.ResponseWriter, r *http.Request) int {
	token := S3SignQuery("GET", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 86400)
	w.Header().Set("Location", token.Location)
	return 302
}

func (s *S3Redirector) PutLink(meta *Meta) *link {
	token := S3SignHeader("PUT", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), meta.Oid)
	header := make(map[string]string)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = meta.Oid
	header["x-amz-date"] = token.Time.Format(isoLayout)

	return &link{Href: token.Location, Header: header}
}

func (s *S3Redirector) Verify(meta *Meta) (bool, error) {
	token := S3SignQuery("HEAD", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 30)
	req, err := http.NewRequest("HEAD", token.Location, nil)
	if err != nil {
		return false, err
	}

	res, err := http.DefaultClient.Do(req)
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

type MetaStore struct {
}

func (s *MetaStore) MetaLink(v *RequestVars) string {
	return Config.MetaEndpoint + "/" + path.Join(v.User, v.Repo, "media", "blobs", v.Oid)
}

func (s *MetaStore) Get(v *RequestVars) (*Meta, error) {
	req, err := http.NewRequest("GET", s.MetaLink(v), nil)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	req.Header.Set("Accept", Config.ApiMediaType)
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode == 200 {
		var m Meta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Printf("[META] error - %s", err)
			return nil, err
		}

		logger.Printf("[META] status - %d", res.StatusCode)
		return &m, nil
	}

	if res.StatusCode == 204 {
		return &Meta{Oid: v.Oid, Size: v.Size, PathPrefix: v.PathPrefix}, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func (s *MetaStore) Send(v *RequestVars) (*Meta, error) {
	req, err := signedApiPost(s.MetaLink(v), v)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 403 {
		logger.Printf("[META] 403")
		return nil, apiAuthError
	}

	if res.StatusCode == 200 || res.StatusCode == 201 {
		var m Meta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Printf("[META] error - %s", err)
			return nil, err
		}
		m.existing = res.StatusCode == 200
		logger.Printf("[META] status - %d", res.StatusCode)
		return &m, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func (s *MetaStore) Verify(v *RequestVars) error {
	url := Config.MetaEndpoint + "/" + path.Join(v.User, v.Repo, "media", "blobs", "verify", v.Oid)

	req, err := signedApiPost(url, v)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[VERIFY] error - %s", err)
		return err
	}

	defer res.Body.Close()

	if res.StatusCode == 403 {
		logger.Printf("[VERIFY] 403")
		return apiAuthError
	}

	logger.Printf("[VERIFY] status - %d", res.StatusCode)
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
