package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"fmt"
	"io"
)

// CreateNewSharedKey translates CryptManager#createNewSharedKey.
func CreateNewSharedKey() ([]byte, error) {
	key := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// CreateNewKeyPair translates CryptManager#createNewKeyPair.
func CreateNewKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 1024)
}

// GetServerIDHash translates CryptManager#getServerIdHash.
func GetServerIDHash(serverID string, publicKeyDER []byte, sharedKey []byte) []byte {
	h := sha1.New()
	h.Write([]byte(serverID))
	h.Write(sharedKey)
	h.Write(publicKeyDER)
	return h.Sum(nil)
}

// DecodePublicKey translates CryptManager#decodePublicKey.
func DecodePublicKey(publicKeyDER []byte) (*rsa.PublicKey, error) {
	pk, err := x509.ParsePKIXPublicKey(publicKeyDER)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := pk.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("decoded key is not RSA public key")
	}
	return rsaKey, nil
}

// DecryptSharedKey translates CryptManager#decryptSharedKey.
func DecryptSharedKey(privateKey *rsa.PrivateKey, encrypted []byte) ([]byte, error) {
	return DecryptData(privateKey, encrypted)
}

// EncryptData translates CryptManager#encryptData.
func EncryptData(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
}

// DecryptData translates CryptManager#decryptData.
func DecryptData(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, data)
}

type cfb8 struct {
	block   cipher.Block
	state   []byte
	encrypt bool
}

func newCFB8(block cipher.Block, iv []byte, encrypt bool) (*cfb8, error) {
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("iv length mismatch: got=%d want=%d", len(iv), block.BlockSize())
	}
	s := make([]byte, len(iv))
	copy(s, iv)
	return &cfb8{
		block:   block,
		state:   s,
		encrypt: encrypt,
	}, nil
}

func (c *cfb8) XORKeyStream(dst, src []byte) {
	if len(dst) < len(src) {
		panic("cfb8: output smaller than input")
	}

	tmp := make([]byte, c.block.BlockSize())
	for i := 0; i < len(src); i++ {
		c.block.Encrypt(tmp, c.state)

		in := src[i]
		out := in ^ tmp[0]
		dst[i] = out

		copy(c.state, c.state[1:])
		if c.encrypt {
			c.state[len(c.state)-1] = out
		} else {
			c.state[len(c.state)-1] = in
		}
	}
}

func newAESCFB8Streams(sharedKey []byte) (cipher.Stream, cipher.Stream, error) {
	block, err := aes.NewCipher(sharedKey)
	if err != nil {
		return nil, nil, err
	}
	iv := make([]byte, 16)
	copy(iv, sharedKey)

	enc, err := newCFB8(block, iv, true)
	if err != nil {
		return nil, nil, err
	}
	dec, err := newCFB8(block, iv, false)
	if err != nil {
		return nil, nil, err
	}
	return enc, dec, nil
}

type streamWriter struct {
	w      io.Writer
	stream cipher.Stream
}

func (w *streamWriter) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	w.stream.XORKeyStream(buf, p)
	return w.w.Write(buf)
}

type streamReader struct {
	r      io.Reader
	stream cipher.Stream
}

func (r *streamReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.stream.XORKeyStream(p[:n], p[:n])
	}
	return n, err
}

// EncryptOutputStream translates CryptManager#encryptOuputStream.
func EncryptOutputStream(sharedKey []byte, output io.Writer) (io.Writer, error) {
	enc, _, err := newAESCFB8Streams(sharedKey)
	if err != nil {
		return nil, err
	}
	return &streamWriter{w: output, stream: enc}, nil
}

// DecryptInputStream translates CryptManager#decryptInputStream.
func DecryptInputStream(sharedKey []byte, input io.Reader) (io.Reader, error) {
	_, dec, err := newAESCFB8Streams(sharedKey)
	if err != nil {
		return nil, err
	}
	return &streamReader{r: input, stream: dec}, nil
}
