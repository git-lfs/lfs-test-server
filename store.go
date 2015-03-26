package main

import (
	"errors"
	"net/http"
	"path"

	"github.com/rubyist/circuitbreaker"
)

var (
	Client = &httpClient{
		cb: circuit.NewRateBreaker(0.20, 100),
	}

	apiAuthError = errors.New("auth error")
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
