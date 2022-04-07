package Crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"os"

	// "strings"
	"fmt"
	"io/ioutil"
)

type PrivateKey *ecdsa.PrivateKey

// PointsFromDER Deserialises signature from DER encoded format
func PointsFromDER(der []byte) (*big.Int, *big.Int) {
	R, S := &big.Int{}, &big.Int{}
	data := asn1.RawValue{}
	if _, err := asn1.Unmarshal(der, &data); err != nil {
		panic(err.Error())
	}
	// The format of our DER string is 0x02 + rlen + r + 0x02 + slen + s
	rLen := data.Bytes[1] // The entire length of R + offset of 2 for 0x02 and rlen
	r := data.Bytes[2 : rLen+2]
	// Ignore the next 0x02 and slen bytes and just take the start of S to the end of the byte array
	s := data.Bytes[rLen+4:]
	R.SetBytes(r)
	S.SetBytes(s)
	return R, S
}

// PointsToDER serialises signature to DER encoded format
func PointsToDER(r, s *big.Int) []byte {
	// Ensure the encoded bytes for the r and s values are canonical and
	// thus suitable for DER encoding.
	rb := canonicalizeInt(r)
	sb := canonicalizeInt(s)
	// total length of returned signature is 1 byte for each magic and
	// length (6 total), plus lengths of r and s
	length := 6 + len(rb) + len(sb)
	b := make([]byte, length)
	b[0] = 0x30
	b[1] = byte(length - 2)
	b[2] = 0x02
	b[3] = byte(len(rb))
	offset := copy(b[4:], rb) + 4
	b[offset] = 0x02
	b[offset+1] = byte(len(sb))
	copy(b[offset+2:], sb)
	return b
}

//Hash returns the SHA256 hash value
func Hash(b []byte) [32]byte {
	h := sha256.Sum256(b)
	return h
}

//EncodeToHex converts byte slice to the string
func EncodeToHex(data []byte) string {
	return hex.EncodeToString(data)
}

//DecodeToBytes converts string to byte slice
func DecodeToBytes(data string) []byte {
	b, _ := hex.DecodeString(data)
	return b
}

//CheckForKeys checks local availability of public key true if present false if not
func CheckForKeys() bool {
	// Returns true if file is present in the directory
	filename := "PrivateKey.pem"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

//SerializePublicKey serialises the public key to byte slice
func SerializePublicKey(PublicKey *ecdsa.PublicKey) []byte {
	key := elliptic.Marshal(PublicKey, PublicKey.X, PublicKey.Y)
	return key // uncompressed public key in bytes lenghth 65
}

//SerializePrivateKey serialises the private key to byte slice
func SerializePrivateKey(privateKey *ecdsa.PrivateKey) []byte {
	key, _ := x509.MarshalECPrivateKey(privateKey)
	return key
}

//DeserializePublicKey deserialises the public key from byte slice to ecdsa.PublicKey
func DeserializePublicKey(data []byte) *ecdsa.PublicKey {
	var PublicKey ecdsa.PublicKey
	PublicKey.Curve = elliptic.P256()
	PublicKey.X, PublicKey.Y = elliptic.Unmarshal(elliptic.P256(), data)
	return &PublicKey
}

//DeserializePrivateKey deserialises the private key from byte slice to ecdsa.PrivateKey
func DeserializePrivateKey(data []byte) *ecdsa.PrivateKey {
	PrivateKey, _ := x509.ParseECPrivateKey(data)
	return PrivateKey
}

//GenerateKeys generates the private key and stores in a pem file
func GenerateKeys() *ecdsa.PrivateKey {
	// function generates keys and creates a pem file with key stored in that
	PrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println(err)
	}
	serialKey := SerializePrivateKey(PrivateKey)
	file, _ := os.OpenFile("PrivateKey.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: serialKey} // serializing and writing to the pem file
	if err := pem.Encode(file, pemBlock); err != nil {
		fmt.Println(err)
	}
	file.Close()
	return PrivateKey
}

//LoadKeys generates and returns the public key from the private key which is dtored in the pen file
func LoadKeys() *ecdsa.PrivateKey {
	// Loads the key from a pem file in directory
	filename := "PrivateKey.pem"
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	}
	block, _ := pem.Decode(bytes)
	PrivateKey := DeserializePrivateKey(block.Bytes)
	return PrivateKey
}

//Sign signes the hash byte slice with with the private keygiven returns signature
func Sign(hash []byte, key *ecdsa.PrivateKey) []byte {
	// returns the serialized form of the signature
	var signature [72]byte
	if len(hash) != 32 { // check the length of hash
		fmt.Println("Invalid hash")
		return signature[:]
	}
	r, s, _ := ecdsa.Sign(rand.Reader, key, hash)
	copy(signature[:], PointsToDER(r, s))
	return signature[:]
}

//Verify returns true if signature verified false otherwise
func Verify(signature []byte, PublicKey *ecdsa.PublicKey, hash []byte) bool {
	// fmt.Println(len(signature))
	r, s := PointsFromDER(signature)
	v := ecdsa.Verify(PublicKey, hash, r, s)
	return v
}

//canonicalizeInt Converts bigint to byte slice
func canonicalizeInt(val *big.Int) []byte {
	b := val.Bytes()
	if len(b) == 0 {
		b = []byte{0x00}
	}
	if b[0]&0x80 != 0 {
		paddedBytes := make([]byte, len(b)+1)
		copy(paddedBytes[1:], b)
		b = paddedBytes
	}
	return b
}
