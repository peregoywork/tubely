package main

import (
	"io"
	"time"
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

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	psClient := s3.NewPresignClient(s3Client)
	preHTTP, err := psClient.PresignGetObject( // (*v4.PresignedHTTPRequest, error)
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key: &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}

	return preHTTP.URL, nil
}

