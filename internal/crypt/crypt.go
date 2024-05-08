package crypt

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path"
)

func SignDataWithSHA256(plainText []byte, key string) (string, error) {
	if key != "" {
		h := hmac.New(sha256.New, []byte(key))
		h.Write(plainText)
		s := h.Sum(nil)
		h.Reset()
		return hex.EncodeToString(s), nil
	}
	return "", fmt.Errorf("key parameter is empty")
}

func CheckHashSHA256(resultHash, reqHash string) bool {
	return hmac.Equal([]byte(resultHash), []byte(reqHash))
}

// GenKeys - generates a public and private key pair and stores them in the outdir directory.
func GenKeys(outdir string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return err
	}

	publicKey := &privateKey.PublicKey
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	err = os.WriteFile(path.Join(outdir, "private.pem"), privateKeyPEM, 0644)
	if err != nil {
		return err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	err = os.WriteFile(path.Join(outdir, "public.pem"), publicKeyPEM, 0644)
	if err != nil {
		return err
	}

	return nil
}

// InitPublicKey - takes the path to the public key and returns publicKey.
func InitPublicKey(p string) (publicKey *rsa.PublicKey, err error) {
	publicKeyPEM, err := os.ReadFile(p)
	if err != nil {
		return publicKey, err
	}
	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
	pk, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	publicKey = pk.(*rsa.PublicKey)
	return
}

// InitPrivateKey - takes the path to the private key and returns privateKey.
func InitPrivateKey(p string) (privateKey *rsa.PrivateKey, err error) {
	privateKeyPEM, err := os.ReadFile(p)
	if err != nil {
		return
	}
	privateKeyBlock, _ := pem.Decode(privateKeyPEM)
	privateKey, err = x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	return
}

// split - divides buf into limit size.
func split(buf []byte, limit int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/limit+1)
	for len(buf) >= limit {
		chunk, buf = buf[:limit], buf[limit:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}
	return chunks
}

// EncryptPKCS1v15 the given message with RSA and the padding scheme from PKCS #1 v1.5.
func EncryptPKCS1v15(key *rsa.PublicKey, plainText []byte) ([]byte, error) {
	partLen := key.Size() - 11
	chunks := split(plainText, partLen)

	//may be replace on: buffer := bytes.NewBufferString("")
	ciphertext := make([]byte, 0, len(plainText)/partLen+1)
	buffer := bytes.NewBuffer(ciphertext)

	for _, chunk := range chunks {
		encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, key, chunk)
		if err != nil {
			return encrypted, err
		}
		_, err = buffer.Write(encrypted)
		if err != nil {
			return encrypted, err
		}
	}
	return buffer.Bytes(), nil
}

// DecryptPKCS1v15 the cipher text using RSA and the padding scheme from PKCS #1 v1.5.
func DecryptPKCS1v15(key *rsa.PrivateKey, cipherText []byte) ([]byte, error) {
	partLen := key.Size()
	chunks := split(cipherText, partLen)

	//may be replace on: buffer := bytes.NewBufferString("")
	plaintext := make([]byte, 0, len(cipherText))
	buffer := bytes.NewBuffer(plaintext)
	
	for _, chunk := range chunks {
		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, key, chunk)
		if err != nil {
			return decrypted, err
		}
		_, err = buffer.Write(decrypted)
		if err != nil {
			return decrypted, err
		}
	}

	return buffer.Bytes(), nil
}
