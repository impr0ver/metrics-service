package crypt

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var waitInitKeys sync.WaitGroup

func TestSignDataWithSHA256(t *testing.T) {

	tests := []struct {
		name      string
		plainText string
		key       string
		want      string
	}{
		{"test #1", "1234567890", "mytestkey", "1b02ac62e12780a078820ff0e4b46054b5a87d579b67d0b9d97ae740767d4d27"},
		{"test #2", "123456789012345678901234567890123456789012345678901234567890", "mytestkey", "49681c3bbfd59c670ee8ee2e74d52a6b8dc41c40ff55076cab8dd8bca0477a3e"},
		{"test #3", "1", "mytestkey", "9eda81ff0223b02a4b70aa60367d8bae1cd71791793e019780fb91a401e55347"},
		{"test #4", "MySecretText", "mytestkey", "b611d3f1104be3638d6eb2b23bc189ac690902a95a71aad6438a4fbbc1c2f061"},
		{"test #5", "", "mytestkey", "c12dbaf0bc80b7c08ea31cc3969ba6fd312c26bc39392dbe160f6fa7f399e375"},
		{"test #6", "", "", "key parameter is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := SignDataWithSHA256([]byte(tt.plainText), tt.key)
			if err != nil {
				assert.ErrorContains(t, err, tt.want)
			} else {
				assert.Equal(t, hash, tt.want)
			}

		})
	}
}

func TestCheckHashSHA256(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want string
	}{
		{"test #1", "1b02ac62e12780a078820ff0e4b46054b5a87d579b67d0b9d97ae740767d4d27", "1b02ac62e12780a078820ff0e4b46054b5a87d579b67d0b9d97ae740767d4d27"},
		{"test #2", "49681c3bbfd59c670ee8ee2e74d52a6b8dc41c40ff55076cab8dd8bca0477a3e", "49681c3bbfd59c670ee8ee2e74d52a6b8dc41c40ff55076cab8dd8bca0477a3e"},
		{"test #3", "9eda81ff0223b02a4b70aa60367d8bae1cd71791793e019780fb91a401e55347", "9eda81ff0223b02a4b70aa60367d8bae1cd71791793e019780fb91a401e55347"},
		{"test #4", "12345", "12345"},
		{"test #5", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := CheckHashSHA256(tt.hash, tt.want)
			assert.Equal(t, res, true)
		})
	}
}

func TestInitKeys(t *testing.T) {
	waitInitKeys.Add(1)
	err := GenKeys("./")
	assert.NoError(t, err, "GenKeys() failed")
	waitInitKeys.Done()
}

func TestEncryptPKCS1v15(t *testing.T) {
	waitInitKeys.Wait()

	pubKey, err := InitPublicKey("./public.pem")
	assert.NoError(t, err, "InitPublicKey() failed")

	privKey, err := InitPrivateKey("./private.pem")
	assert.NoError(t, err, "InitPrivateKey() failed")

	plainText := ([]byte)("GO - best courses in Yandex Praktikum!!!!!!")

	cipherText, err := EncryptPKCS1v15(pubKey, plainText)
	assert.NoError(t, err, "EncryptMsg() failed")
	t.Logf("Encrypt success! result is: %s\n", cipherText)
	
	decryptText, err := DecryptPKCS1v15(privKey, cipherText)
	assert.NoError(t, err, "DecryptMsg() failed")

	value := bytes.Equal(plainText, decryptText)
	assert.True(t, value, "DecryptMsg() failed, decripted text is not equal plain text")

	if string(decryptText) != string(plainText) {
		t.Fatalf("plainText(%s) and decrypted(%s) are not same", plainText, decryptText)
	}

	os.Remove("./public.pem")
	os.Remove("./private.pem")
}

func TestEncryptPKCS1v15_negative(t *testing.T) {
	waitInitKeys.Wait()

	pubKey, err := InitPublicKey("./noname.pem")
	assert.Error(t, err, "open ./noname.pem: no such file or directory")

	privKey, err := InitPrivateKey("./noname2.pem")
	assert.Error(t, err, "open ./noname2.pem: no such file or directory")

	t.Logf("params: %v\n%v\n", pubKey, privKey)


	os.Remove("./public.pem")
	os.Remove("./private.pem")
}
