package utils

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// SendMessage sends an encrypted message to the given ID
func SendMessage(id string, message string) error {
	key := DeriveKey(id)
	hashedTag := hex.EncodeToString(key)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Encrypt the message
	encrypted, err := Encrypt(message, key)
	if err != nil {
		fmt.Print("Failure - encryption error")
		return err
	}

	// Create nostr event
	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	ev := nostr.Event{
		PubKey:    pk,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"t", hashedTag}},
		Content:   encrypted,
	}
	ev.Sign(sk)

	// Publish to relays
	err = PublishEvent(ctx, ev)
	if err != nil {
		fmt.Print("Failure - publish error")
		return err
	}

	fmt.Print("Success")
	return nil
}
