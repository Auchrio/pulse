package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const UserSecret = "super-secret-key"
const HistoryLimit = 5

var Relays = []string{"wss://relay.damus.io", "wss://nos.lol", "wss://relay.snort.social"}

// DeriveKey creates an encryption key from an ID and the user secret
func DeriveKey(id string) []byte {
	h := sha256.Sum256([]byte(id + UserSecret))
	return h[:]
}

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plaintext string, key []byte) (string, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	return hex.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

// Decrypt decrypts hex-encoded ciphertext using AES-256-GCM
func Decrypt(hexData string, key []byte) (string, error) {
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

// FetchHistory retrieves historical messages from relays
func FetchHistory(ctx context.Context, id string, key []byte) ([]*nostr.Event, error) {
	hashedTag := hex.EncodeToString(key)
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
				Limit: HistoryLimit,
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

	return allHistory, nil
}

// PublishEvent publishes an event to all relays
func PublishEvent(ctx context.Context, event nostr.Event) error {
	var wg sync.WaitGroup
	for _, url := range Relays {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			r, err := nostr.RelayConnect(ctx, u)
			if err == nil {
				r.Publish(ctx, event)
				r.Close()
			}
		}(url)
	}
	wg.Wait()
	return nil
}
