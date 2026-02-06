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
func FetchHistory(ctx context.Context, id string, key []byte, verbose bool) ([]*nostr.Event, error) {
	hashedTag := hex.EncodeToString(key)
	var allHistory []*nostr.Event
	var histMu sync.Mutex
	var wg sync.WaitGroup

	tracker := NewStatusTracker(verbose)

	// Create a longer timeout context for relay connections + message wait
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for _, url := range Relays {
		tracker.AddRelay(url)
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			tracker.UpdateStatus(u, "pending")
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				tracker.UpdateStatusWithReason(u, "error", err.Error())
				return
			}

			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{hashedTag}},
				Kinds: []int{nostr.KindTextNote},
				Limit: HistoryLimit,
			}})

			// Wait maximum 300ms for messages from this relay
			timeout := time.After(300 * time.Millisecond)
		Loop:
			for {
				select {
				case ev := <-sub.Events:
					histMu.Lock()
					allHistory = append(allHistory, ev)
					histMu.Unlock()
					tracker.UpdateStatusWithReason(u, "success", "message retrieved")
					break Loop
				case <-timeout:
					tracker.UpdateStatusWithReason(u, "cancelled", "300ms timeout reached")
					break Loop
				case <-ctx.Done():
					tracker.UpdateStatusWithReason(u, "cancelled", "context cancelled")
					break Loop
				}
			}
		}(url)
	}
	wg.Wait()

	if verbose {
		tracker.FinalizeStatus()
		tracker.DisplayStatus()
		fmt.Printf("Total time: %dms\n", tracker.GetTotalDuration().Milliseconds())
	}

	// Sort History: Oldest to Newest
	sort.Slice(allHistory, func(i, j int) bool {
		return allHistory[i].CreatedAt < allHistory[j].CreatedAt
	})

	return allHistory, nil
}

// PublishEvent publishes an event to all relays
func PublishEvent(ctx context.Context, event nostr.Event, verbose bool) error {
	var wg sync.WaitGroup

	tracker := NewStatusTracker(verbose)

	for _, url := range Relays {
		tracker.AddRelay(url)
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			tracker.UpdateStatus(u, "pending")
			r, err := nostr.RelayConnect(ctx, u)
			if err == nil {
				err = r.Publish(ctx, event)
				r.Close()
				if err == nil {
					tracker.UpdateStatusWithReason(u, "success", "published")
				} else {
					tracker.UpdateStatusWithReason(u, "error", err.Error())
				}
			} else {
				tracker.UpdateStatusWithReason(u, "error", err.Error())
			}
		}(url)
	}
	wg.Wait()

	if verbose {
		tracker.FinalizeStatus()
		tracker.DisplayStatus()
		fmt.Printf("Total time: %dms\n", tracker.GetTotalDuration().Milliseconds())
	}

	return nil
}
