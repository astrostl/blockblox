package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

const (
	baseURL            = "https://apis.roblox.com/user-settings-api/v1"
	settingsURL        = baseURL + "/user-settings/settings-and-options"
	updateURL          = baseURL + "/user-settings"
	usersURL           = "https://users.roblox.com/v1/users/authenticated"
	userByIDURL        = "https://users.roblox.com/v1/users/%d"
	parentalControlURL = "https://apis.roblox.com/parental-controls-api/v1/parental-controls/get-weekly-screentime"
	tempScreenTimeURL  = "https://apis.roblox.com/parental-controls-api/v1/parental-controls/add-temporary-screentime"
	restrictionURL     = "https://usermoderation.roblox.com/v2/not-approved"
	banDetailsURL      = "https://usermoderation.roblox.com/v1/not-approved"
	csrfTokenHeader    = "X-Csrf-Token"
)

var version = "dev"

type Client struct {
	httpClient     *http.Client
	security       string
	browserTracker string
	csrfToken      string
}

type SettingsResponse struct {
	DailyScreenTimeLimit struct {
		CurrentValue int `json:"currentValue"`
	} `json:"dailyScreenTimeLimit"`
}

type UpdateRequest struct {
	DailyScreenTimeLimit int `json:"dailyScreenTimeLimit"`
}

type UserResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type DailyScreentime struct {
	DaysAgo       int `json:"daysAgo"`
	MinutesPlayed int `json:"minutesPlayed"`
}

type WeeklyScreentimeResponse struct {
	DailyScreentimes []DailyScreentime `json:"dailyScreentimes"`
}

type Restriction struct {
	Source           int    `json:"source"`
	ModerationStatus int    `json:"moderationStatus"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	DurationSeconds  int    `json:"durationSeconds"`
}

type RestrictionResponse struct {
	Restriction *Restriction `json:"restriction"`
}

type BanDetails struct {
	PunishedUserId            int64  `json:"punishedUserId"`
	MessageToUser             string `json:"messageToUser"`
	PunishmentTypeDescription string `json:"punishmentTypeDescription"`
	EndDate                   string `json:"endDate"`
}

func NewClient() (*Client, error) {
	security := os.Getenv("ROBLOX_SECURITY")
	browserTracker := os.Getenv("ROBLOX_BROWSER_TRACKER")

	if security == "" {
		return nil, fmt.Errorf("ROBLOX_SECURITY environment variable not set")
	}
	if browserTracker == "" {
		return nil, fmt.Errorf("ROBLOX_BROWSER_TRACKER environment variable not set")
	}

	return &Client{
		httpClient:     &http.Client{},
		security:       security,
		browserTracker: browserTracker,
	}, nil
}

func (c *Client) addCookies(req *http.Request) {
	req.Header.Set("Cookie", fmt.Sprintf(".ROBLOSECURITY=%s; RBXEventTrackerV2=%s", c.security, c.browserTracker))
}

func (c *Client) fetchCSRFToken() error {
	req, err := http.NewRequest("POST", updateURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}

	c.addCookies(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	token := resp.Header.Get(csrfTokenHeader)
	if token == "" {
		return fmt.Errorf("failed to get CSRF token from response")
	}

	c.csrfToken = token
	return nil
}

func (c *Client) GetScreenTime() (int, error) {
	req, err := http.NewRequest("GET", settingsURL, nil)
	if err != nil {
		return 0, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var settings SettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return 0, err
	}

	return settings.DailyScreenTimeLimit.CurrentValue, nil
}

func (c *Client) GetUser() (*UserResponse, error) {
	req, err := http.NewRequest("GET", usersURL, nil)
	if err != nil {
		return nil, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check if user is moderated - try to get user info via ban details or HTML scrape
		if strings.Contains(string(body), "moderated") {
			// Try ban details first (has user ID for bans)
			if ban, err := c.GetBanDetails(); err == nil && ban.PunishedUserId > 0 {
				return c.GetUserByID(ban.PunishedUserId)
			}
			// Fall back to HTML scrape (works for screen time blocks)
			if user, err := c.GetUserFromHTML(); err == nil {
				return user, nil
			}
		}
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) GetUserFromHTML() (*UserResponse, error) {
	req, err := http.NewRequest("GET", "https://www.roblox.com/not-approved", nil)
	if err != nil {
		return nil, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)

	// Parse data-userid="..." and data-name="..."
	userIDRe := regexp.MustCompile(`data-userid="(\d+)"`)
	nameRe := regexp.MustCompile(`data-name="([^"]+)"`)

	userIDMatch := userIDRe.FindStringSubmatch(html)
	nameMatch := nameRe.FindStringSubmatch(html)

	if userIDMatch == nil || nameMatch == nil {
		return nil, fmt.Errorf("could not parse user info from HTML")
	}

	userID, _ := strconv.ParseInt(userIDMatch[1], 10, 64)
	name := nameMatch[1]

	// Get display name from public API
	if fullUser, err := c.GetUserByID(userID); err == nil {
		return fullUser, nil
	}

	// Fall back to just what we scraped
	return &UserResponse{
		ID:          userID,
		Name:        name,
		DisplayName: name,
	}, nil
}

func (c *Client) GetUserByID(userID int64) (*UserResponse, error) {
	url := fmt.Sprintf(userByIDURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) GetTodayConsumption(userID int64) (int, error) {
	url := fmt.Sprintf("%s?userId=%d", parentalControlURL, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var weekly WeeklyScreentimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&weekly); err != nil {
		return 0, err
	}

	for _, day := range weekly.DailyScreentimes {
		if day.DaysAgo == 0 {
			return day.MinutesPlayed, nil
		}
	}

	return 0, nil
}

func (c *Client) SetScreenTime(minutes int) error {
	if c.csrfToken == "" {
		if err := c.fetchCSRFToken(); err != nil {
			return fmt.Errorf("failed to fetch CSRF token: %w", err)
		}
	}

	payload := UpdateRequest{DailyScreenTimeLimit: minutes}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", updateURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	c.addCookies(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrfTokenHeader, c.csrfToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle CSRF token expiration
	if resp.StatusCode == http.StatusForbidden {
		newToken := resp.Header.Get(csrfTokenHeader)
		if newToken != "" {
			c.csrfToken = newToken
			return c.SetScreenTime(minutes)
		}
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) AddTemporaryScreenTime(minutes int) error {
	if c.csrfToken == "" {
		if err := c.fetchCSRFToken(); err != nil {
			return fmt.Errorf("failed to fetch CSRF token: %w", err)
		}
	}

	payload := map[string]int{"minutes": minutes}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", tempScreenTimeURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	c.addCookies(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrfTokenHeader, c.csrfToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle CSRF token expiration
	if resp.StatusCode == http.StatusForbidden {
		newToken := resp.Header.Get(csrfTokenHeader)
		if newToken != "" {
			c.csrfToken = newToken
			return c.AddTemporaryScreenTime(minutes)
		}
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) GetRestriction() (*Restriction, error) {
	req, err := http.NewRequest("GET", restrictionURL, nil)
	if err != nil {
		return nil, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result RestrictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Restriction, nil
}

func (c *Client) GetBanDetails() (*BanDetails, error) {
	req, err := http.NewRequest("GET", banDetailsURL, nil)
	if err != nil {
		return nil, err
	}

	c.addCookies(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var result BanDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func formatTimeUntil(isoDate string) string {
	endTime, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate
	}
	duration := time.Until(endTime)
	if duration < 0 {
		return "expired"
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	mins := int(duration.Minutes()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d day(s)", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hour(s)", hours))
	}
	if mins > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d minute(s)", mins))
	}
	return strings.Join(parts, " ")
}

func formatResetTime(isoDate string) string {
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate
	}
	local := t.Local()
	now := time.Now()

	// Check if it's today or tomorrow
	if local.YearDay() == now.YearDay() && local.Year() == now.Year() {
		return local.Format("today at 3:04 PM")
	}
	tomorrow := now.AddDate(0, 0, 1)
	if local.YearDay() == tomorrow.YearDay() && local.Year() == tomorrow.Year() {
		return local.Format("tomorrow at 3:04 PM")
	}
	return local.Format("Mon Jan 2 at 3:04 PM")
}

func (c *Client) CheckRestrictionError() string {
	restriction, err := c.GetRestriction()
	if err != nil || restriction == nil {
		return ""
	}
	switch restriction.Source {
	case 1:
		if ban, err := c.GetBanDetails(); err == nil {
			return fmt.Sprintf("%s\nReason: %s\nEnds in: %s", ban.PunishmentTypeDescription, ban.MessageToUser, formatTimeUntil(ban.EndDate))
		}
		return "Account may be banned. Open roblox.com in a browser to confirm."
	case 2:
		return "Screen time limit reached. Use 'blockblox temp <minutes>' to add temporary time."
	default:
		return ""
	}
}

func formatDuration(minutes int) string {
	if minutes == 0 || minutes >= 1440 {
		return "No limit"
	}
	hours := minutes / 60
	mins := minutes % 60
	if hours > 0 && mins > 0 {
		return fmt.Sprintf("%d hour(s) %d minute(s)", hours, mins)
	} else if hours > 0 {
		return fmt.Sprintf("%d hour(s)", hours)
	}
	return fmt.Sprintf("%d minute(s)", mins)
}

func parseDuration(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Try plain number first (minutes)
	if mins, err := strconv.Atoi(s); err == nil {
		return mins, nil
	}

	// Try duration patterns: 4h, 90m, 4h15m
	re := regexp.MustCompile(`^(?:(\d+)h)?(?:(\d+)m)?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil || (matches[1] == "" && matches[2] == "") {
		return 0, fmt.Errorf("invalid duration format: %s (use: 90, 90m, 4h, 4h15m)", s)
	}

	var total int
	if matches[1] != "" {
		hours, _ := strconv.Atoi(matches[1])
		total += hours * 60
	}
	if matches[2] != "" {
		mins, _ := strconv.Atoi(matches[2])
		total += mins
	}

	return total, nil
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

func loadCredentials() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	loadEnvFile(filepath.Join(home, ".blockblox.env"))
}

// Chrome cookie extraction for macOS

func getChromeCookiesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies"), nil
}

func getChromeEncryptionKey() ([]byte, error) {
	// Get the encryption key from macOS Keychain
	cmd := exec.Command("security", "find-generic-password", "-s", "Chrome Safe Storage", "-w")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Chrome encryption key from Keychain: %w", err)
	}

	password := strings.TrimSpace(string(output))

	// Derive the key using PBKDF2
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1003, 16, sha1.New)

	return key, nil
}

func decryptCookieValue(encryptedValue []byte, key []byte) (string, error) {
	if len(encryptedValue) < 3 {
		return "", fmt.Errorf("encrypted value too short")
	}

	// Chrome on macOS prefixes encrypted cookies with "v10"
	if string(encryptedValue[:3]) != "v10" {
		// Not encrypted, return as-is
		return string(encryptedValue), nil
	}

	encryptedValue = encryptedValue[3:]

	if len(encryptedValue) < aes.BlockSize {
		return "", fmt.Errorf("encrypted value too short for AES")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Chrome uses a fixed IV of 16 spaces
	iv := []byte("                ") // 16 spaces

	mode := cipher.NewCBCDecrypter(block, iv)

	decrypted := make([]byte, len(encryptedValue))
	mode.CryptBlocks(decrypted, encryptedValue)

	// Remove PKCS7 padding
	if len(decrypted) > 0 {
		padding := int(decrypted[len(decrypted)-1])
		if padding > 0 && padding <= aes.BlockSize && padding <= len(decrypted) {
			decrypted = decrypted[:len(decrypted)-padding]
		}
	}

	// Find the start of the actual cookie value (skip any decryption artifacts)
	result := string(decrypted)

	// ROBLOSECURITY starts with "_|WARNING:"
	if idx := strings.Index(result, "_|WARNING:"); idx > 0 {
		result = result[idx:]
	}
	// RBXEventTrackerV2 contains "browserid="
	if idx := strings.Index(result, "CreateDate="); idx > 0 {
		result = result[idx:]
	}

	return result, nil
}

func extractChromeCookies() (security string, browserTracker string, err error) {
	cookiesPath, err := getChromeCookiesPath()
	if err != nil {
		return "", "", err
	}

	// Check if cookies file exists
	if _, err := os.Stat(cookiesPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("Chrome cookies file not found at %s", cookiesPath)
	}

	// Copy cookies file to temp location (Chrome locks the original)
	tmpFile, err := os.CreateTemp("", "chrome-cookies-*.db")
	if err != nil {
		return "", "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Copy the file
	input, err := os.ReadFile(cookiesPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read cookies file: %w", err)
	}
	if err := os.WriteFile(tmpPath, input, 0600); err != nil {
		return "", "", fmt.Errorf("failed to copy cookies file: %w", err)
	}

	// Get encryption key
	key, err := getChromeEncryptionKey()
	if err != nil {
		return "", "", err
	}

	// Open the database
	db, err := sql.Open("sqlite3", tmpPath)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	// Query for Roblox cookies
	rows, err := db.Query(`
		SELECT name, encrypted_value
		FROM cookies
		WHERE host_key LIKE '%roblox.com'
		AND name IN ('.ROBLOSECURITY', 'RBXEventTrackerV2')
	`)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var encryptedValue []byte
		if err := rows.Scan(&name, &encryptedValue); err != nil {
			continue
		}

		value, err := decryptCookieValue(encryptedValue, key)
		if err != nil {
			continue
		}

		switch name {
		case ".ROBLOSECURITY":
			security = value
		case "RBXEventTrackerV2":
			browserTracker = value
		}
	}

	if security == "" {
		return "", "", fmt.Errorf(".ROBLOSECURITY cookie not found - make sure you're logged into Roblox in Chrome")
	}
	if browserTracker == "" {
		return "", "", fmt.Errorf("RBXEventTrackerV2 cookie not found")
	}

	return security, browserTracker, nil
}

func saveCredentials(security, browserTracker string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	envPath := filepath.Join(home, ".blockblox.env")
	content := fmt.Sprintf("ROBLOX_SECURITY=%s\nROBLOX_BROWSER_TRACKER=%s\n", security, browserTracker)

	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Printf("Credentials saved to %s\n", envPath)
	return nil
}

func runInit() error {
	fmt.Println("Extracting Roblox credentials from Chrome...")

	security, browserTracker, err := extractChromeCookies()
	if err != nil {
		return err
	}

	fmt.Println("Found credentials!")
	if err := saveCredentials(security, browserTracker); err != nil {
		return err
	}

	// Set env vars so subsequent get works
	os.Setenv("ROBLOX_SECURITY", security)
	os.Setenv("ROBLOX_BROWSER_TRACKER", browserTracker)
	return nil
}

func printUsage() {
	fmt.Println("blockblox - Roblox Screen Time Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  blockblox init          Extract credentials from Chrome")
	fmt.Println("  blockblox get           Get current screen time limit")
	fmt.Println("  blockblox set <time>    Set screen time limit (0 = no limit)")
	fmt.Println("  blockblox temp <time>   Add temporary screen time (works when screen time exceeded)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blockblox set 90        Set limit to 90 minutes")
	fmt.Println("  blockblox set 90m       Set limit to 90 minutes")
	fmt.Println("  blockblox set 4h        Set limit to 4 hours")
	fmt.Println("  blockblox set 4h15m     Set limit to 4 hours 15 minutes")
	fmt.Println("  blockblox set 0         Remove limit")
	fmt.Println("  blockblox temp 5        Add 5 minutes temporarily")
	fmt.Println("  blockblox temp 15m      Add 15 minutes temporarily")
	fmt.Println()
	fmt.Println("Credentials stored in ~/.blockblox.env")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Handle flags that don't require credentials
	switch os.Args[1] {
	case "-v", "-version", "--version", "version":
		fmt.Println(version)
		return
	case "-h", "-help", "--help", "help":
		printUsage()
		return
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
		os.Args[1] = "get" // Fall through to get
	}

	// Load credentials for other commands
	loadCredentials()

	client, err := NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'blockblox init' to extract credentials from Chrome\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "get":
		user, err := client.GetUser()
		if err != nil {
			if msg := client.CheckRestrictionError(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error getting user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("User: %s (@%s)\n", user.DisplayName, user.Name)

		// Check for restrictions before trying other APIs
		if restriction, _ := client.GetRestriction(); restriction != nil {
			switch restriction.Source {
			case 1: // Ban
				if ban, err := client.GetBanDetails(); err == nil {
					fmt.Printf("\n%s\nReason: %s\nEnds in: %s\n", ban.PunishmentTypeDescription, ban.MessageToUser, formatTimeUntil(ban.EndDate))
				}
				os.Exit(1)
			case 2: // Screen time
				fmt.Printf("\nScreen time limit reached.\nResets: %s\n", formatResetTime(restriction.EndTime))
				fmt.Println("\nUse 'blockblox temp <minutes>' to add temporary time.")
				os.Exit(1)
			}
		}

		minutes, err := client.GetScreenTime()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting screen time: %v\n", err)
			os.Exit(1)
		}
		if minutes >= 60 {
			fmt.Printf("Limit: %s (%d minutes)\n", formatDuration(minutes), minutes)
		} else {
			fmt.Printf("Limit: %s\n", formatDuration(minutes))
		}

		consumed, err := client.GetTodayConsumption(user.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting consumption: %v\n", err)
			os.Exit(1)
		}
		if consumed >= 60 {
			fmt.Printf("Consumed: %s (%d minutes)\n", formatDuration(consumed), consumed)
		} else {
			fmt.Printf("Consumed: %s\n", formatDuration(consumed))
		}
		if minutes > 0 && minutes < 1440 {
			if consumed > minutes {
				fmt.Printf("Status: Temporary time active (over limit by %s)\n", formatDuration(consumed-minutes))
			} else {
				fmt.Printf("Remaining: %s\n", formatDuration(minutes-consumed))
			}
		} else {
			fmt.Println("Remaining: Unlimited")
		}

	case "set":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: missing minutes argument")
			fmt.Fprintln(os.Stderr, "Usage: blockblox set <minutes>")
			os.Exit(1)
		}

		minutes, err := parseDuration(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if minutes < 0 {
			fmt.Fprintln(os.Stderr, "Error: duration cannot be negative")
			os.Exit(1)
		}
		if minutes == 0 {
			minutes = 1440 // 24 hours = no limit
		}

		user, err := client.GetUser()
		if err != nil {
			if msg := client.CheckRestrictionError(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error getting user: %v\n", err)
			os.Exit(1)
		}

		// Check for restrictions before trying to set
		if restriction, _ := client.GetRestriction(); restriction != nil {
			fmt.Printf("User: %s (@%s)\n", user.DisplayName, user.Name)
			switch restriction.Source {
			case 1: // Ban
				if ban, err := client.GetBanDetails(); err == nil {
					fmt.Fprintf(os.Stderr, "\n%s\nReason: %s\nEnds in: %s\n", ban.PunishmentTypeDescription, ban.MessageToUser, formatTimeUntil(ban.EndDate))
				}
				os.Exit(1)
			case 2: // Screen time
				fmt.Fprintf(os.Stderr, "\nScreen time limit reached.\nResets: %s\n", formatResetTime(restriction.EndTime))
				fmt.Fprintln(os.Stderr, "\nUse 'blockblox temp <minutes>' to add temporary time.")
				os.Exit(1)
			}
		}

		if err := client.SetScreenTime(minutes); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting screen time: %v\n", err)
			os.Exit(1)
		}
		displayMinutes := minutes
		if minutes >= 1440 {
			displayMinutes = 0
		}

		consumed, err := client.GetTodayConsumption(user.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting consumption: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User: %s (@%s)\n", user.DisplayName, user.Name)
		if minutes >= 60 {
			fmt.Printf("Limit set to: %s (%d minutes)\n", formatDuration(minutes), displayMinutes)
		} else {
			fmt.Printf("Limit set to: %s\n", formatDuration(minutes))
		}
		if consumed >= 60 {
			fmt.Printf("Consumed: %s (%d minutes)\n", formatDuration(consumed), consumed)
		} else {
			fmt.Printf("Consumed: %s\n", formatDuration(consumed))
		}
		if displayMinutes > 0 {
			if consumed > displayMinutes {
				fmt.Printf("Status: Temporary time active (over limit by %s)\n", formatDuration(consumed-displayMinutes))
			} else {
				fmt.Printf("Remaining: %s\n", formatDuration(displayMinutes-consumed))
			}
		} else {
			fmt.Println("Remaining: Unlimited")
		}

	case "temp":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: missing time argument")
			fmt.Fprintln(os.Stderr, "Usage: blockblox temp <time>")
			os.Exit(1)
		}

		minutes, err := parseDuration(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if minutes <= 0 {
			fmt.Fprintln(os.Stderr, "Error: duration must be positive")
			os.Exit(1)
		}

		// Check for ban (temp doesn't work for bans)
		if restriction, _ := client.GetRestriction(); restriction != nil && restriction.Source == 1 {
			if ban, err := client.GetBanDetails(); err == nil {
				if user, err := client.GetUserByID(ban.PunishedUserId); err == nil {
					fmt.Fprintf(os.Stderr, "User: %s (@%s)\n\n", user.DisplayName, user.Name)
				}
				fmt.Fprintf(os.Stderr, "%s\nReason: %s\nEnds in: %s\n", ban.PunishmentTypeDescription, ban.MessageToUser, formatTimeUntil(ban.EndDate))
			} else {
				fmt.Fprintln(os.Stderr, "Account is banned. Open roblox.com in a browser for details.")
			}
			os.Exit(1)
		}

		// Show user info (works via HTML scrape even when blocked)
		if user, err := client.GetUser(); err == nil {
			fmt.Printf("User: %s (@%s)\n", user.DisplayName, user.Name)
		}

		if err := client.AddTemporaryScreenTime(minutes); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding temporary screen time: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added %s of temporary screen time\n", formatDuration(minutes))
		fmt.Println("Note: There is no way to check remaining temp time. It expires silently.")

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
