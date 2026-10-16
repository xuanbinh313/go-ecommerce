package memory

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

const (
	xxPrime1 uint64 = 11400714785074694791
	xxPrime2 uint64 = 14029467366897019727
	xxPrime3 uint64 = 1609587929392839161
	xxPrime4 uint64 = 9650029242287828579
	xxPrime5 uint64 = 2870177450012600261
)

func xxRound(acc, input uint64) uint64 {
	acc += input * xxPrime2
	acc = bits.RotateLeft64(acc, 31)
	return acc * xxPrime1
}
func xxMerge(acc, value uint64) uint64 { acc ^= xxRound(0, value); return acc*xxPrime1 + xxPrime4 }

// Hash matches the source loader's xxhash-wasm h64ToString (XXH64, seed 0).
func Hash(input string) string {
	b := []byte(input)
	i := 0
	var h uint64
	if len(b) >= 32 {
		v1 := xxPrime1
		v1 += xxPrime2 // XXH64 arithmetic intentionally wraps modulo 2^64.
		v2, v3, v4 := xxPrime2, uint64(0), uint64(0)
		v4 -= xxPrime1
		for i <= len(b)-32 {
			v1 = xxRound(v1, binary.LittleEndian.Uint64(b[i:]))
			v2 = xxRound(v2, binary.LittleEndian.Uint64(b[i+8:]))
			v3 = xxRound(v3, binary.LittleEndian.Uint64(b[i+16:]))
			v4 = xxRound(v4, binary.LittleEndian.Uint64(b[i+24:]))
			i += 32
		}
		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) + bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		h = xxMerge(h, v1)
		h = xxMerge(h, v2)
		h = xxMerge(h, v3)
		h = xxMerge(h, v4)
	} else {
		h = xxPrime5
	}
	h += uint64(len(b))
	for i <= len(b)-8 {
		k := xxRound(0, binary.LittleEndian.Uint64(b[i:]))
		h ^= k
		h = bits.RotateLeft64(h, 27)*xxPrime1 + xxPrime4
		i += 8
	}
	if i <= len(b)-4 {
		h ^= uint64(binary.LittleEndian.Uint32(b[i:])) * xxPrime1
		h = bits.RotateLeft64(h, 23)*xxPrime2 + xxPrime3
		i += 4
	}
	for i < len(b) {
		h ^= uint64(b[i]) * xxPrime5
		h = bits.RotateLeft64(h, 11) * xxPrime1
		i++
	}
	h ^= h >> 33
	h *= xxPrime2
	h ^= h >> 29
	h *= xxPrime3
	h ^= h >> 32
	return fmt.Sprintf("%016x", h)
}
