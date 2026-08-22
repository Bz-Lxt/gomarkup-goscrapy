package bloom

import (
	"hash/fnv"
	"math"
)

// DoubleHash returns two independent 64-bit hashes for the classic
// h_i(x) = (h1 + i*h2) mod m construction (Kirsch-Mitzenmacher).
func DoubleHash(data []byte) (h1, h2 uint64) {
	f1 := fnv.New64a()
	_, _ = f1.Write(data)
	h1 = f1.Sum64()

	f2 := fnv.New64()
	_, _ = f2.Write(data)
	h2 = f2.Sum64()
	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15
	}
	return h1, h2
}

func positions(data []byte, m uint64, k int) []uint64 {
	if m == 0 || k <= 0 {
		return nil
	}
	h1, h2 := DoubleHash(data)
	out := make([]uint64, k)
	for i := 0; i < k; i++ {
		out[i] = (h1 + uint64(i)*h2) % m
	}
	return out
}

// OptimalM returns bit length for n items at false-positive rate p.
func OptimalM(n int, p float64) uint64 {
	if n <= 0 {
		n = 1
	}
	if p <= 0 || p >= 1 {
		p = 0.01
	}
	m := math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2))
	return uint64(m)
}

// OptimalK returns hash function count for m bits and n items.
func OptimalK(m uint64, n int) int {
	if n <= 0 {
		n = 1
	}
	k := int(math.Round(float64(m) / float64(n) * math.Ln2))
	if k < 1 {
		return 1
	}
	if k > 16 {
		return 16
	}
	return k
}

// DefaultM and DefaultK match Requirements §6: 1e7 URLs @ 1% ≈ 95850583 bits, k=7.
const (
	DefaultM = uint64(95850583)
	DefaultK = 7
)
