package s3

import (
	"devtool/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

func (r *s3Repository) ReadExportObject(ctx domain.Context, bucket, key string, limit int64) ([]byte, error) {
	if limit < 1 || limit > domain.ExportAssetMaxBytes {
		return nil, domain.ErrAgentInput
	}
	out, e := r.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if e != nil {
		return nil, e
	}
	defer out.Body.Close()
	if out.ContentLength != nil && *out.ContentLength > limit {
		return nil, domain.ErrAgentProjection
	}
	data, e := io.ReadAll(io.LimitReader(out.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(data)) > limit {
		return nil, domain.ErrAgentProjection
	}
	return data, nil
}
