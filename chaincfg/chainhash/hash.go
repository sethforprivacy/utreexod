// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chainhash

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// HashSize of array used to store hashes.  See Hash.
const HashSize = 32

// MaxHashStringSize is the maximum length of a Hash hash string.
const MaxHashStringSize = HashSize * 2

var (
	// TagBIP0340Challenge is the BIP-0340 tag for challenges.
	TagBIP0340Challenge = []byte("BIP0340/challenge")

	// TagBIP0340Aux is the BIP-0340 tag for aux data.
	TagBIP0340Aux = []byte("BIP0340/aux")

	// TagBIP0340Nonce is the BIP-0340 tag for nonces.
	TagBIP0340Nonce = []byte("BIP0340/nonce")

	// TagTapSighash is the tag used by BIP 341 to generate the sighash
	// flags.
	TagTapSighash = []byte("TapSighash")

	// TagTagTapLeaf is the message tag prefix used to compute the hash
	// digest of a tapscript leaf.
	TagTapLeaf = []byte("TapLeaf")

	// TagTapBranch is the message tag prefix used to compute the
	// hash digest of two tap leaves into a taproot branch node.
	TagTapBranch = []byte("TapBranch")

	// TagTapTweak is the message tag prefix used to compute the hash tweak
	// used to enable a public key to commit to the taproot branch root
	// for the witness program.
	TagTapTweak = []byte("TapTweak")

	// precomputedTags is a map containing the SHA-256 hash of the BIP-0340
	// tags.
	precomputedTags = map[string]Hash{
		string(TagBIP0340Challenge): sha256.Sum256(TagBIP0340Challenge),
		string(TagBIP0340Aux):       sha256.Sum256(TagBIP0340Aux),
		string(TagBIP0340Nonce):     sha256.Sum256(TagBIP0340Nonce),
		string(TagTapSighash):       sha256.Sum256(TagTapSighash),
		string(TagTapLeaf):          sha256.Sum256(TagTapLeaf),
		string(TagTapBranch):        sha256.Sum256(TagTapBranch),
		string(TagTapTweak):         sha256.Sum256(TagTapTweak),
	}

	// TagUtreexoV1 is the tag used by utreexo v1 serialized hashes to
	// generate the leaf hashes to be committed into the accumulator.
	TagUtreexoV1 = []byte("UtreexoV1")

	precomputedUtreexoTags = map[string][64]byte{
		string(TagUtreexoV1): sha512.Sum512(TagUtreexoV1),
	}

	// UTREEXO_TAG_V1 is the version tag to be prepended to the leafhash. It's just the sha512 hash of the string
	// `UtreexoV1` represented as a vector of [u8] ([85 116 114 101 101 120 111 86 49]).
	// The same tag is "5574726565786f5631" as a hex string.
	UTREEXO_TAG_V1 = [64]byte{
		0x5b, 0x83, 0x2d, 0xb8, 0xca, 0x26, 0xc2, 0x5b, 0xe1, 0xc5, 0x42, 0xd6, 0xcc, 0xed, 0xdd, 0xa8,
		0xc1, 0x45, 0x61, 0x5c, 0xff, 0x5c, 0x35, 0x72, 0x7f, 0xb3, 0x46, 0x26, 0x10, 0x80, 0x7e, 0x20,
		0xae, 0x53, 0x4d, 0xc3, 0xf6, 0x42, 0x99, 0x19, 0x99, 0x31, 0x77, 0x2e, 0x03, 0x78, 0x7d, 0x18,
		0x15, 0x6e, 0xb3, 0x15, 0x1e, 0x0e, 0xd1, 0xb3, 0x09, 0x8b, 0xdc, 0x84, 0x45, 0x86, 0x18, 0x85,
	}

	// UTREEXO_TAG_V1_APPEND is just UTREEXO_TAG_V1 doubled so it can be appeneded directly to a hash digest.
	UTREEXO_TAG_V1_APPEND = [128]byte{
		0x5b, 0x83, 0x2d, 0xb8, 0xca, 0x26, 0xc2, 0x5b, 0xe1, 0xc5, 0x42, 0xd6, 0xcc, 0xed, 0xdd, 0xa8,
		0xc1, 0x45, 0x61, 0x5c, 0xff, 0x5c, 0x35, 0x72, 0x7f, 0xb3, 0x46, 0x26, 0x10, 0x80, 0x7e, 0x20,
		0xae, 0x53, 0x4d, 0xc3, 0xf6, 0x42, 0x99, 0x19, 0x99, 0x31, 0x77, 0x2e, 0x03, 0x78, 0x7d, 0x18,
		0x15, 0x6e, 0xb3, 0x15, 0x1e, 0x0e, 0xd1, 0xb3, 0x09, 0x8b, 0xdc, 0x84, 0x45, 0x86, 0x18, 0x85,

		0x5b, 0x83, 0x2d, 0xb8, 0xca, 0x26, 0xc2, 0x5b, 0xe1, 0xc5, 0x42, 0xd6, 0xcc, 0xed, 0xdd, 0xa8,
		0xc1, 0x45, 0x61, 0x5c, 0xff, 0x5c, 0x35, 0x72, 0x7f, 0xb3, 0x46, 0x26, 0x10, 0x80, 0x7e, 0x20,
		0xae, 0x53, 0x4d, 0xc3, 0xf6, 0x42, 0x99, 0x19, 0x99, 0x31, 0x77, 0x2e, 0x03, 0x78, 0x7d, 0x18,
		0x15, 0x6e, 0xb3, 0x15, 0x1e, 0x0e, 0xd1, 0xb3, 0x09, 0x8b, 0xdc, 0x84, 0x45, 0x86, 0x18, 0x85,
	}
)

// ErrHashStrSize describes an error that indicates the caller specified a hash
// string that has too many characters.
var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

// Hash is used in several of the bitcoin messages and common structures.  It
// typically represents the double sha256 of data.
type Hash [HashSize]byte

// String returns the Hash as the hexadecimal string of the byte-reversed
// hash.
func (hash Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

// CloneBytes returns a copy of the bytes which represent the hash as a byte
// slice.
//
// NOTE: It is generally cheaper to just slice the hash directly thereby reusing
// the same bytes rather than calling this method.
func (hash *Hash) CloneBytes() []byte {
	newHash := make([]byte, HashSize)
	copy(newHash, hash[:])

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HashSize.
func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return fmt.Errorf("invalid hash length of %v, want %v", nhlen,
			HashSize)
	}
	copy(hash[:], newHash)

	return nil
}

// IsEqual returns true if target is the same as hash.
func (hash *Hash) IsEqual(target *Hash) bool {
	if hash == nil && target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}

// MarshalJSON serialises the hash as a JSON appropriate string value.
func (hash Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(hash.String())
}

// UnmarshalJSON parses the hash with JSON appropriate string value.
func (hash *Hash) UnmarshalJSON(input []byte) error {
	var sh string
	err := json.Unmarshal(input, &sh)
	if err != nil {
		return err
	}
	newHash, err := NewHashFromStr(sh)
	if err != nil {
		return err
	}

	return hash.SetBytes(newHash[:])
}

// NewHash returns a new Hash from a byte slice.  An error is returned if
// the number of bytes passed in is not HashSize.
func NewHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// TaggedHash implements the tagged hash scheme described in BIP-340. We use
// sha-256 to bind a message hash to a specific context using a tag:
// sha256(sha256(tag) || sha256(tag) || msg).
func TaggedHash(tag []byte, msgs ...[]byte) *Hash {
	// Check to see if we've already pre-computed the hash of the tag. If
	// so then this'll save us an extra sha256 hash.
	shaTag, ok := precomputedTags[string(tag)]
	if !ok {
		shaTag = sha256.Sum256(tag)
	}

	// h = sha256(sha256(tag) || sha256(tag) || msg)
	h := sha256.New()
	h.Write(shaTag[:])
	h.Write(shaTag[:])

	for _, msg := range msgs {
		h.Write(msg)
	}

	taggedHash := h.Sum(nil)

	// The function can't error out since the above hash is guaranteed to
	// be 32 bytes.
	hash, _ := NewHash(taggedHash)

	return hash
}

// TaggedHash512_256 implements a tagged hash scheme for utreexo leaves. We use
// sha-512_256 to bind a message hash to a specific context using a tag:
// sha512_256(sha512(tag) || sha512(tag) || leafdata).
func TaggedHash512_256(tag []byte, serialize func(io.Writer)) *Hash {
	// Check to see if we've already pre-computed the hash of the tag. If
	// so then this'll save us an extra sha512 hash.
	shaTag, ok := precomputedUtreexoTags[string(tag)]
	if !ok {
		shaTag = sha512.Sum512(tag)
	}

	// h = sha512_256(sha512(tag) || sha512(tag) || leafdata)
	h := sha512.New512_256()
	h.Write(shaTag[:])
	h.Write(shaTag[:])
	serialize(h)

	taggedHash := h.Sum(nil)
	return (*Hash)(taggedHash)
}

// NewHashFromStr creates a Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash.
func NewHashFromStr(hash string) (*Hash, error) {
	ret := new(Hash)
	err := Decode(ret, hash)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Decode decodes the byte-reversed hexadecimal string encoding of a Hash to a
// destination.
func Decode(dst *Hash, src string) error {
	// Return error if hash string is too long.
	if len(src) > MaxHashStringSize {
		return ErrHashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.  When not, pad
	// with a leading zero.
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

	// Hex decode the source bytes to a temporary destination.
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

	// Reverse copy from the temporary hash to destination.  Because the
	// temporary was zeroed, the written result will be correctly padded.
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}

// Uint64sToPackedHashes packs the passed in uint64s into the 32 byte hashes. 4 uint64s are packed into
// each 32 byte hash and if there's leftovers, it's filled with maxuint64.
func Uint64sToPackedHashes(ints []uint64) []Hash {
	// 4 uint64s fit into a 32 byte slice. For len(ints) < 4, count is 0.
	count := len(ints) / 4

	// If there's leftovers, we need to allocate 1 more.
	if len(ints)%4 != 0 {
		count++
	}

	hashes := make([]Hash, count)
	hashIdx := 0
	for i := range ints {
		// Move on to the next hash after putting in 4 uint64s into a hash.
		if i != 0 && i%4 == 0 {
			hashIdx++
		}

		// 8 is the size of a uint64.
		start := (i % 4) * 8
		binary.LittleEndian.PutUint64(hashes[hashIdx][start:start+8], ints[i])
	}

	// Pad the last hash with math.MaxUint64 if needed. We check this by seeing
	// if modulo 4 doesn't equate 0.
	if len(ints)%4 != 0 {
		// Start at the end.
		end := HashSize

		// Get the count of how many empty uint64 places we should pad.
		padAmount := 4 - len(ints)%4
		for i := 0; i < padAmount; i++ {
			// 8 is the size of a uint64.
			binary.LittleEndian.PutUint64(hashes[len(hashes)-1][end-8:end], math.MaxUint64)
			end -= 8
		}
	}

	return hashes
}

// PackedHashesToUint64 returns the uint64s in the packed hashes as a slice of uint64s.
func PackedHashesToUint64(hashes []Hash) []uint64 {
	ints := make([]uint64, 0, len(hashes)*4)
	for i := range hashes {
		// We pack 4 ints per hash.
		for j := 0; j < 4; j++ {
			// Offset for each int should be calculated by multiplying by
			// the size of a uint64.
			start := j * 8
			read := binary.LittleEndian.Uint64(hashes[i][start : start+8])

			// If we reach padded values, break.
			if read == math.MaxUint64 {
				break
			}

			// Otherwise we append the read uint64 to the slice.
			ints = append(ints, read)
		}
	}

	return ints
}
