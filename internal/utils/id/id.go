package id

import (
	"crypto/rand"
	"math/big"
	"time"
)

const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// New returns a ULID-shaped identifier with Medusa's optional entity prefix.
func New(prefix string) string {
	var raw [16]byte
	ms := uint64(time.Now().UnixMilli())
	for i := 5; i >= 0; i-- {
		raw[i] = byte(ms)
		ms >>= 8
	}
	if _, err := rand.Read(raw[6:]); err != nil {
		n := uint64(time.Now().UnixNano())
		for i := 15; i >= 6; i-- {
			raw[i] = byte(n)
			n >>= 8
		}
	}
	n := new(big.Int).SetBytes(raw[:])
	base := big.NewInt(32)
	rem := new(big.Int)
	encoded := make([]byte, 26)
	for i := 25; i >= 0; i-- {
		n.DivMod(n, base, rem)
		encoded[i] = alphabet[rem.Int64()]
	}
	value := string(encoded)
	if prefix != "" {
		return prefix + "_" + value
	}
	return value
}
