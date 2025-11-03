package pricing

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// SpecialPricing checks if the AKASH_OWNER is in a predefined list and applies special pricing if so.
func SpecialPricing(owner string) bool {
	specialAccounts := map[string]bool{
		"akash1fxa9ss3dg6nqyz8aluyaa6svypgprk5tw9fa4q": true,
		"akash1fhe3uk7d95vvr69pna7cxmwa8777as46uyxcz8": true,
	}
	return specialAccounts[owner]
}

// CheckWhitelist checks if the AKASH_OWNER is in the whitelist defined by the WHITELIST_URL.
func CheckWhitelist(owner string) error {
	whitelistURL := os.Getenv("WHITELIST_URL")
	whitelistURL = strings.Trim(whitelistURL, "\"") // Trim any double quotes from the URL

	if whitelistURL == "" {
		return nil // No whitelist URL set, skip checking
	}

	whitelistFile := "/tmp/price-script.whitelist"
	if shouldFetchWhitelist(whitelistFile) {
		if err := fetchWhitelist(whitelistURL, whitelistFile); err != nil {
			return fmt.Errorf("error fetching whitelist: %w", err)
		}
	}

	if err := verifyInWhitelist(whitelistFile, os.Getenv("AKASH_OWNER")); err != nil {
		return err
	}

	return nil
}

// shouldFetchWhitelist checks if the whitelist file should be fetched again.
func shouldFetchWhitelist(whitelistFile string) bool {
	fileInfo, err := os.Stat(whitelistFile)
	if os.IsNotExist(err) || time.Since(fileInfo.ModTime()) > 10*time.Minute {
		return true
	}
	return false
}

// fetchWhitelist downloads the whitelist from the given URL and saves it.
func fetchWhitelist(whitelistURL, whitelistFile string) error {
	resp, err := http.Get(whitelistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request error: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(whitelistFile, body, 0644)
}

// verifyInWhitelist checks if the given owner is in the whitelist file.
func verifyInWhitelist(whitelistFile, owner string) error {
	file, err := os.Open(whitelistFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == owner {
			return nil // Owner is in the whitelist
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return fmt.Errorf("%s is not whitelisted", owner)
}
