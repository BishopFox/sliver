package generate

import (
	"encoding/json"
	"fmt"
	insecureRand "math/rand"
	clientpb "sliver/protobuf/client"
	"sliver/server/db"
	"strings"
	"time"
)

const (
	// CanaryBucketName - DNS Canary bucket name
	CanaryBucketName = "canaries"
	canaryPrefix     = "can://"
	canarySize       = 6
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

// ToProtobuf - Return a protobuf version of the struct
func (c *DNSCanary) ToProtobuf() *clientpb.DNSCanary {
	return &clientpb.DNSCanary{
		SliverName:     c.SliverName,
		Domain:         c.Domain,
		Triggered:      c.Triggered,
		FristTriggered: c.FirstTrigger,
		LatestTrigger:  c.LatestTrigger,
		Count:          uint32(c.Count),
	}
}

func canarySubDomain() string {
	subdomain := []rune{}
	index := insecureRand.Intn(len(dnsCharSet) - 12) // ensure first char is alphabetic
	subdomain = append(subdomain, dnsCharSet[index])
	for i := 0; i < canarySize; i++ {
		index := insecureRand.Intn(len(dnsCharSet))
		subdomain = append(subdomain, dnsCharSet[index])
	}
	return string(subdomain)
}

// ListCanaries - List of all embedded canaries
func ListCanaries() ([]*DNSCanary, error) {
	bucket, err := db.GetBucket(CanaryBucketName)
	if err != nil {
		return nil, err
	}

	rawCanaries, err := bucket.Map("")
	canaries := []*DNSCanary{}
	for _, rawCanary := range rawCanaries {
		canary := &DNSCanary{}
		err := json.Unmarshal(rawCanary, canary)
		if err != nil {
			buildLog.Errorf("Failed to parse canary")
			continue
		}
		canaries = append(canaries, canary)
	}
	return canaries, nil
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

// CanaryGenerator - Holds data related to canary generation
type CanaryGenerator struct {
	SliverName    string
	ParentDomains []string
}

// GenerateCanary - Generate a canary domain and save it to the db
// 				    currently this gets called by template engine
func (g *CanaryGenerator) GenerateCanary() string {

	bucket, err := db.GetBucket(CanaryBucketName)
	if err != nil {
		buildLog.Warnf("Failed to fetch canary bucket")
		return ""
	}
	if len(g.ParentDomains) < 1 {
		buildLog.Warnf("No parent domains")
		return ""
	}

	// Don't need secure random here
	insecureRand.Seed(time.Now().UnixNano())
	index := insecureRand.Intn(len(g.ParentDomains))

	parentDomain := g.ParentDomains[index]
	if strings.HasPrefix(parentDomain, ".") {
		parentDomain = parentDomain[1:]
	}

	subdomain := canarySubDomain()
	canaryDomain := fmt.Sprintf("%s.%s", subdomain, parentDomain)
	buildLog.Infof("Generated new canary domain %s", canaryDomain)
	canary, err := json.Marshal(&DNSCanary{
		SliverName: g.SliverName,
		Domain:     canaryDomain,
		Triggered:  false,
		Count:      0,
	})
	if err != nil {
		return ""
	}
	err = bucket.Set(canaryDomain, canary)
	if err != nil {
		buildLog.Errorf("Failed to save canary %s", err)
		return ""
	}
	return fmt.Sprintf("%s%s", canaryPrefix, canaryDomain)
}
