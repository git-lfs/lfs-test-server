package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

var (
	// TODO Might need a way to remove content-md5 if the client doesn't give us one
	signedHeaders = "content-md5;date;host;x-amz-content-sha256"
)

const (
	dateLayout    = "20060102"
	iso8601Layout = "20060102T000000Z"
)

func S3Token(method, path, sha, md5 string, t time.Time) string {
	canonicalRequest := CanonicalRequest(method, path, sha, md5, t)
	stringToSign := StringToSign(canonicalRequest, t)
	signature := Signature(stringToSign, t)

	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/s3/aws4_request,SignedHeaders=%s,Signature=%s",
		Config.AwsKey,
		date(t),
		Config.AwsRegion,
		signedHeaders,
		signature,
	)
}

func Signature(sts string, t time.Time) string {
	kDate := hmacSha256([]byte(fmt.Sprintf("AWS4%s", Config.AwsSecretKey)), []byte(date(t)))
	kRegion := hmacSha256(kDate, []byte(Config.AwsRegion))
	kService := hmacSha256(kRegion, []byte("s3"))
	kCreds := hmacSha256(kService, []byte("aws4_request"))
	return hmacSha256Hex(kCreds, []byte(sts))
}

func StringToSign(request string, t time.Time) string {
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/s3/aws4_request\n%s",
		httpdate(t),
		date(t),
		Config.AwsRegion,
		sha256Hex([]byte(request)),
	)
}

func CanonicalRequest(method, path, sha, md5 string, t time.Time) string {
	return fmt.Sprintf("%s\n%s\n\ncontent-md5:%s\ndate:%s\nhost:%s.s3.amazonaws.com\nx-amz-content-sha256:%s\n\n%s\n%s",
		method,
		path,
		encodedMd5(md5),
		httpdate(t),
		Config.AwsBucket,
		sha,
		signedHeaders,
		sha,
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

func date(t time.Time) string {
	return t.Format(dateLayout)
}

func httpdate(t time.Time) string {
	return t.Format(time.RFC1123)
}

func iso8601(t time.Time) string {
	return t.Format(iso8601Layout)
}

func sha256Hex(value []byte) string {
	hash := sha256.New()
	hash.Write(value)
	return hex.EncodeToString(hash.Sum(nil))
}

// Takes a hex encoded representation of an md5sum, decodes and base64 encodes it
func encodedMd5(value string) string {
	v, _ := hex.DecodeString(value) // TODO: handle the error
	return base64.StdEncoding.EncodeToString(v)
}
