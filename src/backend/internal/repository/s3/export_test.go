package s3

import (
	"context"
	"devtool/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExportObjectBoundedRead(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/bucket/server-owned-key" {
			t.Errorf("unexpected object path %s", r.URL.Path)
		}
		w.Write([]byte("12345"))
	}))
	defer server.Close()
	client := sdk.New(sdk.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "secret", ""), BaseEndpoint: aws.String(server.URL), UsePathStyle: true})
	repo := NewS3Repository(client).(domain.ExportObjectReader)
	if b, e := repo.ReadExportObject(context.Background(), "bucket", "server-owned-key", 5); e != nil || string(b) != "12345" {
		t.Fatalf("read %q %v", b, e)
	}
	if _, e := repo.ReadExportObject(context.Background(), "bucket", "server-owned-key", 4); e == nil {
		t.Fatal("oversize accepted")
	}
	if _, e := repo.ReadExportObject(context.Background(), "bucket", "server-owned-key", domain.ExportAssetMaxBytes+1); e == nil || hits != 2 {
		t.Fatal("invalid byte limit reached storage")
	}
}
