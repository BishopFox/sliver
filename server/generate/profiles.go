package generate

import (
	"encoding/json"
	"sliver/server/db"
)

const (
	profilesBucketName = "profiles"
)

// ProfileSave - Save a sliver profile to disk
func ProfileSave(name string, config *SliverConfig) error {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return bucket.Set(name, configJSON)
}

// ProfileByName - Fetch a single profile from the database
func ProfileByName(name string) (*SliverConfig, error) {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return nil, err
	}
	rawProfile, err := bucket.Get(name)
	config := &SliverConfig{}
	err = json.Unmarshal(rawProfile, config)
	return config, err
}

// Profiles - Fetch a map of name<->profiles current in the database
func Profiles() map[string]*SliverConfig {
	bucket, err := db.GetBucket(profilesBucketName)
	if err != nil {
		return nil
	}
	rawProfiles, err := bucket.Map("")
	if err != nil {
		return nil
	}

	profiles := map[string]*SliverConfig{}
	for name, rawProfile := range rawProfiles {
		config := &SliverConfig{}
		err := json.Unmarshal(rawProfile, config)
		if err != nil {
			continue // We should probably log these failures ...
		}
		profiles[name] = config
	}
	return profiles
}
