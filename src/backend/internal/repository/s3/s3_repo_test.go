package s3

import (
	"context"
	"devtool/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPublicUploadCapabilitySignedConstraints(t *testing.T) {
	client := sdk.New(sdk.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "test-secret", ""), BaseEndpoint: aws.String("http://localhost:9000"), UsePathStyle: true})
	repo := NewS3Repository(client).(domain.ScopedUploadRepository)
	raw, h, e := repo.PresignUploadConstrained(context.Background(), "bucket", "key", "image/png", 7, time.Minute)
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
		if raw, _, e := repo.PresignUploadConstrained(context.Background(), "bucket", "key", "image/png", 7, ttl); e == nil || raw != "" {
			t.Fatalf("unsafe TTL %v produced capability", ttl)
		}
	}
}
