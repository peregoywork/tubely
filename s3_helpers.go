package main

import (
	"io"
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func uploadToBucket(c context.Context, s3Client *s3.Client, bucket, key, mediaType string, body io.Reader) (*s3.PutObjectOutput, error) {
	return s3Client.PutObject(c, &s3.PutObjectInput{
		Bucket: &bucket,
		Key: &key,
		Body: body,
		ContentType: &mediaType,
	})
}

// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
// 	psClient := s3.NewPresignClient(s3Client
// }
