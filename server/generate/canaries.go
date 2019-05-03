package generate

import (
	"encoding/json"
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

// DNSCanary - DNS canary
type DNSCanary struct {
	SliverName    string `json:"sliver_name"`
	Domain        string `json:"domain"`
	Triggered     bool   `json:"triggered"`
	FirstTrigger  string `json:"first_trigger"`
	LatestTrigger string `json:"latest_trigger"`
	Count         int    `json:"count"`
}

func canarySubDomain() string {
	insecureRand.Seed(time.Now().UnixNano())
	subdomain := []rune{}
	for i := 0; i < canarySize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		subdomain = append(subdomain, dnsCharSet[index])
	}
	return string(subdomain)
}

// CheckCanary - Check if a canary exists
func CheckCanary(domain string) (*DNSCanary, error) {
	bucket, err := db.GetBucket(CanaryBucketName)
	if err != nil {
		return nil, err
	}
	data, err := bucket.Get(domain)
	if err != nil {
		return nil, err
	}
	canary := &DNSCanary{}
	err = json.Unmarshal(data, canary)
	return canary, err
}

// UpdateCanary - Update an existing canary
func UpdateCanary(canary *DNSCanary) error {
	bucket, err := db.GetBucket(CanaryBucketName)
	if err != nil {
		return err
	}
	canaryData, err := json.Marshal(canary)
	if err != nil {
		return err
	}
	return bucket.Set(canary.Domain, canaryData)
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
	canary, err := json.Marshal(&DNSCanary{
		SliverName: sliverName,
		Domain:     canaryDomain,
		Triggered:  false,
		Count:      0,
	})
	if err != nil {
		return "", err
	}
	err = bucket.Set(canaryDomain, canary)
	return canaryDomain, err
}
