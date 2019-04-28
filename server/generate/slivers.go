package generate

import (
	"encoding/json"
	"fmt"
	"sliver/server/db"
)

const (
	// sliverBucketName - Name of the bucket that stores data related to slivers
	sliverBucketName = "slivers"

	// sliverConfigNamespace - Namespace that contains sliver configs
	sliverConfigNamespace = "config"
)

// SliverConfigByName - Get a sliver's config by it's codename
func SliverConfigByName(name string) (*SliverConfig, error) {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return nil, err
	}
	rawConfig, err := bucket.Get(fmt.Sprintf("%s.%s", sliverConfigNamespace, name))
	if err != nil {
		return nil, err
	}
	config := &SliverConfig{}
	err = json.Unmarshal(rawConfig, config)
	return config, err
}

// SliverConfigSave - Save a configuration to the database
func SliverConfigSave(config *SliverConfig) error {
	bucket, err := db.GetBucket(sliverBucketName)
	if err != nil {
		return err
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = bucket.Set(fmt.Sprintf("%s.%s", sliverConfigNamespace, config.Name), rawConfig)
	return err
}
