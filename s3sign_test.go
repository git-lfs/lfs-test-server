package main

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
	"time"
)

var (
	content    = "Welcome to Amazon S3."
	contentSha string
	contentMd5 string
	path       = "/test%24file.text"
	now        time.Time
)

func TestCanonicalRequest(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, contentMd5, now)
	if cr != expectedCanonicalRequest {
		t.Fatalf("incorrect canonical request\nexpected:\n%s\n\ngot:\n%s", expectedCanonicalRequest, cr)
	}
}

func TestStringToSign(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, contentMd5, now)
	sts := StringToSign(cr, now)
	if sts != expectedStringToSign {
		t.Fatalf("incorrect string to sign\nexpected:\n%s\n\ngot:\n%s", expectedStringToSign, sts)
	}
}

func TestSignature(t *testing.T) {
	testSetup()

	cr := CanonicalRequest("PUT", path, contentSha, contentMd5, now)
	sts := StringToSign(cr, now)
	signature := Signature(sts, now)
	if signature != expectedSignature {
		t.Fatalf("incorrect signature\nexpected:\n%s\n\ngot:\n%s", expectedSignature, signature)
	}
}

func TestS3SignedToken(t *testing.T) {
	testSetup()

	token := S3SignedToken("PUT", path, contentSha, contentMd5, now)
	if token != expectedToken {
		t.Fatalf("incorrect token\nexpected:\n%s\n\ngot:\n%s", expectedToken, token)
	}
}

func TestS3Token(t *testing.T) {
	testSetup()

	token := S3NewToken("PUT", path, contentSha, contentMd5)

	if token.Token != S3SignedToken("PUT", path, contentSha, contentMd5, token.Time) {
		t.Fatalf("incorrect token signature")
	}

	if token.Location != expectedLocation {
		t.Fatalf("incorrect token location\nexpected:\n%s\n\ngot:\n%s", expectedLocation, token.Location)
	}
}

func testSetup() {
	Config.AwsKey = "AKIAIOSFODNN7EXAMPLE"
	Config.AwsSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	Config.AwsBucket = "examplebucket"

	contentSha = sha256Hex([]byte(content))

	hash := md5.New()
	hash.Write([]byte(content))
	contentMd5 = hex.EncodeToString(hash.Sum(nil))

	now, _ = time.Parse(time.RFC822, "24 May 13 00:00 GMT")
}

const (
	expectedCanonicalRequest = `PUT
/test%24file.text

content-md5:1EfQ6PKJ8WoS/2AnznfCWA==
date:Fri, 24 May 2013 00:00:00 GMT
host:examplebucket.s3.amazonaws.com
x-amz-content-sha256:44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072

content-md5;date;host;x-amz-content-sha256
44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072`

	expectedStringToSign = `AWS4-HMAC-SHA256
Fri, 24 May 2013 00:00:00 GMT
20130524/us-east-1/s3/aws4_request
35918b6d5f2f402b12a82af55c3e432bd99cdc840c8553ced35891fbedd9c2fc`

	expectedSignature = `b765307d50cb6df6018156c64491b2f177dcb4484655926e9a7d7105bdc5fb87`

	expectedToken = `AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=content-md5;date;host;x-amz-content-sha256,Signature=b765307d50cb6df6018156c64491b2f177dcb4484655926e9a7d7105bdc5fb87`

	expectedLocation = `https://examplebucket.s3.amazonaws.com/test%24file.text`
)
