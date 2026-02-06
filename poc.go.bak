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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// --- CONFIGURATION ---
var Relays = []string{"wss://relay.damus.io", "wss://nos.lol", "wss://relay.snort.social"}

const UserSecret = "super-secret-key"
const HistoryLimit = 5

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
	return hex.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
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

	fmt.Printf("\n--- Loading up to %d historical messages for ID: %s ---\n", HistoryLimit, id)

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	// --- HISTORY FETCHING ---
	var allHistory []*nostr.Event
	var histMu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range Relays {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}

			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{hashedTag}},
				Kinds: []int{nostr.KindTextNote},
				Limit: HistoryLimit, // Uses the hardcoded variable
			}})

			timeout := time.After(1500 * time.Millisecond)
		Loop:
			for {
				select {
				case ev := <-sub.Events:
					histMu.Lock()
					allHistory = append(allHistory, ev)
					histMu.Unlock()
				case <-timeout:
					break Loop
				}
			}
		}(url)
	}
	wg.Wait()

	// Sort History: Oldest to Newest
	sort.Slice(allHistory, func(i, j int) bool {
		return allHistory[i].CreatedAt < allHistory[j].CreatedAt
	})

	// Print History (with de-duplication)
	for _, ev := range allHistory {
		seenMu.Lock()
		if seenEvents[ev.ID] {
			seenMu.Unlock()
			continue
		}
		seenEvents[ev.ID] = true
		seenMu.Unlock()

		msg, err := decrypt(ev.Content, key)
		if err == nil {
			fmt.Println(msg)
		}
	}

	fmt.Printf("--- Connected as [%s] ---\n", username)

	// LIVE LISTENER
	for _, url := range Relays {
		go func(u string) {
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}

			// We use a custom pointer helper because nostr.TimestampPtr might vary by version
			now := nostr.Now()
			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{hashedTag}},
				Kinds: []int{nostr.KindTextNote},
				Since: &now,
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

		// ANSI Clear line and print our formatted message
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
