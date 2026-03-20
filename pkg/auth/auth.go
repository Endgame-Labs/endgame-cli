package auth

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
)

type Config struct {
	APIKey string `json:"api_key,omitempty"`
	OrgID  string `json:"org_id,omitempty"`
}

func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".endgame-auth.json"), nil
}

func Save(config Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("not authenticated")
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadCredentials() (string, string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ENDGAME_API_KEY"))
	orgID := strings.TrimSpace(os.Getenv("ENDGAME_ORG_ID"))

	if apiKey != "" && orgID != "" {
		return apiKey, orgID, nil
	}

	config, err := Load()
	if err != nil {
		if apiKey != "" || orgID != "" {
			return apiKey, orgID, nil
		}
		return "", "", err
	}

	if apiKey == "" {
		apiKey = strings.TrimSpace(config.APIKey)
	}
	if orgID == "" {
		orgID = strings.TrimSpace(config.OrgID)
	}

	return apiKey, orgID, nil
}

func promptInput(label string) (string, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func Login() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	fmt.Println("Endgame Authentication")
	fmt.Println()
	fmt.Printf("Credentials will be stored in: %s\n", configPath)
	fmt.Println()

	apiKey, err := promptInput("Enter your Endgame API key: ")
	if err != nil {
		return err
	}
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	orgID, err := promptInput("Enter your Endgame org ID: ")
	if err != nil {
		return err
	}
	if orgID == "" {
		return errors.New("org ID cannot be empty")
	}

	client, err := endgame.NewClient(apiKey, orgID)
	if err != nil {
		return err
	}
	if err := client.Initialize(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	if _, err := client.ListTools(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	if err := Save(Config{APIKey: apiKey, OrgID: orgID}); err != nil {
		return err
	}

	fmt.Printf("\nAuthenticated for org %s\n", orgID)
	return nil
}

func Verify() (string, error) {
	apiKey, orgID, err := LoadCredentials()
	if err != nil {
		return "", err
	}

	client, err := endgame.NewClient(apiKey, orgID)
	if err != nil {
		return "", err
	}

	if err := client.Initialize(); err != nil {
		return "", err
	}
	if _, err := client.ListTools(); err != nil {
		return "", err
	}

	return client.OrgID(), nil
}

func Logout() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
