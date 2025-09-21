package backup

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BackupItem struct {
	Source      string
	Destination string
}

type Backup struct {
	Targets  []BackupItem
	confPath string
	Logger   *slog.Logger
}

func NewBackup(targets string, logger *slog.Logger) (*Backup, error) {
	items, err := parseBackupItems(targets)
	if err != nil {
		return nil, err
	}

	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	confPath := filepath.Join(os.TempDir(), "rclone.conf")

	err = os.WriteFile(confPath, []byte(config), 0644)
	if err != nil {
		return nil, err
	}

	if logger != nil {
		logger.Info("Backup config initialized")
	}

	return &Backup{Targets: items, confPath: confPath, Logger: logger}, nil
}

func (b *Backup) Backup(isFatal bool) error {
	for _, item := range b.Targets {
		if err := b.BackupItem(item); err != nil {
			if isFatal {
				return err
			} else {
				b.Logger.Error("Backup failed", "err", err)
			}
		}
	}
	return nil
}

func (b *Backup) BackupItem(item BackupItem) error {
	b.Logger.Info("Backup started", "source", item.Source, "destination", item.Destination)
	cmd := exec.Command("rclone", "sync", "--links", item.Source, fmt.Sprintf("%s:%s", "gdrive", item.Destination))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RCLONE_CONFIG=%s", b.confPath))
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute rclone: %v, %s", err, stdout)
	}

	b.Logger.Info("Backup completed successfully", "source", item.Source, "destination", item.Destination, "stdout", stdout)

	return nil
}

func getConfig() (string, error) {
	baseTemplate := `
[gdrive]
type = drive
client_id = #client_id#
client_secret = #client_secret#
scope = drive
token = #token#
team_drive = 
`

	creds, err := getCredentials()
	if err != nil {
		return "", err
	}

	token, err := getToken()
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(baseTemplate, "#client_id#", creds.ClientID), "#client_secret#", creds.ClientSecret), "#token#", token), nil
}

type Credentials struct {
	ClientID     string
	ClientSecret string
}

func getCredentials() (*Credentials, error) {
	file, err := os.Open("credentials/credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %v", err)
	}
	defer file.Close()

	var creds struct {
		Installed struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		} `json:"installed"`
	}
	if err := json.NewDecoder(file).Decode(&creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials file: %v", err)
	}

	return &Credentials{
		ClientID:     creds.Installed.ClientID,
		ClientSecret: creds.Installed.ClientSecret,
	}, nil
}

func getToken() (string, error) {
	file, err := os.ReadFile("credentials/token.json")
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %v", err)
	}
	return string(file), nil
}

func parseBackupItems(targetsString string) ([]BackupItem, error) {
	if targetsString == "" {
		return nil, fmt.Errorf("no backup targets specified")
	}
	targets := strings.Split(targetsString, ",")
	if len(targets) < 1 {
		return nil, fmt.Errorf("no backup targets specified")
	}
	var items []BackupItem
	for _, target := range targets {
		parts := strings.SplitN(target, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid backup target format: %s", target)
		}
		items = append(items, BackupItem{
			Source:      parts[0],
			Destination: parts[1],
		})
	}
	return items, nil
}
