package tools

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

const (
	// Support user email - hardcoded for security
	supportUserEmail = "manny@blue.cc"
	ownerLevel       = "OWNER"
)

// AccessResult represents the result of an access operation
type AccessResult struct {
	Action      string          `json:"action"`
	User        UserInfo        `json:"user"`
	Company     CompanyInfo     `json:"company"`
	Projects    []ProjectInfo   `json:"projects,omitempty"`
	CompanyUser CompanyUserInfo `json:"company_user,omitempty"`
}

// UserInfo represents basic user data
type UserInfo struct {
	ID    string `json:"id"`
	UID   string `json:"uid"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CompanyInfo represents basic company data
type CompanyInfo struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// ProjectInfo represents basic project data
type ProjectInfo struct {
	ID     string `json:"id"`
	UID    string `json:"uid"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CompanyUserInfo represents the company user record
type CompanyUserInfo struct {
	ID    string `json:"id"`
	UID   string `json:"uid"`
	Level string `json:"level"`
}

// getDatabaseURL loads the database URL from environment
func getDatabaseURL() (string, error) {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return "", fmt.Errorf("DATABASE_URL not set in environment")
	}

	// Convert from prisma format to Go MySQL driver format
	// From: mysql://user:pass@host:port/dbname
	// To: user:pass@tcp(host:port)/dbname?parseTime=true
	dbURL = strings.TrimPrefix(dbURL, "mysql://")

	// Find the @ separator
	atIdx := strings.LastIndex(dbURL, "@")
	if atIdx == -1 {
		return "", fmt.Errorf("invalid DATABASE_URL format")
	}

	userPass := dbURL[:atIdx]
	hostAndDB := dbURL[atIdx+1:]

	// Find the / separator for host:port and dbname
	slashIdx := strings.Index(hostAndDB, "/")
	if slashIdx == -1 {
		return "", fmt.Errorf("invalid DATABASE_URL format: missing database name")
	}

	hostPort := hostAndDB[:slashIdx]
	dbName := hostAndDB[slashIdx+1:]

	// Remove any query params from dbName for now, we'll add our own
	if qIdx := strings.Index(dbName, "?"); qIdx != -1 {
		dbName = dbName[:qIdx]
	}

	return fmt.Sprintf("%s@tcp(%s)/%s?parseTime=true", userPass, hostPort, dbName), nil
}

// generateCUID generates a CUID-like identifier
func generateCUID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	timestamp := time.Now().UnixMilli()

	// Start with 'c' prefix like real CUIDs
	result := make([]byte, 25)
	result[0] = 'c'

	// Add timestamp component (base36-ish)
	ts := timestamp
	for i := 8; i >= 1; i-- {
		result[i] = chars[ts%36]
		ts /= 36
	}

	// Add random component
	for i := 9; i < 25; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

// RunCompanyAccess handles the company-access command
func RunCompanyAccess(args []string) error {
	fs := flag.NewFlagSet("company-access", flag.ExitOnError)

	// Define flags
	companySlug := fs.String("company", "", "Company slug (required)")
	projectSlugs := fs.String("projects", "", "Comma-separated project slugs (optional)")
	remove := fs.Bool("remove", false, "Remove access instead of granting")
	outputFormat := fs.String("output", "detailed", "Output format: detailed, json")

	// Parse args
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate
	if *companySlug == "" {
		fmt.Println("Error: --company is required")
		fmt.Println("\nUsage:")
		fmt.Println("  company-access --company SLUG                    Grant owner access to company")
		fmt.Println("  company-access --company SLUG --projects P1,P2   Also grant access to projects")
		fmt.Println("  company-access --company SLUG --remove           Remove access from company")
		fmt.Println("\nExamples:")
		fmt.Println("  company-access --company acme-corp")
		fmt.Println("  company-access --company acme-corp --projects website,mobile-app")
		fmt.Println("  company-access --company acme-corp --remove")
		fmt.Println("  company-access --company acme-corp --output json")
		return fmt.Errorf("--company is required")
	}

	// Get database connection
	dbURL, err := getDatabaseURL()
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Parse project slugs
	var projects []string
	if *projectSlugs != "" {
		projects = strings.Split(*projectSlugs, ",")
		for i := range projects {
			projects[i] = strings.TrimSpace(projects[i])
		}
	}

	if *remove {
		return removeAccess(db, *companySlug, projects, *outputFormat)
	}

	return grantAccess(db, *companySlug, projects, *outputFormat)
}

// grantAccess adds the support user as owner to company and projects
func grantAccess(db *sql.DB, companySlug string, projectSlugs []string, outputFormat string) error {
	// 1. Find the support user
	var user UserInfo
	err := db.QueryRow(`
		SELECT id, uid, email, CONCAT(firstName, ' ', lastName) as name
		FROM User
		WHERE email = ?
	`, supportUserEmail).Scan(&user.ID, &user.UID, &user.Email, &user.Name)
	if err != nil {
		return fmt.Errorf("failed to find support user (%s): %v", supportUserEmail, err)
	}

	// 2. Find the company
	var company CompanyInfo
	err = db.QueryRow(`
		SELECT id, uid, slug, name
		FROM Company
		WHERE slug = ? OR id = ?
	`, companySlug, companySlug).Scan(&company.ID, &company.UID, &company.Slug, &company.Name)
	if err != nil {
		return fmt.Errorf("failed to find company (%s): %v", companySlug, err)
	}

	// 3. Upsert CompanyUser with OWNER level
	companyUserUID := fmt.Sprintf("%s:%s", company.UID, user.UID)
	companyUserID := generateCUID()

	// Check if already exists
	var existingID string
	var existingLevel string
	err = db.QueryRow(`
		SELECT id, level FROM CompanyUser WHERE uid = ?
	`, companyUserUID).Scan(&existingID, &existingLevel)

	var companyUser CompanyUserInfo
	if err == sql.ErrNoRows {
		// Insert new
		_, err = db.Exec(`
			INSERT INTO CompanyUser (id, uid, level, company, user, createdAt, updatedAt)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		`, companyUserID, companyUserUID, ownerLevel, company.ID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to create CompanyUser: %v", err)
		}
		companyUser = CompanyUserInfo{ID: companyUserID, UID: companyUserUID, Level: ownerLevel}
		fmt.Printf("✓ Created CompanyUser with OWNER access\n")
	} else if err != nil {
		return fmt.Errorf("failed to check existing CompanyUser: %v", err)
	} else {
		// Update existing
		if existingLevel != ownerLevel {
			_, err = db.Exec(`UPDATE CompanyUser SET level = ?, updatedAt = NOW() WHERE id = ?`, ownerLevel, existingID)
			if err != nil {
				return fmt.Errorf("failed to update CompanyUser level: %v", err)
			}
			fmt.Printf("✓ Updated CompanyUser level from %s to OWNER\n", existingLevel)
		} else {
			fmt.Printf("✓ CompanyUser already has OWNER access\n")
		}
		companyUser = CompanyUserInfo{ID: existingID, UID: companyUserUID, Level: ownerLevel}
	}

	// 4. Handle projects
	var projectResults []ProjectInfo
	for _, projectSlug := range projectSlugs {
		project, err := grantProjectAccess(db, company.ID, projectSlug, user)
		if err != nil {
			fmt.Printf("✗ Failed to grant access to project %s: %v\n", projectSlug, err)
			continue
		}
		projectResults = append(projectResults, project)
	}

	// Output result
	result := AccessResult{
		Action:      "grant",
		User:        user,
		Company:     company,
		Projects:    projectResults,
		CompanyUser: companyUser,
	}

	if outputFormat == "json" {
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("ACCESS GRANTED\n")
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("User:    %s (%s)\n", user.Name, user.Email)
		fmt.Printf("Company: %s (slug: %s)\n", company.Name, company.Slug)
		fmt.Printf("Level:   %s\n", ownerLevel)
		if len(projectResults) > 0 {
			fmt.Println("\nProjects:")
			for _, p := range projectResults {
				fmt.Printf("  • %s (slug: %s) - %s\n", p.Name, p.Slug, p.Status)
			}
		}
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Println("\n⚠️  Remember to remove access when done:")
		fmt.Printf("    company-access --company %s --remove\n", company.Slug)
	}

	return nil
}

// grantProjectAccess adds the support user as owner to a project
func grantProjectAccess(db *sql.DB, companyID, projectSlug string, user UserInfo) (ProjectInfo, error) {
	// Find the project
	var project ProjectInfo
	err := db.QueryRow(`
		SELECT id, uid, slug, name
		FROM Project
		WHERE (slug = ? OR id = ?) AND company = ?
	`, projectSlug, projectSlug, companyID).Scan(&project.ID, &project.UID, &project.Slug, &project.Name)
	if err != nil {
		return project, fmt.Errorf("project not found: %v", err)
	}

	// Upsert ProjectUser with OWNER level
	projectUserUID := fmt.Sprintf("%s:%s", project.UID, user.UID)
	projectUserID := generateCUID()

	// Check if already exists
	var existingID string
	var existingLevel string
	err = db.QueryRow(`
		SELECT id, level FROM ProjectUser WHERE uid = ?
	`, projectUserUID).Scan(&existingID, &existingLevel)

	if err == sql.ErrNoRows {
		// Insert new
		_, err = db.Exec(`
			INSERT INTO ProjectUser (id, uid, level, project, user, createdAt, updatedAt)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())
		`, projectUserID, projectUserUID, ownerLevel, project.ID, user.ID)
		if err != nil {
			return project, fmt.Errorf("failed to create ProjectUser: %v", err)
		}
		project.Status = "created with OWNER access"
		fmt.Printf("✓ Created ProjectUser for %s with OWNER access\n", project.Name)
	} else if err != nil {
		return project, fmt.Errorf("failed to check existing ProjectUser: %v", err)
	} else {
		// Update existing
		if existingLevel != ownerLevel {
			_, err = db.Exec(`UPDATE ProjectUser SET level = ?, updatedAt = NOW() WHERE id = ?`, ownerLevel, existingID)
			if err != nil {
				return project, fmt.Errorf("failed to update ProjectUser level: %v", err)
			}
			project.Status = fmt.Sprintf("updated from %s to OWNER", existingLevel)
			fmt.Printf("✓ Updated ProjectUser for %s from %s to OWNER\n", project.Name, existingLevel)
		} else {
			project.Status = "already OWNER"
			fmt.Printf("✓ Already OWNER of project %s\n", project.Name)
		}
	}

	return project, nil
}

// removeAccess removes the support user from company and projects
func removeAccess(db *sql.DB, companySlug string, projectSlugs []string, outputFormat string) error {
	// 1. Find the support user
	var user UserInfo
	err := db.QueryRow(`
		SELECT id, uid, email, CONCAT(firstName, ' ', lastName) as name
		FROM User
		WHERE email = ?
	`, supportUserEmail).Scan(&user.ID, &user.UID, &user.Email, &user.Name)
	if err != nil {
		return fmt.Errorf("failed to find support user (%s): %v", supportUserEmail, err)
	}

	// 2. Find the company
	var company CompanyInfo
	err = db.QueryRow(`
		SELECT id, uid, slug, name
		FROM Company
		WHERE slug = ? OR id = ?
	`, companySlug, companySlug).Scan(&company.ID, &company.UID, &company.Slug, &company.Name)
	if err != nil {
		return fmt.Errorf("failed to find company (%s): %v", companySlug, err)
	}

	// 3. Delete all ProjectUser records for this user in this company
	result, err := db.Exec(`
		DELETE pu FROM ProjectUser pu
		INNER JOIN Project p ON pu.project = p.id
		WHERE pu.user = ? AND p.company = ?
	`, user.ID, company.ID)
	if err != nil {
		return fmt.Errorf("failed to delete ProjectUser records: %v", err)
	}
	projectsRemoved, _ := result.RowsAffected()
	if projectsRemoved > 0 {
		fmt.Printf("✓ Removed access from %d project(s)\n", projectsRemoved)
	}

	// 4. Delete CompanyUser record
	companyUserUID := fmt.Sprintf("%s:%s", company.UID, user.UID)
	result, err = db.Exec(`DELETE FROM CompanyUser WHERE uid = ?`, companyUserUID)
	if err != nil {
		return fmt.Errorf("failed to delete CompanyUser: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("✓ Removed CompanyUser record\n")
	} else {
		fmt.Printf("✓ No CompanyUser record found (already removed)\n")
	}

	// Output result
	if outputFormat == "json" {
		result := AccessResult{
			Action:  "remove",
			User:    user,
			Company: company,
		}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("ACCESS REMOVED\n")
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("User:     %s (%s)\n", user.Name, user.Email)
		fmt.Printf("Company:  %s (slug: %s)\n", company.Name, company.Slug)
		fmt.Printf("Projects: %d removed\n", projectsRemoved)
		fmt.Println("═══════════════════════════════════════════════════")
	}

	return nil
}
