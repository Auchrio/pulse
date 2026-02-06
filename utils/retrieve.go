package utils

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RetrieveMessage gets the most recent message from the given ID
func RetrieveMessage(id string, verbose bool) error {
	startTime := time.Now()
	key := DeriveKey(id)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if verbose {
		fmt.Println("Retrieving message...")
	}

	history, err := FetchHistory(ctx, id, key, verbose)
	if err != nil {
		return err
	}

	if len(history) == 0 {
		return fmt.Errorf("no messages found")
	}

	// Get the most recent message (sorted by CreatedAt in ascending order, so last is newest)
	mostRecent := history[len(history)-1]
	msg, err := Decrypt(mostRecent.Content, key)
	if err != nil {
		return err
	}

	fmt.Print(strings.TrimRight(msg, "\n"))

	if verbose {
		fmt.Printf("\nTotal operation time: %dms\n", time.Since(startTime).Milliseconds())
	}

	return nil
}
