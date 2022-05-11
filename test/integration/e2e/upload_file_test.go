package uploadfile

import (
	"bytes"
	"context"
	"strings"
	"testing"

	avro "github.com/KL-Engineering/s3-kafka-lambda/api/avro/avro_gencode"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func TestUploadFile(t *testing.T) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		GroupID:     "consumer-group-" + uuid.NewString(),
		Topic:       "S3FileCreatedUpdated",
		StartOffset: kafka.LastOffset,
	})

	content := uuid.NewString() + ",org1"
	key := uuid.NewString() + ".csv"
	bucket := "organization"

	result, err := s3Client(t).PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          strings.NewReader(content),
		ContentLength: int64(len(content)),
		ContentType:   aws.String("text/csv"),
	})

	assert.Nil(t, err, "error putting s3 object to bucket")

	t.Logf("result %v", result.ETag)
	message, err := r.ReadMessage(context.Background())

	assert.Nil(t, err, "error reading message from topic")

	val, err := avro.DeserializeS3FileCreatedUpdated(bytes.NewReader(message.Value[5:]))

	assert.Nil(t, err, "error deserializing message from topic")

	expected := s3FileCreatedUpdated(content, key, val.Metadata.Tracking_id, bucket)
	assert.Equal(t, expected, val)
}

func s3Client(t *testing.T) *s3.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID: "aws",
			URL:         "http://localhost:4566",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(customResolver))

	assert.Nil(t, err, "error creating aws config")

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return s3Client
}

func s3FileCreatedUpdated(content string, key string, trackingId string, bucket string) avro.S3FileCreatedUpdated {
	expected := avro.NewS3FileCreatedUpdated()
	payload := avro.NewS3FileCreatedUpdatedPayload()
	payload.Aws_region = "us-east-1"
	payload.Bucket_name = bucket
	payload.Content_length = int64(len(content))
	payload.Content_type = "csv"
	payload.Key = key
	payload.Operation_type = bucket
	expected.Payload = payload
	metadata := avro.NewS3FileCreatedUpdatedMetadata()
	metadata.Origin_application = "s3"
	metadata.Region = "us-east-1"
	metadata.Tracking_id = trackingId
	expected.Metadata = metadata

	return expected
}
