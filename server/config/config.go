package config

import (
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
)

const (
	configBucketName = "config"
)

var (
	configRootLog = log.NamedLogger("config", "root")
)

// SetConfig - Set config key/value pair
func SetConfig(key, value string) error {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return err
	}
	return bucket.Set(key, []byte(value))
}

// GetConfig - Get config value
func GetConfig(key string) (string, error) {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return "", err
	}
	value, err := bucket.Get(key)
	return string(value), err
}

// ListConfig - List config contents
func ListConfig() (map[string]string, error) {
	bucket, err := db.GetBucket(configBucketName)
	if err != nil {
		return nil, err
	}
	config, err := bucket.Map("")
	if err != nil {
		return nil, err
	}
	// Convert []byte's to strings
	configStr := map[string]string{}
	for key, value := range config {
		configStr[key] = string(value)
	}
	return configStr, nil
}
