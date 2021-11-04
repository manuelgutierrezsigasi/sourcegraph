package bloomfilter

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"math"
	"unicode"
	"unicode/utf16"
)

// BloomFilterBits is the number of bits allocated for new bloom filters.
//
// This parameter, along with BloomFilterNumHashFunctions (defined below), gives us a 1 in 1.38x10^9
// false positive rate if we assume that the number of unique URIs referrable by an external package
// is of the order of 10k. See the following link for a bloom calculator: https://hur.st/bloomfilter.
const BloomFilterBits = 64 * 1024

// BloomFilterNumHashFunctions is the number of hash functions to use to determine if a value is a
// member of the filter.
const BloomFilterNumHashFunctions = 16

// encodedFilterPayload holds the state and parameters necessary to revive an encoded bloom filter.
// This includes the bitstring encoded as arrays of 32-bit integers as well as the number of hash
// functions each identifier is identified with. This is necessary to encode as increasing its value
// after its created will make all inserted identifiers un-findable.
type encodedFilterPayload struct {
	Buckets          []int32 `json:"buckets"`
	NumHashFunctions int32   `json:"numHashFunctions"`
}

// CreateFilter allocates a new bloom filter and inserts all of the given identifiers. The returned
// value is an encoded and compressed payload that can be passed to Decode to test specific values
// for membership within the identifier set.
func CreateFilter(identifiers []string) ([]byte, error) {
	buckets := make([]int32, BloomFilterBits)
	for _, identifier := range identifiers {
		addToFilter(buckets, BloomFilterNumHashFunctions, identifier)
	}

	return encodeFilter(buckets, int32(BloomFilterNumHashFunctions))
}

// Decode decodes the filter and returns a function that can be called to test if a specific value is
// probably a member of the underlying set. This method returns an error if the encoded filter cannot be
// decoded (improperly compressed or invalid JSON).
func Decode(encodedFilter []byte) (func(identifier string) bool, error) {
	r, err := gzip.NewReader(bytes.NewReader(encodedFilter))
	if err != nil {
		return nil, err
	}

	var payload encodedFilterPayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}

	buckets := payload.Buckets
	numHashFunctions := payload.NumHashFunctions

	test := func(identifier string) bool {
		return testFilter(buckets, numHashFunctions, identifier)
	}

	return test, nil
}

// encodeFilters marshalls and compresses the given bloom filter state.
func encodeFilter(buckets []int32, numHashFunctions int32) ([]byte, error) {
	payload := encodedFilterPayload{
		Buckets:          buckets,
		NumHashFunctions: numHashFunctions,
	}

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	// Encode payload through gzip writer
	if err := json.NewEncoder(gzipWriter).Encode(payload); err != nil {
		return nil, err
	}

	// Ensure gzip trailer is flushed to underlying buffer
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// The following code is a port of bloomfilter 0.0.18 from npm. We chose not to recreate all the bloom
// filters stored in Postgres because we want a transitionary period where both services (TS and Go) can
// exist and be behaviorally equivalent.
//
// There are not a large number of differences, but there are some subtle ones around overflow behavior
// and UTF-8/16 encoding. The accompanying test suite uses filters generated by the original TypeScript
// code to ensure that they can be read without a migration step. We may want to run a migration step to
// simplify this dependency, but it is in no way urgent.
//
// The original code available at https://github.com/jasondavies/bloomfilter.js.

// addToFilter sets k bits in the given buckets representing the k hash locations for the given identifier.
func addToFilter(buckets []int32, numHashFunctions int32, identifier string) {
	for _, b := range hashLocations(identifier, int32(len(buckets))*32, numHashFunctions) {
		bucketIndex, indexInBucket := index(b)
		buckets[bucketIndex] |= 1 << indexInBucket
	}
}

// testFilter determines if the k bits representing the k hash locations for the given identifier
// have been set.
func testFilter(buckets []int32, numHashFunctions int32, identifier string) bool {
	for _, b := range hashLocations(identifier, int32(len(buckets))*32, numHashFunctions) {
		bucketIndex, indexInBucket := index(b)

		// If any location is NOT set, then the identifier is guaranateed not to be a member of this
		// bloom filter. If all locations are set, it is unlikely but possible that this identifier
		// was not inserted into this bloom filter.
		if buckets[bucketIndex]&(1<<indexInBucket) == 0 {
			return false
		}
	}

	return true
}

// index returns the bucket index and the index within the bucket for a target bit.
//
// The bloom filter bucket's array represents a long bitstring of length len(buckets)*32. In
// order to set or check a single bit in this bitstring, we find the bucket that contains the
// target bit and the index within that bucket.
func index(b int32) (int32, int32) {
	return int32(math.Floor(float64(b) / 32)), b % 32
}

// Original notes:
// See http://willwhim.wpengine.com/2011/09/03/producing-n-hash-functions-by-hashing-only-once/.
func hashLocations(v string, m, k int32) []int32 {
	a := fowlerNollVo1a(v, 0)
	b := fowlerNollVo1a(v, 1576284489) // The seed value is chosen randomly
	x := a % m
	r := make([]int32, k)

	for i := int32(0); i < k; i++ {
		if x < 0 {
			r[i] = x + m
		} else {
			r[i] = x
		}
		x = (x + b) % m
	}

	return r
}

// Original notes:
// Fowler/Noll/Vo hashing. This function optionally takes a seed value that is incorporated
// into the offset basis. Almost any choice of offset basis will serve so long as it is non-zero,
// according to http://www.isthe.com/chongo/tech/comp/fnv/index.html.
func fowlerNollVo1a(v string, seed int32) int32 {
	q := 2166136261
	a := int64(int32(q) ^ seed)

	for _, r := range utf16Runes(v) {
		c := int64(r)
		if d := c & 0xff00; d != 0 {
			a = fowlerNollVoMultiply(int32(a ^ d>>8))
		}
		a = fowlerNollVoMultiply(int32(a) ^ int32(c&0xff))
	}

	return fowlerNollVoMix(int32(a))
}

// Original notes:
// Equivalent to `a * 16777619 mod 2**32`.
func fowlerNollVoMultiply(a int32) int64 {
	return int64(a) + int64(a<<1) + int64(a<<4) + int64(a<<7) + int64(a<<8) + int64(a<<24)
}

// Original notes:
// See https://web.archive.org/web/20131019013225/http://home.comcast.net/~bretm/hash/6.html.
func fowlerNollVoMix(a int32) int32 {
	a += a << 13
	a ^= int32(uint32(a) >> 7)
	a += a << 3
	a ^= int32(uint32(a) >> 17)
	a += a << 5
	return a
}

// utf16Runes converts the given string into a slice of UTF-16 encoded runes. This works by
// determining if each rune is a UTF-16 surrogate pair. If it is, we replace the rune with
// both runes composing the surrogate pair. Otherwise, we leave the original rune alone.
//
// This is a necessary step as existing filters were created in TypeScript, which treated
// strings as encoded in UTF-16, not UTF-8. We need to do this translation for runes that
// fall outside of the basic multilingual plane, or we wont be able to retrieve the original
// identifiers.
func utf16Runes(v string) []rune {
	var runes []rune
	for _, r := range v {
		// If the pair is not surrogate, U+FFFD is returned for both runes
		if a, b := utf16.EncodeRune(r); a == unicode.ReplacementChar && b == unicode.ReplacementChar {
			runes = append(runes, r)
		} else {
			runes = append(runes, a, b)
		}
	}

	return runes
}
