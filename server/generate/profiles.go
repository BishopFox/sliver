package generate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sliver/server/assets"
)

const (
	profilesDirName = "profiles"
)

func getProfileDir() string {
	appDir := assets.GetRootAppDir()
	profilesDir := path.Join(appDir, profilesDirName)
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		log.Printf("Creating bin directory: %s", profilesDir)
		err = os.MkdirAll(profilesDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return profilesDir
}

// SaveProfile - Save a sliver profile to disk
func SaveProfile(name string, config *SliverConfig) error {
	filename := fmt.Sprintf("%s", filepath.Base(name))
	porfileDir := getProfileDir()
	saveTo, _ := filepath.Abs(path.Join(porfileDir, filename))
	configJSON, _ := json.Marshal(config)
	err := ioutil.WriteFile(saveTo, configJSON, 0644)
	if err != nil {
		log.Printf("Failed to write config to: %s (%v)", saveTo, err)
		return err
	}
	log.Printf("Saved profile to: %s", saveTo)
	return nil
}

// GetProfiles - List existing profile names
func GetProfiles() map[string]*SliverConfig {

	profileDir := getProfileDir()
	profileFiles, err := ioutil.ReadDir(profileDir)
	if err != nil {
		log.Printf("No profiles found %v", err)
		return map[string]*SliverConfig{}
	}

	profiles := map[string]*SliverConfig{}
	for _, porfileFileInfo := range profileFiles {
		profilePath := path.Join(profileDir, porfileFileInfo.Name())
		log.Printf("Parsing profile %s", profilePath)
		profileFile, err := os.Open(profilePath)
		if err != nil {
			log.Printf("Open failed %v", err)
			continue
		}
		data, err := ioutil.ReadAll(profileFile)
		if err != nil {
			log.Printf("Read failed %v", err)
			continue
		}
		config := SliverConfig{}
		err = json.Unmarshal(data, &config)
		if err != nil {
			log.Printf("Failed to parse profile %v", err)
			continue
		}
		profiles[porfileFileInfo.Name()] = &config
		profileFile.Close()
	}

	return profiles
}
