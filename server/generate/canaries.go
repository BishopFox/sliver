package generate

import (
	"errors"
	"fmt"
	insecureRand "math/rand"
	"sliver/server/db"
	"strings"
	"time"
)

const (
	// CanaryBucketName - DNS Canary bucket name
	CanaryBucketName = "canaries"

	canarySize = 6
)

var (
	dnsCharSet = []rune("abcdefghijklmnopqrstuvwxyz0123456789-_")
)

func canarySubDomain() string {
	insecureRand.Seed(time.Now().UnixNano())
	subdomain := []rune{}
	for i := 0; i < canarySize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		subdomain = append(subdomain, dnsCharSet[index])
	}
	return string(subdomain)
}

// generateCanaryDomain - Generate a canary domain and save it to the db
func generateCanaryDomain(sliverName string, parentDomain string) (string, error) {
	bucket, err := db.GetBucket(CanaryBucketName)
	if err != nil {
		return "", err
	}
	if len(parentDomain) < 3 {
		return "", errors.New("Invalid parent domain")
	}
	if strings.HasPrefix(parentDomain, ".") {
		parentDomain = parentDomain[1:]
	}

	subdomain := canarySubDomain()
	canaryDomain := fmt.Sprintf("%s.%s", subdomain, parentDomain)
	bucket.Set(canaryDomain, []byte(sliverName))
	return canaryDomain, nil
}
