package utils

import (
	"context"
	"fmt"
	"strings"
)

// RetrieveMessage gets the most recent message from the given ID
func RetrieveMessage(id string) error {
	key := DeriveKey(id)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	history, err := FetchHistory(ctx, id, key)
	if err != nil {
		return err
	}

	if len(history) == 0 {
		return fmt.Errorf("no messages found")
	}

	// Get the most recent message
	mostRecent := history[len(history)-1]
	msg, err := Decrypt(mostRecent.Content, key)
	if err != nil {
		return err
	}

	fmt.Print(strings.TrimRight(msg, "\n"))
	return nil
}
