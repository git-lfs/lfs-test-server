package main

import (
	"net/url"
	"testing"
)

var (
	content    = "Welcome to Amazon S3."
	contentSha string
	filePath   = "/test%24file.text"
)

func TestS3SignedHeaderToken(t *testing.T) {
	testSetup()

	token := signedHeaderToken("PUT", filePath, contentSha, now)

	if token != expectedHeaderToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedHeaderToken, token)
	}
}

func TestS3SignedQueryToken(t *testing.T) {
	testSetup()

	token := signedQueryToken("GET", filePath, 300, now)

	if token != expectedQueryToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedQueryToken, token)
	}
}

func TestS3HeaderSignatureLocation(t *testing.T) {
	sig := S3SignHeader("PUT", filePath, contentSha)
	if sig.Location != expectedLocation {
		t.Fatalf("incorrect header location\nexpected:\n%s\n\ngot:\n%s", expectedLocation, sig.Location)
	}
}

func TestS3QuerySignatureLocation(t *testing.T) {
	sig := S3SignQuery("GET", filePath, 300)

	u, err := url.Parse(sig.Location)
	if err != nil {
		t.Fatalf("error parsing query location %s", err)
	}

	if h := u.Host; h != "examplebucket.s3.amazonaws.com" {
		t.Fatalf("incorrect url host\nexpected:\nexamplebucket.s3.amazonaws.com\n\ngot:\n%s", h)
	}

	dp, _ := url.QueryUnescape(filePath)
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
	expectedHeaderToken = `AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=host;x-amz-content-sha256;x-amz-date,Signature=1e15d288c066d54cc20d8f664ade3fa96e6744b2861304671f6eb633045a2d4a`
	expectedQueryToken  = `aa3bfd20c9769c0a95f556fc43ed6665f3f2f875950ad4fe53e84de73f0b2022`
	expectedLocation    = `https://examplebucket.s3.amazonaws.com/test%24file.text`
)
