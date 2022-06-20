package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
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
	var operations map[string]interface{}
	err = json.Unmarshal([]byte(os.Getenv("OPERATIONS")), &operations)
	if err != nil {
		logger.Errorf("Failed to retrieve operation mappings - %v", err)
		panic("Failed to retrieve operation mappings")
	}
	if len(operations) == 0 {
		logger.Errorf("No operation mappings")
		panic("No operation mappings")
	}
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
	messages = mapToMessages(s3Event, operations, schemaIDBytes, messages, logger)

	err = writer.WriteMessages(
		context.Background(),
		messages...,
	)

	if err != nil {
		logger.Errorf("Failed to write to kafka - %v", err)

		panic("Failed to write to kafka")
	}
}

func mapToMessages(s3Event events.S3Event, operations map[string]interface{}, schemaIDBytes []byte, messages []kafka.Message, logger *zap.SugaredLogger) []kafka.Message {
	for _, record := range s3Event.Records {
		op := operations[record.S3.Bucket.Name]
		if op == nil {
			logger.Errorf("Failed to get operation and skipping for %v", record.S3.Object.Key)
			continue
		}
		if _, ok := op.(string); ok {
			operation := op.(interface{}).(string)
			message := message(record, operation)
			kafkaMessage, err := kafkaMessage(message, schemaIDBytes)
			if err != nil {
				logger.Errorf("Failed to serialize to kafka message for %v - %v", record.S3.Object.Key, err)
				continue
			}
			messages = append(messages, kafkaMessage)
		} else {
			logger.Errorf("Skipping for %v as operation mapping value is not string", record.S3.Object.Key)
		}
	}
	return messages
}

func newLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	defer func(logger *zap.Logger) {
		logger.Sync()
	}(logger)
	sugar := logger.Sugar()
	return sugar, nil
}

func message(record events.S3EventRecord, operation string) avro.S3FileCreatedUpdated {
	s3 := record.S3
	metadata := avro.NewS3FileCreatedUpdatedMetadata()
	metadata.Origin_application = "s3"
	metadata.Tracking_uuid = uuid.New().String()
	metadata.Region = record.AWSRegion

	payload := avro.NewS3FileCreatedUpdatedPayload()
	payload.Aws_region = record.AWSRegion
	payload.Bucket_name = s3.Bucket.Name
	payload.Content_length = s3.Object.Size
	payload.Key = s3.Object.Key
	payload.Content_type = fileType(s3.Object.Key)
	payload.Operation_type = operation

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

func kafkaMessage(payload avro.S3FileCreatedUpdated, schemaIDBytes []byte) (kafka.Message, error) {
	var buf bytes.Buffer
	err := payload.Serialize(&buf)
	if err != nil {
		return kafka.Message{}, err
	}
	valueBytes := buf.Bytes()

	var recordValue []byte
	recordValue = append(recordValue, byte(0))
	recordValue = append(recordValue, schemaIDBytes...)
	recordValue = append(recordValue, valueBytes...)

	return kafka.Message{
		Value: recordValue,
	}, nil
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
