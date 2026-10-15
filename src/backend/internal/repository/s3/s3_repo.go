package s3

import (
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
