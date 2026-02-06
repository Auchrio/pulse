package utils

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// ListenForMessage listens for a new message on the given ID and prints it
// timeout is in seconds, 0 means no timeout
func ListenForMessage(id string, verbose bool, timeoutSeconds int) error {
	key := DeriveKey(id)
	hashedTag := hex.EncodeToString(key)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	var messageReceived sync.WaitGroup
	messageReceived.Add(1)
	var foundMessage bool

	// Start listening from now
	for _, url := range Relays {
		go func(u string) {
			if foundMessage {
				return
			}

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
				// Only accept messages from others, not from ourselves
				if ev.PubKey != pk {
					msg, err := Decrypt(ev.Content, key)
					if err == nil {
						fmt.Print(msg)
						foundMessage = true
						messageReceived.Done()
						cancel()
						return
					}
				}
			}
		}(url)
	}

	// Wait for a message with timeout
	done := make(chan struct{})
	go func() {
		messageReceived.Wait()
		close(done)
	}()

	// Handle timeout
	if timeoutSeconds == 0 {
		// No timeout - wait indefinitely
		<-done
		return nil
	} else {
		// Wait with timeout
		select {
		case <-done:
			return nil
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			return fmt.Errorf("no message received within timeout")
		}
	}
}
