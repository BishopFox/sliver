package anthropic

import (
	"cmp"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/auth/bearer"
)

func bedrockBasicAuthConfig(apiKey string) aws.Config {
	return aws.Config{
		Region:                  cmp.Or(os.Getenv("AWS_REGION"), "us-east-1"),
		BearerAuthTokenProvider: bearer.StaticTokenProvider{Token: bearer.Token{Value: apiKey}},
	}
}

func bedrockPrefixModelWithRegion(modelID string) string {
	region := os.Getenv("AWS_REGION")
	if len(region) < 2 {
		region = "us-east-1"
	}
	prefix := region[:2] + "."
	if strings.HasPrefix(modelID, prefix) {
		return modelID
	}
	return prefix + modelID
}
