package Utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io"
)

// GenerateAsymmetricKeys generates a new RSA public/private key pair
func GenerateAsymmetricKeys(bits int) *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, bits)
	return privateKey
}

// GenerateSymmetricKeys generates a new AES 1024 bit key
func GenerateSymmetricKeys() []byte {
	var cryptoKey = make([]byte, 16)
	rand.Read(cryptoKey)

	return cryptoKey
}

// AsymmetricDecrypt decrypts input using RSA private key
// Returns []byte containing decrypted data
func AsymmetricDecrypt(input []byte, key *rsa.PrivateKey) ([]byte, error) {
	output, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, input, nil)
	return output, err
}

// AsymmetricEncrypt encrypts input using RSA public key
// Returns []byte containing encrypted data
func AsymmetricEncrypt(input []byte, key *rsa.PublicKey) ([]byte, error) {
	output, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, input, nil)
	return output, err
}

// AsymmetricKeyFromString is provided with an RSA public key key in PEM format, and returns a rsa.PublicKey
// usable with other calls
func AsymmetricKeyFromString(input []byte) *rsa.PublicKey {
	block, _ := pem.Decode(input)
	if block == nil {
		return nil
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub
	}

	return nil
}

// AsymmetricKeyToString converts a provided rsa.PublicKey into PEM format for transfer
func AsymmetricKeyToString(input *rsa.PublicKey) []byte {

	pubKey, _ := x509.MarshalPKIXPublicKey(input)

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKey,
	})

	return pemData
}

// SymmetricDecrypt decrypt AES data provided in Base64 format with the provided key
func SymmetricDecrypt(input string, key []byte) (string, error) {
	cipherText, err := base64.URLEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		err = errors.New("Ciphertext block size is too short")
		return "", err
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)

	decodedmess := string(cipherText)
	return decodedmess, nil
}

// SymmetricEncrypt encrypts provided []byte using AES, returning Base64 encoded data
func SymmetricEncrypt(input []byte, key []byte) (string, error) {
	plainText := []byte(input)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//returns to base64 encoded string
	encmess := base64.URLEncoding.EncodeToString(cipherText)
	return encmess, nil

}
