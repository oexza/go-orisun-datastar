package storage

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Provider interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	PublicURL(key string) string
}

type NoopProvider struct{}

func (NoopProvider) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	return nil
}

func (NoopProvider) PublicURL(key string) string {
	return "/static/" + key
}

type S3Provider struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewS3Provider(ctx context.Context, endpoint, accessKey, secretKey, bucket, publicURL, region string, forcePathStyle bool) (*S3Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = forcePathStyle
	})
	return &S3Provider{client: client, bucket: bucket, publicURL: publicURL}, nil
}

func (p *S3Provider) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

func (p *S3Provider) PublicURL(key string) string {
	return p.publicURL + "/" + key
}
