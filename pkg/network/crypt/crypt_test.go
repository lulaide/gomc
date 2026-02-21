package crypt

import (
	"bytes"
	"crypto/x509"
	"io"
	"testing"
)

func TestCreateSharedKeyLength(t *testing.T) {
	key, err := CreateNewSharedKey()
	if err != nil {
		t.Fatalf("CreateNewSharedKey failed: %v", err)
	}
	if len(key) != 16 {
		t.Fatalf("shared key length mismatch: got=%d want=16", len(key))
	}
}

func TestRSAEncryptDecryptRoundTrip(t *testing.T) {
	priv, err := CreateNewKeyPair()
	if err != nil {
		t.Fatalf("CreateNewKeyPair failed: %v", err)
	}

	msg := []byte("gomc-login-secret")
	encrypted, err := EncryptData(&priv.PublicKey, msg)
	if err != nil {
		t.Fatalf("EncryptData failed: %v", err)
	}
	decrypted, err := DecryptData(priv, encrypted)
	if err != nil {
		t.Fatalf("DecryptData failed: %v", err)
	}
	if !bytes.Equal(decrypted, msg) {
		t.Fatalf("roundtrip mismatch: got=%x want=%x", decrypted, msg)
	}
}

func TestDecodePublicKey(t *testing.T) {
	priv, err := CreateNewKeyPair()
	if err != nil {
		t.Fatalf("CreateNewKeyPair failed: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey failed: %v", err)
	}

	decoded, err := DecodePublicKey(der)
	if err != nil {
		t.Fatalf("DecodePublicKey failed: %v", err)
	}
	if decoded.N.Cmp(priv.PublicKey.N) != 0 || decoded.E != priv.PublicKey.E {
		t.Fatal("decoded public key mismatch")
	}
}

func TestGetServerIDHashDeterministic(t *testing.T) {
	a := GetServerIDHash("server", []byte{1, 2, 3}, []byte{4, 5, 6})
	b := GetServerIDHash("server", []byte{1, 2, 3}, []byte{4, 5, 6})
	if !bytes.Equal(a, b) {
		t.Fatal("hash output must be deterministic")
	}
}

func TestAESCFB8StreamRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("hello gomc cfb8 stream")

	var encrypted bytes.Buffer
	encWriter, err := EncryptOutputStream(key, &encrypted)
	if err != nil {
		t.Fatalf("EncryptOutputStream failed: %v", err)
	}
	if _, err := encWriter.Write(plaintext); err != nil {
		t.Fatalf("encrypt write failed: %v", err)
	}

	decReader, err := DecryptInputStream(key, bytes.NewReader(encrypted.Bytes()))
	if err != nil {
		t.Fatalf("DecryptInputStream failed: %v", err)
	}
	decrypted, err := io.ReadAll(decReader)
	if err != nil {
		t.Fatalf("decrypt read failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("cfb8 roundtrip mismatch: got=%q want=%q", decrypted, plaintext)
	}
}
