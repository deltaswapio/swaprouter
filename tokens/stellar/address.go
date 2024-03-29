package stellar

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/deltaswapio/swaprouter/v3/common"
	"github.com/stellar/go/strkey"
)

// IsValidAddress check address
func (b *Bridge) IsValidAddress(addr string) bool {
	_, err := strkey.Decode(strkey.VersionByteAccountID, addr)
	return err == nil
}

// PublicKeyToAddress impl
func (b *Bridge) PublicKeyToAddress(pubKey string) (string, error) {
	return PublicKeyHexToAddress(pubKey)
}

// PublicKeyHexToAddress convert public key hex to stellar address
func PublicKeyHexToAddress(pubKeyHex string) (string, error) {
	pubKey := pubKeyHex
	if common.HasHexPrefix(pubKey) {
		pubKey = pubKey[2:]
	}
	pub, err := hex.DecodeString(pubKey)
	if err != nil {
		return "", err
	}
	if len(pub) == ed25519.PublicKeySize+1 && pub[0] == 0xED {
		return PublicKeyToAddress(pub[1:])
	}
	if len(pub) == ed25519.PublicKeySize {
		return PublicKeyToAddress(pub)
	}
	return "", fmt.Errorf("public key format error : %v", pubKeyHex)
}

// PublicKeyToAddress public key to address
func PublicKeyToAddress(pubkey []byte) (string, error) {
	pubkeyAddr, err := strkey.Encode(strkey.VersionByteAccountID, pubkey)
	if err != nil {
		return "", err
	}
	return pubkeyAddr, nil
}

// VerifyMPCPubKey verify mpc address and public key is matching
func VerifyMPCPubKey(mpcAddress, mpcPubkey string) error {
	pubkeyAddr, err := PublicKeyHexToAddress(mpcPubkey)
	if err != nil {
		return err
	}
	if !strings.EqualFold(pubkeyAddr, mpcAddress) {
		return fmt.Errorf("mpc address %v and public key address %v is not match", mpcAddress, pubkeyAddr)
	}
	return nil
}

// FormatPublicKeyToPureHex format public key, get rid of hex prefix and ED prefix
func FormatPublicKeyToPureHex(pubKeyHex string) (string, error) {
	pubKey := pubKeyHex
	if common.HasHexPrefix(pubKey) {
		pubKey = pubKey[2:]
	}
	pub, err := hex.DecodeString(pubKey)
	if err != nil {
		return "", err
	}
	if len(pub) == ed25519.PublicKeySize+1 && pub[0] == 0xED {
		return pubKey[2:], nil
	}
	if len(pub) == ed25519.PublicKeySize {
		return pubKey, nil
	}
	return "", fmt.Errorf("public key format error : %v", pubKeyHex)
}
