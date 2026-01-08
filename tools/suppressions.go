package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

const emailitBaseURL = "https://api.emailit.com/v1"

// Suppression represents an email suppression record
type Suppression struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Email     string `json:"email"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
	KeepUntil *string `json:"keep_until"`
}

// SuppressionList represents the API response for listing suppressions
type SuppressionList struct {
	Data         []Suppression `json:"data"`
	TotalRecords int           `json:"total_records"`
}

// getEmailitAPIKey loads the API key from environment
func getEmailitAPIKey() (string, error) {
	godotenv.Load()

	apiKey := os.Getenv("EMAILIT_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("EMAILIT_API_KEY not set in environment")
	}
	return apiKey, nil
}

// RunSuppressions handles the suppressions command
func RunSuppressions(args []string) error {
	fs := flag.NewFlagSet("suppressions", flag.ExitOnError)

	// Define flags
	list := fs.Bool("list", false, "List all suppressions")
	check := fs.String("check", "", "Check if an email is suppressed")
	remove := fs.String("remove", "", "Remove suppression for an email")

	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate - need at least one action
	if !*list && *check == "" && *remove == "" {
		fmt.Println("Error: specify an action")
		fmt.Println("\nUsage:")
		fmt.Println("  suppressions --list                    List all suppressions")
		fmt.Println("  suppressions --check EMAIL             Check if email is suppressed")
		fmt.Println("  suppressions --remove EMAIL            Remove suppression for email")
		return fmt.Errorf("action required")
	}

	apiKey, err := getEmailitAPIKey()
	if err != nil {
		return err
	}

	if *list {
		return listSuppressions(apiKey)
	}

	if *check != "" {
		return checkSuppression(apiKey, *check)
	}

	if *remove != "" {
		return removeSuppression(apiKey, *remove)
	}

	return nil
}

// listSuppressions lists all email suppressions
func listSuppressions(apiKey string) error {
	req, err := http.NewRequest("GET", emailitBaseURL+"/suppressions", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result SuppressionList
	if err := json.Unmarshal(body, &result); err != nil {
		// Try unmarshaling as array directly
		var suppressions []Suppression
		if err := json.Unmarshal(body, &suppressions); err != nil {
			// Just print raw response
			fmt.Println("Suppressions:")
			fmt.Println(string(body))
			return nil
		}
		result.Data = suppressions
	}

	if len(result.Data) == 0 {
		fmt.Println("No suppressions found.")
		return nil
	}

	fmt.Printf("Found %d suppression(s) (total: %d):\n\n", len(result.Data), result.TotalRecords)
	for _, s := range result.Data {
		fmt.Printf("%s\n", s.Email)
		fmt.Printf("  ID: %d | Reason: %s\n", s.ID, s.Reason)
	}

	return nil
}

// checkSuppression checks if a specific email is suppressed
func checkSuppression(apiKey, email string) error {
	suppression, err := findSuppressionByEmail(apiKey, email)
	if err != nil {
		return err
	}

	if suppression == nil {
		fmt.Printf("Email '%s' is NOT suppressed.\n", email)
		return nil
	}

	fmt.Printf("Email '%s' IS suppressed:\n", email)
	fmt.Printf("  ID: %d\n", suppression.ID)
	fmt.Printf("  Reason: %s\n", suppression.Reason)

	return nil
}

// findSuppressionByEmail searches the suppression list for a specific email
func findSuppressionByEmail(apiKey, email string) (*Suppression, error) {
	req, err := http.NewRequest("GET", emailitBaseURL+"/suppressions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result SuppressionList
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	for _, s := range result.Data {
		if s.Email == email {
			return &s, nil
		}
	}

	return nil, nil
}

// removeSuppression removes a suppression for an email
func removeSuppression(apiKey, email string) error {
	// First find the suppression to get its ID
	suppression, err := findSuppressionByEmail(apiKey, email)
	if err != nil {
		return err
	}

	if suppression == nil {
		fmt.Printf("Email '%s' is not in the suppression list.\n", email)
		return nil
	}

	// Delete by ID
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/suppressions/%d", emailitBaseURL, suppression.ID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully removed suppression for '%s' (ID: %d).\n", email, suppression.ID)
	return nil
}
