package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
}
type R2 struct {
	bucket    string
	publicURL string
	client    *s3.PresignClient
}

func New(ctx context.Context, cfg Config) (*R2, error) {
	if strings.TrimSpace(cfg.AccountID) == "" || strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" || strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("storage: incomplete R2 configuration")
	}
	base := s3.New(s3.Options{Region: "auto", BaseEndpoint: aws.String("https://" + cfg.AccountID + ".r2.cloudflarestorage.com"), Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")})
	return &R2{bucket: cfg.Bucket, publicURL: strings.TrimRight(cfg.PublicURL, "/"), client: s3.NewPresignClient(base)}, nil
}
func (r *R2) PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (url string, err error) {
	key = strings.TrimLeft(strings.TrimSpace(key), "/")
	if key == "" {
		return "", fmt.Errorf("storage: key is required")
	}
	if expires <= 0 {
		expires = 15 * time.Minute
	}
	req, e := r.client.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key), ContentType: aws.String(contentType)}, s3.WithPresignExpires(expires))
	if e != nil {
		return "", e
	}
	return req.URL, nil
}
func (r *R2) PublicURL(key string) string {
	key = strings.TrimLeft(key, "/")
	if r.publicURL == "" {
		return ""
	}
	return r.publicURL + "/" + key
}
func (r *R2) Bucket() string { return r.bucket }
