package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"
)

var (
	signedHeaders = "host;x-amz-content-sha256;x-amz-date"
)

const (
	dateLayout = "20060102"
	isoLayout  = "20060102T150405Z"
)

type S3Token struct {
	Token    string
	Location string
	Time     time.Time
}

func S3SignHeader(method, path, sha string) *S3Token {
	t := time.Now().UTC()
	return &S3Token{
		Token:    signedHeaderToken(method, path, sha, t),
		Location: fmt.Sprintf("https://s3.amazonaws.com/%s%s", Config.AwsBucket, path),
		Time:     t,
	}
}

func S3SignQuery(method, path string, expires int) *S3Token {
	t := time.Now().UTC()

	sig := signedQueryToken(method, path, expires, t)

	v := url.Values{}
	v.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	v.Set("X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request", Config.AwsKey, t.Format(dateLayout), Config.AwsRegion))
	v.Set("X-Amz-Date", t.Format(isoLayout))
	v.Set("X-Amz-Expires", fmt.Sprintf("%d", expires))
	v.Set("X-Amz-SignedHeaders", "host")
	v.Set("X-Amz-Signature", sig)
	return &S3Token{
		Token:    sig,
		Location: fmt.Sprintf("https://s3.amazonaws.com/%s%s?%s", Config.AwsBucket, path, v.Encode()),
		Time:     t,
	}
}

func signedHeaderToken(method, path, sha string, t time.Time) string {
	return token(canonicalRequestHeader(method, path, sha, t), t)
}

func signedQueryToken(method, path string, expires int, t time.Time) string {
	c := canonicalRequestQuery(method, path, expires, t)
	stringToSign := stringToSign(c, t)
	return signature(stringToSign, t)
}

func canonicalRequestHeader(method, path, sha string, t time.Time) string {
	return fmt.Sprintf("%s\n%s\n\nhost:s3.amazonaws.com\nx-amz-content-sha256:%s\nx-amz-date:%s\n\n%s\n%s",
		method,
		path,
		sha,
		t.Format(isoLayout),
		signedHeaders,
		sha,
	)
}

func canonicalRequestQuery(method, path string, expires int, t time.Time) string {
	return fmt.Sprintf("%s\n%s\nX-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=%s%%2F%s%%2F%s%%2Fs3%%2Faws4_request&X-Amz-Date=%s&X-Amz-Expires=%d&X-Amz-SignedHeaders=host\nhost:%s.s3.amazonaws.com\n\nhost\nUNSIGNED-PAYLOAD",
		method,
		path,
		Config.AwsKey,
		t.Format(dateLayout),
		Config.AwsRegion,
		t.Format(isoLayout),
		expires,
		Config.AwsBucket,
	)
}

func stringToSign(request string, t time.Time) string {
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/s3/aws4_request\n%s",
		t.Format(isoLayout),
		t.Format(dateLayout),
		Config.AwsRegion,
		sha256Hex([]byte(request)),
	)
}

func signature(sts string, t time.Time) string {
	kDate := hmacSha256([]byte(fmt.Sprintf("AWS4%s", Config.AwsSecretKey)), []byte(t.Format(dateLayout)))
	kRegion := hmacSha256(kDate, []byte(Config.AwsRegion))
	kService := hmacSha256(kRegion, []byte("s3"))
	kCreds := hmacSha256(kService, []byte("aws4_request"))
	return hmacSha256Hex(kCreds, []byte(sts))
}

func token(c string, t time.Time) string {
	stringToSign := stringToSign(c, t)
	sig := signature(stringToSign, t)

	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/s3/aws4_request,SignedHeaders=%s,Signature=%s",
		Config.AwsKey,
		t.Format(dateLayout),
		Config.AwsRegion,
		signedHeaders,
		sig,
	)
}

func hmacSha256(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func hmacSha256Hex(key, message []byte) string {
	return hex.EncodeToString(hmacSha256(key, message))
}

func sha256Hex(value []byte) string {
	hash := sha256.New()
	hash.Write(value)
	return hex.EncodeToString(hash.Sum(nil))
}
