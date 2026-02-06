package utils

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// StartChat enters interactive chat mode for the given ID
func StartChat(id string, username string) error {
	key := DeriveKey(id)
	hashedTag := hex.EncodeToString(key)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	// Fetch history first
	history, err := FetchHistory(ctx, id, key)
	if err != nil {
		return err
	}

	seenEvents := make(map[string]bool)

	// Print history
	for _, ev := range history {
		if seenEvents[ev.ID] {
			continue
		}
		seenEvents[ev.ID] = true

		msg, err := Decrypt(ev.Content, key)
		if err == nil {
			fmt.Println(msg)
		}
	}

	fmt.Printf("--- Connected as [%s] ---\n", username)

	var listenMu sync.Mutex

	// Start live listener
	for _, url := range Relays {
		go func(u string) {
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}

			now := nostr.Now()
			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{hashedTag}},
				Kinds: []int{nostr.KindTextNote},
				Since: &now,
			}})

			for ev := range sub.Events {
				listenMu.Lock()
				if seenEvents[ev.ID] {
					listenMu.Unlock()
					continue
				}
				seenEvents[ev.ID] = true
				listenMu.Unlock()

				msg, err := Decrypt(ev.Content, key)
				if err == nil && ev.PubKey != pk {
					fmt.Printf("\r\033[K%s\n> ", msg)
				}
			}
		}(url)
	}

	// Sender loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		text := strings.TrimSpace(input)
		if text == "" {
			continue
		}

		timestamp := time.Now().Format("15:04")
		formattedMsg := fmt.Sprintf("[%s] %s: %s", timestamp, username, text)

		encrypted, _ := Encrypt(formattedMsg, key)
		ev := nostr.Event{
			PubKey:    pk,
			CreatedAt: nostr.Now(),
			Kind:      nostr.KindTextNote,
			Tags:      nostr.Tags{{"t", hashedTag}},
			Content:   encrypted,
		}
		ev.Sign(sk)

		listenMu.Lock()
		seenEvents[ev.ID] = true
		listenMu.Unlock()

		// Clear line and print our formatted message
		fmt.Printf("\033[A\033[K%s\n", formattedMsg)

		PublishEvent(ctx, ev)
	}
}
