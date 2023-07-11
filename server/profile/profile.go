package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boojack/slash/server/version"
	"github.com/spf13/viper"
)

// Profile is the configuration to start main server.
type Profile struct {
	// Data is the data directory
	Data string `json:"-"`
	// DSN points to store data
	DSN string `json:"-"`
	// Mode can be "prod" or "dev"
	Mode string `json:"mode"`
	// Port is the binding port for server
	Port int `json:"port"`
	// Version is the current version of server
	Version string `json:"version"`
}

func checkDSN(dataDir string) (string, error) {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + dataDir)
		if err != nil {
			return "", err
		}
		dataDir = absDir
	}

	// Trim trailing / in case user supplies
	dataDir = strings.TrimRight(dataDir, "/")

	if _, err := os.Stat(dataDir); err != nil {
		return "", fmt.Errorf("unable to access data folder %s, err %w", dataDir, err)
	}

	return dataDir, nil
}

// GetDevProfile will return a profile for dev or prod.
func GetProfile() (*Profile, error) {
	profile := Profile{}
	err := viper.Unmarshal(&profile)
	if err != nil {
		return nil, err
	}

	if profile.Mode != "dev" && profile.Mode != "prod" {
		profile.Mode = "dev"
	}

	if profile.Mode == "prod" && profile.Data == "" {
		profile.Data = "/var/opt/slash"
	}

	dataDir, err := checkDSN(profile.Data)
	if err != nil {
		fmt.Printf("Failed to check dsn: %s, err: %+v\n", dataDir, err)
		return nil, err
	}

	profile.Data = dataDir
	profile.DSN = fmt.Sprintf("%s/slash_%s.db", dataDir, profile.Mode)
	profile.Version = version.GetCurrentVersion(profile.Mode)
	return &profile, nil
}
