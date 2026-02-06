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
	"time"

	"github.com/nbd-wtf/go-nostr"
)

var Relays = []string{"wss://relay.damus.io", "wss://nos.lol"}

const UserSecret = "my-app-salt-2026"

// Derives a 32-byte AES key from the ID and Secret
func deriveKey(id string) []byte {
	h := sha256.Sum256([]byte(id + UserSecret))
	return h[:]
}

func encrypt(plaintext string, key []byte) (string, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	// Seal(nonce, nonce...) appends the ciphertext to the nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func decrypt(hexData string, key []byte) (string, error) {
	data, _ := hex.DecodeString(hexData)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	return string(plaintext), err
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Mode (u/d): ")
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(strings.ToLower(mode))

	fmt.Print("Enter ID: ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)

	key := deriveKey(id)
	hashedTag := hex.EncodeToString(key) // Use the key hash as the Nostr tag
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if mode == "u" {
		fmt.Print("Message: ")
		msg, _ := reader.ReadString('\n')
		encryptedMsg, _ := encrypt(strings.TrimSpace(msg), key)

		ev := nostr.Event{
			PubKey:    "anonymous", // placeholder, Sign will overwrite
			CreatedAt: nostr.Now(),
			Kind:      nostr.KindTextNote,
			Tags:      nostr.Tags{{"t", hashedTag}},
			Content:   encryptedMsg,
		}
		sk := nostr.GeneratePrivateKey()
		ev.Sign(sk)

		fmt.Println("Encrypted & Publishing...")
		for _, url := range Relays {
			r, _ := nostr.RelayConnect(ctx, url)
			r.Publish(ctx, ev)
		}
		fmt.Println("✅ Done.")

	} else if mode == "d" {
		fmt.Println("Searching for encrypted message...")
		resultChan := make(chan string)
		for _, url := range Relays {
			go func(u string) {
				r, err := nostr.RelayConnect(ctx, u)
				if err != nil {
					return
				}
				sub, _ := r.Subscribe(ctx, []nostr.Filter{{
					Tags:  nostr.TagMap{"t": []string{hashedTag}},
					Kinds: []int{nostr.KindTextNote},
					Limit: 1,
				}})
				for ev := range sub.Events {
					decrypted, err := decrypt(ev.Content, key)
					if err == nil {
						resultChan <- decrypted
					}
				}
			}(url)
		}

		select {
		case msg := <-resultChan:
			fmt.Printf("\n--- Decrypted Message ---\n%s\n", msg)
		case <-time.After(5 * time.Second):
			fmt.Println("❌ No valid message found.")
		}
	}
}
