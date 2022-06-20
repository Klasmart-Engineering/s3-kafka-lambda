# s3-kafka-lambda

## Running locally

prerequisite

- Installed gogen-avro to generate avro `go install github.com/actgardner/gogen-avro/v10/cmd/...@latest`
- Create your file
- Installed kcat to read avro messages from topic `brew install kcat`

Build and local deployment to localstack
`make && make local-deploy`
  
Put your created file to bucket
`aws --endpoint-url=http://localhost:4566 s3api put-object --bucket my-bucket --key newfile.csv --body=newfile.csv`

Logs for the lambda
`aws --endpoint-url=http://localhost:4566 logs tail '/aws/lambda/fileevent' --follow`

Get avro messages from topic
`kcat -b localhost:9092 -t S3FileCreatedUpdated  -s avro  -r http://localhost:8081`

## makefile

`build` generates avro structs, compiles go code and zip.
`clean` removes go build artifacts.
`local-deploy` starts Red Panda, creates s3 bucket, Red Panda topic, deploys lambda-function (requires `build`) & s3 notification for lambda-function.
`local-deploy-clean` deletes; s3 bucket, Red Panda topic and lambda-function.

## Operation mapping env
Provide this env variable bucket to operation type mapping when deploying the lambda

```
OPERATIONS = "{\"organization\":\"ORGANIZATION\",\"school\":\"SCHOOL\",\"user\":\"USER\",\"class\":\"CLASS\",\"organization-membership\":\"ORGANIZATION_MEMBERSHIP\",\"class-details\":\"CLASS_DETAILS\",\"school-membership\":\"SCHOOL_MEMBERSHIP\",\"class-roster\":\"CLASS_ROSTER\"}
```