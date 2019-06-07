package cloudflare

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/bishopfox/sliver/server/log"
)

var (
	awsLog = log.NamedLogger("infrastruture", "aws")
)

func getAWSCredentials() *credentials.Credentials {
	creds := credentials.NewEnvCredentials()
	_, err := creds.Get()
	if err != nil {
		return nil
	}
	return creds
}
