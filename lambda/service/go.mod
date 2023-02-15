module github.com/pennsieve/publishing-service/service

go 1.18

replace github.com/pennsieve/publishing-service/api => ../../api

require (
	github.com/aws/aws-lambda-go v1.34.1
	github.com/pennsieve/publishing-service/api v0.0.0-20230214163953-790952f66bb6 // replaced (see above)
	github.com/sirupsen/logrus v1.9.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.17.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.12 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.9.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.22 // indirect
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.1.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.29.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.17.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.3 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pennsieve/pennsieve-go-api v1.4.4 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)
