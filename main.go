package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"pulse/utils"

	"github.com/spf13/cobra"
)

var chatMode bool
var listenMode bool
var verbose bool
var generateConfig bool

var rootCmd = &cobra.Command{
	Use:   "pulse <id> [message]",
	Short: "Encrypted messaging via Nostr",
	Long:  "Pulse: Send and receive encrypted messages using Nostr relays",
	Args:  cobra.MinimumNArgs(0),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load config on startup
		config := utils.LoadConfig()
		utils.ApplyConfig(config)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --generate-config
		if generateConfig {
			return utils.GenerateConfig()
		}

		// Require at least ID if not generating config
		if len(args) == 0 {
			return fmt.Errorf("ID is required (or use --generate-config)")
		}

		id := args[0]

		// Check if message was provided before -c flag (error case)
		if chatMode && len(args) > 1 {
			// Find position of -c flag in os.Args
			chatFlagIndex := -1
			for i, arg := range os.Args {
				if arg == "-c" || arg == "--chat" {
					chatFlagIndex = i
					break
				}
			}

			// The second positional argument is at os.Args position 2 (after program name and id)
			// If -c flag position is greater than 2, the second arg came before -c, so it's a message (error)
			if chatFlagIndex > 2 {
				return fmt.Errorf("message cannot be provided with chat mode")
			}
		}

		// Listen mode
		if listenMode {
			return utils.ListenForMessage(id, verbose)
		}

		// Chat mode
		if chatMode {
			username := ""
			if len(args) > 1 {
				username = args[1]
			} else {
				// Check if there's a default username configured
				confPath, err := utils.GetConfigPath()
				var hasDefaultUsername bool
				if err == nil {
					if file, err := os.Open(confPath); err == nil {
						defer file.Close()
						scanner := bufio.NewScanner(file)
						for scanner.Scan() {
							line := strings.TrimSpace(scanner.Text())
							if strings.HasPrefix(line, "default-username") {
								parts := strings.SplitN(line, "=", 2)
								if len(parts) == 2 {
									defaultUsername := strings.TrimSpace(parts[1])
									if defaultUsername != "" {
										username = defaultUsername
										hasDefaultUsername = true
										break
									}
								}
							}
						}
					}
				}

				// Only prompt if no default username was found
				if !hasDefaultUsername {
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter username: ")
					input, _ := reader.ReadString('\n')
					username = strings.TrimSpace(input)
				}
			}

			return utils.StartChat(id, username, verbose)
		}

		// Send message mode
		if len(args) > 1 {
			message := args[1]
			return utils.SendMessage(id, message, verbose)
		}

		// Retrieve mode (no message, no chat flag)
		return utils.RetrieveMessage(id, verbose)
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&chatMode, "chat", "c", false, "Enter chat mode")
	rootCmd.Flags().BoolVarP(&listenMode, "listen", "l", false, "Listen for a new message")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output with relay status")
	rootCmd.Flags().BoolVarP(&generateConfig, "generate-config", "g", false, "Generate pulse.conf with default settings")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
