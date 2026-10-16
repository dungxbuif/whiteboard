package s3

import (
	"errors"
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

func (r *s3Repository) PresignUploadConstrained(ctx domain.Context, bucket, key, mime string, size int64, expires time.Duration) (string, map[string]string, error) {
	if expires < time.Second || expires > domain.AssetCapabilityLifetime {
		return "", nil, errors.New("upload URL lifetime must be between one and sixty seconds")
	}
	req, e := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), ContentType: aws.String(mime), ContentLength: aws.Int64(size)}, func(o *s3.PresignOptions) { o.Expires = expires })
	if e != nil {
		return "", nil, e
	}
	return req.URL, map[string]string{"Content-Type": mime}, nil
}
