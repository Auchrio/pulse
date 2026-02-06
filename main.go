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

var rootCmd = &cobra.Command{
	Use:   "pulse <id> [message]",
	Short: "Encrypted messaging via Nostr",
	Long:  "Pulse: Send and receive encrypted messages using Nostr relays",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return utils.ListenForMessage(id)
		}

		// Chat mode
		if chatMode {
			username := ""
			if len(args) > 1 {
				username = args[1]
			} else {
				// Prompt for username
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter username: ")
				input, _ := reader.ReadString('\n')
				username = strings.TrimSpace(input)
			}

			return utils.StartChat(id, username)
		}

		// Send message mode
		if len(args) > 1 {
			message := args[1]
			return utils.SendMessage(id, message)
		}

		// Retrieve mode (no message, no chat flag)
		return utils.RetrieveMessage(id)
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&chatMode, "chat", "c", false, "Enter chat mode")
	rootCmd.Flags().BoolVarP(&listenMode, "listen", "l", false, "Listen for a new message")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
