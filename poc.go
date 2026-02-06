package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const (
	PublicRelay = "wss://relay.damus.io" // A popular public relay
	UserSecret  = "my-app-salt-2024"     // Hardcoded salt to obfuscate IDs
)

// Helper to turn a simple ID into a 64-character hex tag
func hashID(id string) string {
	h := sha256.New()
	h.Write([]byte(id + UserSecret))
	return hex.EncodeToString(h.Sum(nil))
}

func main() {
	ctx := context.Background()
	userID := "101"
	msgContent := "Hello World"
	hashedTag := hashID(userID)

	// 1. Setup Identity (Invisible to user)
	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	// 2. Connect to Relay
	relay, err := nostr.RelayConnect(ctx, PublicRelay)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Connected to %s\n", PublicRelay)

	// --- UPLOAD SECTION ---
	fmt.Printf("Uploading message to ID: %s (Tag: %s...)\n", userID, hashedTag[:8])

	ev := nostr.Event{
		PubKey:    pk,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"t", hashedTag}},
		Content:   msgContent,
	}
	ev.Sign(sk)

	err = relay.Publish(ctx, ev)
	if err != nil {
		log.Fatal(err)
	}

	// Provide a link to view the raw data on a Nostr explorer
	fmt.Printf("Message Sent! View raw event at: https://nostr.band/note/%s\n", ev.ID)
	fmt.Println("--------------------------------------------------")

	// --- DOWNLOAD SECTION ---
	fmt.Println("Attempting to download message back...")

	// Filter for the hashed tag
	filter := nostr.Filter{
		Tags:  nostr.TagMap{"t": []string{hashedTag}},
		Kinds: []int{nostr.KindTextNote},
		Limit: 1,
	}

	sub, err := relay.Subscribe(ctx, []nostr.Filter{filter})
	if err != nil {
		log.Fatal(err)
	}

	// Wait for the event to come back
	select {
	case receivedEv := <-sub.Events:
		fmt.Printf("Success! Received Content: %s\n", receivedEv.Content)
		fmt.Printf("Verified Publisher: %s\n", receivedEv.PubKey)
	case <-time.After(5 * time.Second):
		fmt.Println("Timed out waiting for message.")
	}
}
