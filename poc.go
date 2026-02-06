package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

var Relays = []string{"wss://relay.damus.io", "wss://nos.lol", "wss://relay.snort.social"}

const UserSecret = "my-app-salt-2026"

var (
	seenEvents = make(map[string]bool)
	seenMu     sync.Mutex
)

// --- CRYPTO HELPERS ---

func deriveKey(id string) []byte {
	h := sha256.Sum256([]byte(id + UserSecret))
	return h[:]
}

func encrypt(plaintext string, key []byte) (string, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func decrypt(hexData string, key []byte) (string, error) {
	data, err := hex.DecodeString(hexData)
	if err != nil {
		return "", err
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}

// --- MAIN ---

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Display Name: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter Chat ID: ")
	idInput, _ := reader.ReadString('\n')
	id := strings.TrimSpace(idInput)

	key := deriveKey(id)
	hashedTag := hex.EncodeToString(key)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("\n--- Connected as [%s] on ID: %s ---\n", username, id)

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	// BACKGROUND LISTENER
	for _, url := range Relays {
		go func(u string) {
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}
			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{hashedTag}},
				Kinds: []int{nostr.KindTextNote},
				Limit: 20,
			}})

			for ev := range sub.Events {
				seenMu.Lock()
				if seenEvents[ev.ID] {
					seenMu.Unlock()
					continue
				}
				seenEvents[ev.ID] = true
				seenMu.Unlock()

				msg, err := decrypt(ev.Content, key)
				// We only print if it's from a partner
				if err == nil && ev.PubKey != pk {
					fmt.Printf("\r\033[K%s\n> ", msg)
				}
			}
		}(url)
	}

	// SENDER LOOP
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		text := strings.TrimSpace(input)
		if text == "" {
			continue
		}

		// Format the message: [15:04] user: message
		timestamp := time.Now().Format("15:04")
		formattedMsg := fmt.Sprintf("[%s] %s: %s", timestamp, username, text)

		encrypted, _ := encrypt(formattedMsg, key)
		ev := nostr.Event{
			PubKey:    pk,
			CreatedAt: nostr.Now(),
			Kind:      nostr.KindTextNote,
			Tags:      nostr.Tags{{"t", hashedTag}},
			Content:   encrypted,
		}
		ev.Sign(sk)

		seenMu.Lock()
		seenEvents[ev.ID] = true
		seenMu.Unlock()

		// Print our own message immediately so it looks local
		fmt.Printf("\033[A\033[K%s\n", formattedMsg)

		for _, url := range Relays {
			go func(u string) {
				r, err := nostr.RelayConnect(ctx, u)
				if err == nil {
					r.Publish(ctx, ev)
					r.Close()
				}
			}(url)
		}
	}
}
