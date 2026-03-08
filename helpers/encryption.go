package helpers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/rerolldrinks/messageservice/models"
)

// EncryptMessage serializes the request to JSON then AES-256-CBC encrypts it.
// Key = SHA256(encryptionKey). The random IV is prepended to the ciphertext
// before Base64 encoding — matching the C# EncryptionHelper exactly.
func EncryptMessage(req models.DecryptedMessageRequest, encryptionKey string) (string, error) {
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message request: %w", err)
	}

	key := sha256.Sum256([]byte(encryptionKey))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	padded := pkcs7Pad(jsonBytes, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	return base64.StdEncoding.EncodeToString(append(iv, ciphertext...)), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}
