package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"regexp"

	avro "github.com/KL-Engineering/s3-kafka-lambda/api/avro/avro_gencode"
	"go.uber.org/zap"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
)

func handler(ctx context.Context, s3Event events.S3Event) {
	logger, err := newLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger - %v", err)
		panic("Failed to initialize logger")
	}
	topicName := os.Getenv("TOPIC_NAME")
	schemaIDBytes, err := schemaIdBytes(topicName)
	if err != nil {
		logger.Errorf("Failed to get schema - %v", err)
		panic("Failed to get schema")
	}

	writer := kafka.Writer{
		Addr:   kafka.TCP(os.Getenv("KAFKA_BROKER")),
		Topic:  topicName,
		Logger: log.New(os.Stdout, "kafka writer: ", 0),
	}

	var messages []kafka.Message
	for _, record := range s3Event.Records {
		message := message(record)
		messages = append(messages, kafkaMessage(message, schemaIDBytes))
	}

	err = writer.WriteMessages(
		context.Background(),
		messages...,
	)

	if err != nil {
		logger.Errorf("Failed to write to kafka - %v", err)

		panic("Failed to write to kafka")
	}
}

func newLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	return sugar, nil
}

func message(record events.S3EventRecord) avro.S3FileCreatedUpdated {
	s3 := record.S3
	metadata := avro.NewS3FileCreatedUpdatedMetadata()
	metadata.Origin_application = "s3"
	metadata.Tracking_id = uuid.New().String()
	metadata.Region = record.AWSRegion

	payload := avro.NewS3FileCreatedUpdatedPayload()
	payload.Aws_region = record.AWSRegion
	payload.Bucket_name = s3.Bucket.Name
	payload.Content_length = s3.Object.Size
	payload.Key = s3.Object.Key
	payload.Content_type = fileType(s3.Object.Key)
	payload.Operation_type = s3.Bucket.Name

	message := avro.NewS3FileCreatedUpdated()
	message.Payload = payload
	message.Metadata = metadata

	return message
}

func schemaIdBytes(topicName string) ([]byte, error) {
	client := srclient.CreateSchemaRegistryClient(os.Getenv("SCHEMA_REGISTRY"))
	schema, err := client.CreateSchema(topicName, avro.NewS3FileCreatedUpdated().Schema(), "AVRO")

	if err != nil {
		return nil, err
	}
	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, uint32(schema.ID()))

	return schemaIDBytes, nil
}

func kafkaMessage(payload avro.S3FileCreatedUpdated, schemaIDBytes []byte) kafka.Message {
	var buf bytes.Buffer
	payload.Serialize(&buf)
	valueBytes := buf.Bytes()

	var recordValue []byte
	recordValue = append(recordValue, byte(0))
	recordValue = append(recordValue, schemaIDBytes...)
	recordValue = append(recordValue, valueBytes...)

	return kafka.Message{
		Value: recordValue,
	}
}

func fileType(key string) string {
	var re = regexp.MustCompile(`(?m)\b[.](?P<filetype>\w+)$`)

	matches := re.FindStringSubmatch(key)
	if matches == nil {
		return key
	}
	fileTypeIndex := re.SubexpIndex("filetype")

	if fileTypeIndex < 0 {
		return key
	}
	return matches[fileTypeIndex]
}

func main() {
	lambda.Start(handler)
}
