package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go_drive_backup/backup"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	logger := slog.Default()

	err := godotenv.Load()
	if err != nil {
		logger.Debug(".env file can't be read", "err", err)
	}

	cmd := &cli.Command{
		Name:  "go_drive_backup",
		Usage: "Backup to Google Drive",
		Commands: []*cli.Command{
			{
				Name:  "auth",
				Usage: "Authorize with Google Drive",
				Action: func(context.Context, *cli.Command) error {
					return cmdAuth()
				},
			},
			{
				Name:  "check-auth",
				Usage: "Check existing authentication",
				Action: func(context.Context, *cli.Command) error {
					return cmdCheckAuth()
				},
			},
			{
				Name:  "backup",
				Usage: "Run backup tasks",
				Action: func(context.Context, *cli.Command) error {
					return cmdBackup(true)
				},
			},
			{
				Name:  "schedule",
				Usage: "Schedule backup tasks",
				Flags: []cli.Flag{
					&cli.Int16Flag{
						Name:     "interval",
						Usage:    "Interval (Seconds) for scheduled backups",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cmdSchedule(cmd.Int16("interval"))
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func cmdCheckAuth() error {
	ctx := context.TODO()

	config, err := getConfig()
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	client, err := getClient(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %w", err)
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to retrieve Drive client: %w", err)
	}

	about, err := srv.About.Get().Fields("user").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve about info: %w", err)
	}

	fmt.Printf("Logged in as %s (%s)\n", about.User.DisplayName, about.User.EmailAddress)
	return nil
}

func cmdAuth() error {
	logger := slog.Default()

	ctx := context.TODO()

	config, err := getConfig()
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	token, err := getTokenFromWeb(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("unable to marshal token to JSON: %w", err)
	}

	err = os.WriteFile("credentials/token.json", data, 0660)
	if err != nil {
		return fmt.Errorf("unable to save token to file: %w", err)
	}

	logger.Info("Token saved to credentials/token.json")
	return err
}

func cmdBackup(stopOnError bool) error {
	logger := slog.Default()
	backupLogger := logger.With("system", "backup")

	backup, err := backup.NewBackup(os.Getenv("BACKUP_TARGETS"), backupLogger)
	if err != nil {
		return fmt.Errorf("unable to create backup instance: %w", err)
	}

	logger.Info("Backup started")
	err = backup.Backup(stopOnError)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	logger.Info("Backup completed successfully")
	return nil
}

func cmdSchedule(interval int16) error {
	logger := slog.Default()
	for {
		err := cmdBackup(false)
		if err != nil {
			logger.Error("Backup failed", "err", err)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func getConfig() (*oauth2.Config, error) {
	credentials, err := os.ReadFile("credentials/credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %v", err)
	}

	return google.ConfigFromJSON(credentials, drive.DriveScope)
}

func getToken() (*oauth2.Token, error) {
	tokenFile, err := os.Open("credentials/token.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %v", err)
	}
	defer tokenFile.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(tokenFile).Decode(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token JSON: %v", err)
	}
	return token, nil
}

func getClient(ctx context.Context, conf *oauth2.Config) (*http.Client, error) {
	token, err := getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}
	return conf.Client(ctx, token), nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then paste redirect url: \n%v\n", authURL)

	var redirectUrl string
	if _, err := fmt.Scan(&redirectUrl); err != nil {
		return nil, fmt.Errorf("unable to read authorization code %v", err)
	}

	authCode, err := parseRedirectURL(redirectUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse redirect URL %v", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web %v", err)
	}
	return tok, nil
}

func parseRedirectURL(u string) (string, error) {
	// Parse the redirect URL and extract the authorization code
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect URL: %v", err)
	}

	// Get the "code" query parameter
	authCode := parsedUrl.Query().Get("code")
	if authCode == "" {
		return "", fmt.Errorf("authorization code not found in redirect URL")
	}

	return authCode, nil
}
