package main

import (
	"testing"
)

var (
	content    = "Welcome to Amazon S3."
	contentSha string
	path       = "/test%24file.text"
)

func TestCanonicalRequest(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, now)
	if cr != expectedCanonicalRequest {
		t.Fatalf("incorrect canonical request\nexpected:\n%s\n\ngot:\n%s", expectedCanonicalRequest, cr)
	}
}

func TestStringToSign(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, now)
	sts := StringToSign(cr, now)
	if sts != expectedStringToSign {
		t.Fatalf("incorrect string to sign\nexpected:\n%s\n\ngot:\n%s", expectedStringToSign, sts)
	}
}

func TestSignature(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, now)
	sts := StringToSign(cr, now)
	signature := Signature(sts, now)
	if signature != expectedSignature {
		t.Fatalf("incorrect signature\nexpected:\n%s\n\ngot:\n%s", expectedSignature, signature)
	}
}

func TestS3SignedToken(t *testing.T) {
	testSetup()

	token := S3SignedToken("PUT", path, contentSha, now)
	if token != expectedToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedToken, token)
	}
}

func TestS3Token(t *testing.T) {
	testSetup()

	token := S3NewToken("PUT", path, contentSha)

	if token.Token != S3SignedToken("PUT", path, contentSha, token.Time) {
		t.Fatalf("incorrect token signature")
	}

	if token.Location != expectedLocation {
		t.Fatalf("incorrect token location\nexpected:\n%s\n\ngot:\n%s", expectedLocation, token.Location)
	}
}

const (
	expectedCanonicalRequest = `PUT
/test%24file.text

date:Fri, 24 May 2013 00:00:00 GMT
host:examplebucket.s3.amazonaws.com
x-amz-content-sha256:44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072

date;host;x-amz-content-sha256
44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072`

	expectedStringToSign = `AWS4-HMAC-SHA256
Fri, 24 May 2013 00:00:00 GMT
20130524/us-east-1/s3/aws4_request
627af852f282e30ce4aa940ee749726c031dfeb5fb8d793c0bf86128165d9e2b`

	expectedSignature = `6eb5d2a63d219da38ce1e6d23114b5800721dc63c2b66dca4f7612d0fb13f975`

	expectedToken = `AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=date;host;x-amz-content-sha256,Signature=6eb5d2a63d219da38ce1e6d23114b5800721dc63c2b66dca4f7612d0fb13f975`

	expectedLocation = `https://examplebucket.s3.amazonaws.com/test%24file.text`
)
