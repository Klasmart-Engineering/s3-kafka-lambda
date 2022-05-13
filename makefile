all: build-all

build-all:
	@echo "Building main & generating avro code..."
	mkdir -p api/avro/avro_gencode && gogen-avro ./api/avro/avro_gencode/ ./api/avro/create_update_s3_file.avsc
	GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o main cmd/main.go && zip main.zip main

build-main:
	@echo "Building main..."
	GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o main cmd/main.go && zip main.zip main

clean:
	@echo "Cleaning up..."
	rm -rf main main.zip

local-deploy:
	@echo "Deploying locally..."
	docker-compose up -d && \
	sleep 1 && \
	aws --endpoint-url=http://localhost:4566 s3 mb s3://organization && \
	docker exec redpanda-l rpk topic create S3FileCreatedUpdated --brokers=localhost:9092 && \
	aws --endpoint-url http://localhost:4566 lambda create-function --function-name fileevent --handler main --runtime go1.x --role your-role --zip-file fileb://main.zip --environment 'Variables={KAFKA_BROKER=redpanda:29092,SCHEMA_REGISTRY=http://redpanda:8081,TOPIC_NAME=S3FileCreatedUpdated}' && \
	aws --endpoint-url=http://localhost:4566 s3api put-bucket-notification-configuration --bucket organization --notification-configuration file://scripts/local/s3-notif-config.json

local-deploy-clean:
	@echo "Local deploy teardown..."
	aws --endpoint-url=http://localhost:4566 s3 rm s3://organization && \
	aws --endpoint-url http://localhost:4566 lambda delete-function --function-name fileevent && \
	docker exec redpanda-l rpk topic delete S3FileCreatedUpdated --brokers=localhost:9092