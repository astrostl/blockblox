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

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/pbkdf2"
)

const (
	baseURL            = "https://apis.roblox.com/user-settings-api/v1"
	settingsURL        = baseURL + "/user-settings/settings-and-options"
	updateURL          = baseURL + "/user-settings"
	usersURL           = "https://users.roblox.com/v1/users/authenticated"
	parentalControlURL = "https://apis.roblox.com/parental-controls-api/v1/parental-controls/get-weekly-screentime"
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
	ID int64 `json:"id"`
}

type DailyScreentime struct {
	DaysAgo       int `json:"daysAgo"`
	MinutesPlayed int `json:"minutesPlayed"`
}

type WeeklyScreentimeResponse struct {
	DailyScreentimes []DailyScreentime `json:"dailyScreentimes"`
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

func (c *Client) GetUserID() (int64, error) {
	req, err := http.NewRequest("GET", usersURL, nil)
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

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return 0, err
	}

	return user.ID, nil
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
	return saveCredentials(security, browserTracker)
}

func printUsage() {
	fmt.Println("blockblox - Roblox Screen Time Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  blockblox init          Extract credentials from Chrome")
	fmt.Println("  blockblox get           Get current screen time limit")
	fmt.Println("  blockblox set <time>    Set screen time limit (0 = no limit)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blockblox set 90        Set limit to 90 minutes")
	fmt.Println("  blockblox set 90m       Set limit to 90 minutes")
	fmt.Println("  blockblox set 4h        Set limit to 4 hours")
	fmt.Println("  blockblox set 4h15m     Set limit to 4 hours 15 minutes")
	fmt.Println("  blockblox set 0         Remove limit")
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
		return
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
		minutes, err := client.GetScreenTime()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting screen time: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Limit: %s (%d minutes)\n", formatDuration(minutes), minutes)

		userID, err := client.GetUserID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting user ID: %v\n", err)
			os.Exit(1)
		}

		consumed, err := client.GetTodayConsumption(userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting consumption: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Consumed: %s (%d minutes)\n", formatDuration(consumed), consumed)

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

		if err := client.SetScreenTime(minutes); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting screen time: %v\n", err)
			os.Exit(1)
		}
		displayMinutes := minutes
		if minutes >= 1440 {
			displayMinutes = 0
		}
		fmt.Printf("Screen time limit set to: %s (%d minutes)\n", formatDuration(minutes), displayMinutes)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
