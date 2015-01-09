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
)

func TestCanonicalRequest(t *testing.T) {
	testSetup()

	now, err := time.Parse(time.RFC822, "24 May 13 00:00 GMT")
	if err != nil {
		t.Fatalf("Error parsing time: %s", err)
	}

	cr := CanonicalRequest("PUT", path, contentSha, contentMd5, now)
	if cr != expectedCanonicalRequest {
		t.Fatalf("incorrect canonical request\nexpected:\n%s\n\ngot:\n%s", expectedCanonicalRequest, cr)
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
}

var (
	expectedCanonicalRequest = `PUT
/test%24file.text

content-md5:1EfQ6PKJ8WoS/2AnznfCWA==
date:Fri, 24 May 2013 00:00:00 GMT
host:examplebucket.s3.amazonaws.com
x-amz-content-sha256:44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072

content-md5;date;host;x-amz-content-sha256
44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072`
)
