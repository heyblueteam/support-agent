package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GetAuthenticatedClient returns an authenticated Gmail service
func GetAuthenticatedClient() (*gmail.Service, error) {
	ctx := context.Background()
	
	// Get OAuth2 config
	config, err := getOAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get OAuth config: %v", err)
	}

	// Get or refresh token
	token, err := getToken(config)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %v", err)
	}

	// Create Gmail service
	client := config.Client(ctx, token)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %v", err)
	}

	return srv, nil
}

// getOAuthConfig loads OAuth2 config from credentials file
func getOAuthConfig() (*oauth2.Config, error) {
	// Check environment variable first
	credPath := os.Getenv("GMAIL_CREDENTIALS_PATH")
	if credPath == "" {
		credPath = "gmail.json"
	}

	// Also check raw credentials from env
	credJSON := os.Getenv("GMAIL_CREDENTIALS")
	var b []byte
	var err error

	if credJSON != "" {
		b = []byte(credJSON)
	} else {
		b, err = ioutil.ReadFile(credPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read credentials file: %v", err)
		}
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %v", err)
	}

	return config, nil
}

// getToken retrieves a token from cache or initiates OAuth flow
func getToken(config *oauth2.Config) (*oauth2.Token, error) {
	tokenFile := getTokenPath()
	
	// Try to load existing token
	tok, err := loadToken(tokenFile)
	if err == nil {
		return tok, nil
	}

	// Get new token via OAuth flow
	tok, err = getTokenFromWeb(config)
	if err != nil {
		return nil, err
	}

	// Save token for future use
	saveToken(tokenFile, tok)
	return tok, nil
}

// getTokenPath returns the path to the token file
func getTokenPath() string {
	tokenDir := os.Getenv("TOKEN_DIR")
	if tokenDir == "" {
		home, _ := os.UserHomeDir()
		tokenDir = filepath.Join(home, ".support-agent")
	}
	
	// Create directory if it doesn't exist
	os.MkdirAll(tokenDir, 0700)
	
	return filepath.Join(tokenDir, "token.json")
}

// loadToken retrieves a token from a local file
func loadToken(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to save token: %v", err)
	}
	defer f.Close()
	
	return json.NewEncoder(f).Encode(token)
}

// getTokenFromWeb initiates OAuth flow to get a new token
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)
	fmt.Print("Enter authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange authorization code: %v", err)
	}

	return tok, nil
}