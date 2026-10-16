package s3

import (
	"context"
	"devtool/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPublicUploadCapabilitySignedConstraints(t *testing.T) {
	client := sdk.New(sdk.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "test-secret", ""), BaseEndpoint: aws.String("http://localhost:9000"), UsePathStyle: true})
	repo := NewS3Repository(client).(domain.ScopedUploadRepository)
	raw, h, e := repo.PresignUploadConstrained(context.Background(), "bucket", "key", "image/png", 7, time.Now().Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(raw)
	signed := u.Query().Get("X-Amz-SignedHeaders")
	if !strings.Contains(signed, "content-type") || !strings.Contains(signed, "content-length") {
		t.Fatalf("upload constraints not signed: %s", signed)
	}
	if h["Content-Type"] != "image/png" || h["Content-Length"] != "" {
		t.Fatalf("browser headers: %+v", h)
	}
}

func TestPublicUploadCapabilityNoUnboundedTTL(t *testing.T) {
	client := sdk.New(sdk.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "test-secret", ""), BaseEndpoint: aws.String("http://localhost:9000"), UsePathStyle: true})
	repo := NewS3Repository(client).(domain.ScopedUploadRepository)
	for _, ttl := range []time.Duration{0, -time.Second, time.Millisecond, 2 * time.Minute} {
		if raw, _, e := repo.PresignUploadConstrained(context.Background(), "bucket", "key", "image/png", 7, time.Now().Add(ttl)); e == nil || raw != "" {
			t.Fatalf("unsafe TTL %v produced capability", ttl)
		}
	}
}

type slowCapabilityCredentials struct{}

func (slowCapabilityCredentials) Retrieve(context.Context) (aws.Credentials, error) {
	time.Sleep(1500 * time.Millisecond)
	return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test-secret"}, nil
}
func TestAssetSignerAbsoluteDeadline(t *testing.T) {
	for _, mode := range []string{"upload", "view"} {
		t.Run(mode, func(t *testing.T) {
			client := sdk.New(sdk.Options{Region: "us-east-1", Credentials: slowCapabilityCredentials{}, BaseEndpoint: aws.String("http://localhost:9000"), UsePathStyle: true})
			repo := NewS3Repository(client)
			deadline := time.Now().Add(3 * time.Second).Truncate(time.Second)
			var raw string
			var e error
			if mode == "upload" {
				raw, _, e = repo.(domain.ScopedUploadRepository).PresignUploadConstrained(context.Background(), "bucket", "key", "image/png", 7, deadline)
			} else {
				raw, e = repo.(domain.ScopedDownloadRepository).PresignDownloadUntil(context.Background(), "bucket", "key", deadline)
			}
			if e == nil {
				u, _ := url.Parse(raw)
				date, pe := time.Parse("20060102T150405Z", u.Query().Get("X-Amz-Date"))
				if pe != nil {
					t.Fatal(pe)
				}
				seconds, pe := strconv.ParseInt(u.Query().Get("X-Amz-Expires"), 10, 64)
				if pe != nil {
					t.Fatal(pe)
				}
				if date.Add(time.Duration(seconds) * time.Second).After(deadline) {
					t.Fatalf("signer emitted URL expiring after deadline: %s", raw)
				}
			} else if raw != "" {
				t.Fatal("failed signing leaked URL")
			}
		})
	}
}
