package s3

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"devtool/internal/domain"
)

type s3Repository struct {
	client        *s3.Client
	presignClient *s3.PresignClient
}

func NewS3Repository(client *s3.Client) domain.S3Repository {
	return &s3Repository{
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}
}

func (r *s3Repository) PresignUpload(ctx domain.Context, bucket, key string, expires time.Duration) (string, error) {
	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *s3Repository) PresignDownload(ctx domain.Context, bucket, key string, expires time.Duration) (string, error) {
	req, err := r.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *s3Repository) Delete(ctx domain.Context, bucket, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (r *s3Repository) PresignUploadConstrained(ctx domain.Context, bucket, key, mime string, size int64, deadline time.Time) (string, map[string]string, error) {
	expires := time.Until(deadline)
	if expires < time.Second || expires > domain.AssetCapabilityLifetime {
		return "", nil, errors.New("upload URL lifetime must be between one and sixty seconds")
	}
	req, e := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), ContentType: aws.String(mime), ContentLength: aws.Int64(size)}, func(o *s3.PresignOptions) { o.Expires = expires })
	if e != nil {
		return "", nil, e
	}
	if e = validateSignedDeadline(req.URL, deadline); e != nil {
		return "", nil, e
	}
	return req.URL, map[string]string{"Content-Type": mime}, nil
}

func (r *s3Repository) PresignDownloadUntil(ctx domain.Context, bucket, key string, deadline time.Time) (string, error) {
	expires := time.Until(deadline)
	if expires < time.Second || expires > domain.AssetCapabilityLifetime {
		return "", domain.ErrInvalidSceneSession
	}
	signed, e := r.PresignDownload(ctx, bucket, key, expires)
	if e != nil {
		return "", e
	}
	if e = validateSignedDeadline(signed, deadline); e != nil {
		return "", e
	}
	return signed, nil
}

// SDK credential retrieval and middleware can block between TTL calculation and
// signature creation. Never release a URL whose actual signed expiry drifts.
func validateSignedDeadline(signed string, deadline time.Time) error {
	if time.Until(deadline) < time.Second {
		return domain.ErrInvalidSceneSession
	}
	parsed, e := url.Parse(signed)
	if e != nil {
		return domain.ErrInvalidCapability
	}
	signedAt, e := time.Parse("20060102T150405Z", parsed.Query().Get("X-Amz-Date"))
	if e != nil {
		return domain.ErrInvalidCapability
	}
	seconds, e := strconv.ParseInt(parsed.Query().Get("X-Amz-Expires"), 10, 64)
	if e != nil || seconds < 1 || seconds > int64(domain.AssetCapabilityLifetime/time.Second) {
		return domain.ErrInvalidCapability
	}
	if signedAt.Add(time.Duration(seconds) * time.Second).After(deadline) {
		return domain.ErrInvalidSceneSession
	}
	return nil
}
