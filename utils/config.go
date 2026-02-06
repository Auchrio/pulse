package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the application configuration
type Config struct {
	Relays          []string
	HistoryLimit    int
	UserSecret      string
	DefaultUsername string
	ListenTimeout   int
}

// GetConfigPath returns the path to the pulse.conf file
func GetConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "pulse.conf"), nil
}

// LoadConfig loads configuration from pulse.conf if it exists
func LoadConfig() *Config {
	confPath, err := GetConfigPath()
	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(confPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	config := &Config{
		Relays:        []string{},
		HistoryLimit:  HistoryLimit,
		UserSecret:    UserSecret,
		ListenTimeout: ListenTimeout,
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "relays":
			// Parse comma-separated relay list
			relayList := strings.Split(value, ",")
			for _, relay := range relayList {
				relay = strings.TrimSpace(relay)
				if relay != "" {
					config.Relays = append(config.Relays, relay)
				}
			}
		case "history-limit":
			if limit, err := strconv.Atoi(value); err == nil {
				config.HistoryLimit = limit
			}
		case "user-secret":
			config.UserSecret = value
		case "default-username":
			config.DefaultUsername = value
		case "listen-timeout":
			if timeout, err := strconv.Atoi(value); err == nil {
				config.ListenTimeout = timeout
			}
		}
	}

	return config
}

// ApplyConfig applies configuration to global variables
func ApplyConfig(config *Config) {
	if config == nil {
		return
	}

	if len(config.Relays) > 0 {
		Relays = config.Relays
	}
	if config.HistoryLimit > 0 {
		HistoryLimit = config.HistoryLimit
	}
	if config.UserSecret != "" {
		UserSecret = config.UserSecret
	}
	if config.ListenTimeout >= 0 {
		ListenTimeout = config.ListenTimeout
	}
}

// GenerateConfig creates a pulse.conf file with default settings
func GenerateConfig() error {
	confPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Check if file already exists
	if _, err := os.Stat(confPath); err == nil {
		return fmt.Errorf("pulse.conf already exists at %s", confPath)
	}

	content := `# Pulse Configuration File
# Key-value pairs to configure the application

# List of Nostr relays (comma-separated)
relays = wss://relay.damus.io, wss://nos.lol, wss://relay.snort.social

# Maximum number of messages to retrieve from history
history-limit = 5

# Secret key for encryption (used with message ID to derive encryption key)
user-secret = super-secret-key

# Default username to use in chat mode (optional)
# If set, skips the username prompt unless overridden from command line
# default-username = YourName

# Listen timeout in seconds (for -l flag, 0 = no timeout)
listen-timeout = 30
`

	file, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	fmt.Printf("Generated pulse.conf at %s\n", confPath)
	return nil
}
