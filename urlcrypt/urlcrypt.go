package urlcrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/Djoulzy/Tools/clog"
)

type Cypher struct {
	HASH_SIZE int // Should be 8
	HEX_KEY   []byte
	HEX_IV    []byte
}

func (uc *Cypher) GenIV_bin() []byte {
	iv_bin := make([]byte, 16)

	newRand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for i := 0; i < 16; i++ {
		iv_bin[i] = byte(newRand.Intn(256))
	}

	return iv_bin
}

func (uc *Cypher) GenIV_hex() []byte {
	bin := uc.GenIV_bin()

	dst := make([]byte, hex.EncodedLen(len(bin)))
	hex.Encode(dst, bin)
	return dst
}

func (*Cypher) pkcs7pad(data []byte, blockSize int) []byte {

	var paddingCount int

	if paddingCount = blockSize - (len(data) % blockSize); paddingCount == 0 {
		paddingCount = blockSize
	}

	return append(data, bytes.Repeat([]byte{byte(paddingCount)}, paddingCount)...)
}

// RemovePkcs7 removes pkcs7 padding from previously padded byte array
func (uc *Cypher) pkcs7unpad(padded []byte, blockSize int) []byte {

	dataLen := len(padded)
	paddingCount := int(padded[dataLen-1])

	if paddingCount > blockSize || paddingCount <= 0 {
		return padded //data is not padded (or not padded correctly), return as is
	}

	padding := padded[dataLen-paddingCount : dataLen-1]

	for _, b := range padding {
		if int(b) != paddingCount {
			return padded //data is not padded (or not padded correcly), return as is
		}
	}

	return padded[:len(padded)-paddingCount] //return data - padding
}

func (uc *Cypher) encodeBase64(b []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(b))
}

func (uc *Cypher) decodeBase64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		clog.Fatal("Crypt", "decodeBase64", err)
	}
	return data
}

func (uc *Cypher) GetMD5Hash(text string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(text))
	md5 := hasher.Sum(nil)

	return md5
}

func (uc *Cypher) Encrypt_b64(text string) ([]byte, error) {
	var iv_bin []byte

	key_bin := make([]byte, hex.DecodedLen(len(uc.HEX_KEY)))
	hex.Decode(key_bin, uc.HEX_KEY)

	if len(uc.HEX_IV) == 0 {
		iv_bin = uc.GenIV_bin()
	} else {
		iv_bin = make([]byte, hex.DecodedLen(len(uc.HEX_IV)))
		hex.Decode(iv_bin, uc.HEX_IV)
	}

	block, err := aes.NewCipher(key_bin)
	if err != nil {
		return nil, err
	}

	textHash := (uc.GetMD5Hash(text))[0:8]

	signedText := append(textHash, []byte(text)...)
	text_padded := uc.pkcs7pad(signedText, aes.BlockSize)

	cbc := cipher.NewCBCEncrypter(block, iv_bin)
	cbc.CryptBlocks(text_padded, text_padded)

	iv_b64 := bytes.TrimRight(bytes.Replace(bytes.Replace(uc.encodeBase64(iv_bin), []byte{'/'}, []byte{'_'}, -1), []byte{'+'}, []byte{'-'}, -1), "=")
	text_b64 := bytes.TrimRight(bytes.Replace(bytes.Replace(uc.encodeBase64(text_padded), []byte{'/'}, []byte{'_'}, -1), []byte{'+'}, []byte{'-'}, -1), "=")

	return append(append(iv_b64, '/'), text_b64...), nil
}

func (uc *Cypher) Decrypt_b64(enc_text string) ([]byte, error) {
	encoded_str := strings.Split(enc_text, "/")
	if len(encoded_str) < 2 {
		return nil, errors.New("Bad string scheme")
	}

	padder := "===="
	iv_pad := ""
	if len(encoded_str[0])%4 > 0 {
		iv_pad = padder[len(encoded_str[0])%4:]
	}
	text_pad := ""
	if len(encoded_str[1])%4 > 0 {
		text_pad = padder[len(encoded_str[1])%4:]
	}

	iv_b64 := strings.Replace(strings.Replace(encoded_str[0], "_", "/", -1), "-", "+", -1) + iv_pad
	text_b64 := strings.Replace(strings.Replace(encoded_str[1], "_", "/", -1), "-", "+", -1) + text_pad

	clog.Info("Crypt", "Decrypt_b64", "IV_B64: %s -- TXT_B64: %s", iv_b64, text_b64)

	key_bin := make([]byte, hex.DecodedLen(len(uc.HEX_KEY)))
	hex.Decode(key_bin, uc.HEX_KEY)
	iv_bin := uc.decodeBase64(iv_b64)
	text_bin := uc.decodeBase64(text_b64)
	block, err := aes.NewCipher(key_bin)
	if err != nil {
		return nil, err
	}

	// iv_bin = iv_bin[:aes.BlockSize]
	// text_bin = text_bin[aes.BlockSize:]
	if len(text_bin)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	cbc := cipher.NewCBCDecrypter(block, iv_bin)
	cbc.CryptBlocks(text_bin, text_bin)
	text_unpadded := uc.pkcs7unpad(text_bin, aes.BlockSize)
	text_unpadded = text_unpadded[uc.HASH_SIZE:]

	return text_unpadded, nil
}
