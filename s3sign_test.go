package main

import (
	"net/url"
	"testing"
)

var (
	content    = "Welcome to Amazon S3."
	contentSha string
	path       = "/test%24file.text"
)

func TestS3SignedHeaderToken(t *testing.T) {
	testSetup()

	token := signedHeaderToken("PUT", path, contentSha, now)

	if token != expectedHeaderToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedHeaderToken, token)
	}
}

func TestS3SignedQueryToken(t *testing.T) {
	testSetup()

	token := signedQueryToken("GET", path, 300, now)

	if token != expectedQueryToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedQueryToken, token)
	}
}

func TestS3HeaderSignatureLocation(t *testing.T) {
	sig := S3SignHeader("PUT", path, contentSha)
	if sig.Location != expectedLocation {
		t.Fatalf("incorrect header location\nexpected:\n%s\n\ngot:\n%s", expectedLocation, sig.Location)
	}
}

func TestS3QuerySignatureLocation(t *testing.T) {
	sig := S3SignQuery("GET", path, 300)

	u, err := url.Parse(sig.Location)
	if err != nil {
		t.Fatalf("error parsing query location %s", err)
	}

	if h := u.Host; h != "examplebucket.s3.amazonaws.com" {
		t.Fatalf("incorrect url host\nexpected:\nexamplebucket.s3.amazonaws.com\n\ngot:\n%s", h)
	}

	dp, _ := url.QueryUnescape(path)
	if p := u.Path; p != dp {
		t.Fatalf("incorrect url path\nexpected:\n%s\n\ngot:\n%s", dp, p)
	}

	v := u.Query()
	if alg := v.Get("X-Amz-Algorithm"); alg != "AWS4-HMAC-SHA256" {
		t.Fatalf("incorrect algorithm\nexpected:\nAWS4-HMAC-SHA256\n\ngot:\n%s", alg)
	}

	if exp := v.Get("X-Amz-Expires"); exp != "300" {
		t.Fatalf("incorrect expires\nexpected:\n300\n\ngot:\n%s", exp)
	}

	if h := v.Get("X-Amz-SignedHeaders"); h != "host" {
		t.Fatalf("incorrect signed headers\nexpected:\nhost\n\ngot:\n%s", h)
	}

	if v.Get("X-Amz-Credential") == "" {
		t.Fatalf("amazon credential not present")
	}

	if v.Get("X-Amz-Date") == "" {
		t.Fatalf("amazon date not present")
	}

	if v.Get("X-Amz-Signature") == "" {
		t.Fatalf("amazon signature not present")
	}
}

const (
	expectedHeaderToken = `AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=date;host;x-amz-content-sha256,Signature=6eb5d2a63d219da38ce1e6d23114b5800721dc63c2b66dca4f7612d0fb13f975`
	expectedQueryToken  = `4696984d3b6c1716e0aa28358ef3a6f2abf18a08c86751488935960fce58000e`
	expectedLocation    = `https://examplebucket.s3.amazonaws.com/test%24file.text`
)
