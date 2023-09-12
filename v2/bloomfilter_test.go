// Package bloomfilter is face-meltingly fast, thread-safe,
// marshalable, unionable, probability- and
// optimal-size-calculating Bloom filter in go
//
// https://github.com/holiman/bloomfilter
//
// Original source:
// https://github.com/steakknife/bloomfilter
//
// Copyright © 2014, 2015, 2018 Barry Allard
// Copyright © 2020 Martin Holst Swende
//
// MIT license
//

package v2

import (
	"fmt"
	"math/rand"
	"testing"
)

// a read-only type that conforms to hash.Hash64, but only Sum64() works.
// It is set by writing the underlying value.
type hashableUint64 uint64

func (h hashableUint64) Write([]byte) (int, error) {
	panic("Unimplemented")
}

func (h hashableUint64) Sum([]byte) []byte {
	panic("Unimplemented")
}

func (h hashableUint64) Reset() {
	panic("Unimplemented")
}

func (h hashableUint64) BlockSize() int {
	panic("Unimplemented")
}

func (h hashableUint64) Size() int {
	panic("Unimplemented")
}

func (h hashableUint64) Sum64() uint64 {
	return uint64(h)
}

func hashableUint64Values() []hashableUint64 {
	return []hashableUint64{
		0,
		7,
		0x0c0ffee0,
		0xdeadbeef,
		0xffffffff,
	}
}

func hashableUint64NotValues() []hashableUint64 {
	return []hashableUint64{
		1,
		5,
		42,
		0xa5a5a5a5,
		0xfffffffe,
	}
}

func Test0(t *testing.T) {
	bf, _ := New(10000)

	t.Log("Filled ratio before adds :", bf.PreciseFilledRatio())
	for _, x := range hashableUint64Values() {
		bf.Add(x)
	}
	t.Log("Filled ratio after adds :", bf.PreciseFilledRatio())

	// these may or may not be true
	for _, y := range hashableUint64Values() {
		if bf.Contains(y) {
			t.Log("value in set querties: may contain ", y)
		} else {
			t.Fatal("value in set queries: definitely does not contain ", y,
				", but it should")
		}
	}

	// these must all be false
	for _, z := range hashableUint64NotValues() {
		if bf.Contains(z) {
			t.Log("value not in set queries: may or may not contain ", z)
		} else {
			t.Log("value not in set queries: definitely does not contain ", z,
				" which is correct")
		}
	}
}

func TestUnion(t *testing.T) {
	f1, _ := New(8 * 500)
	tmp, _ := New(8 * 500)
	if _, err := tmp.Union(f1); err == nil {
		t.Errorf("Incompatible, should error")
	}
	f2, err := f1.NewCompatible()
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(1337)
	// Add some content
	var tests = make([]hashableUint64, 200)
	for i := 0; i < len(tests); i++ {
		tests[i] = hashableUint64(rand.Uint64())
		if i&1 == 0 {
			f1.Add(tests[i])
		} else {
			f2.Add(tests[i])
		}
	}
	unionF, err := f2.Union(f1)
	if err != nil {
		t.Fatal(err)
	}
	copyF, err := unionF.Copy()
	if err != nil {
		t.Fatal(err)
	}

	for i, v := range tests {
		if !unionF.Contains(v) {
			t.Errorf("missing item %d", i)
		}
		if !copyF.Contains(v) {
			t.Errorf("missing item %d", i)
		}
		if i&1 == 0 {
			if !f1.Contains(v) {
				t.Errorf("missing item %d", i)
			}
			if f2.Contains(v) {
				t.Errorf("f2 has item it shouldn't have")
			}
		} else {
			if !f2.Contains(v) {
				t.Errorf("missing item %d", i)
			}
			if f1.Contains(v) {
				t.Errorf("f1 has item it shouldn't have")
			}
		}
	}
	// And test merging f1 into f2
	if err := f2.UnionInPlace(f1); err != nil {
		t.Fatal(err)
	}

	for i, v := range tests {
		if !f2.Contains(v) {
			t.Errorf("missing item %d", i)
		}
		if i&1 == 0 {
			if !f1.Contains(v) {
				t.Errorf("missing item %d", i)
			}
		} else {
			if f1.Contains(v) {
				t.Errorf("f1 has item it shouldn't have")
			}
		}
	}
}

func TestFPRate(t *testing.T) {
	t.Skip("because of hardcoded k=3")
	f, _ := New(8 * 32)
	f.n = 101 // "insert" 101 items
	// yes we could add some more tests here...
	have, want := f.FalsePosititveProbability(), 0.402507
	if int(1000*have) != int(1000*want) {
		t.Errorf("have %08f, want %f", have, want)
	}
}

func BenchmarkAddX10kX5(b *testing.B) {
	bf, _ := New(10000)
	b.Run("add-10kx5", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bf.Add(hashableUint64(rand.Uint32()))
		}
	})
	b.Run("add-10kx5-hash", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bf.AddHash(uint64(rand.Uint32()))
		}
	})
}

func TestAddX10kX5(t *testing.T) {
	b1, _ := New(10000)
	b2, _ := b1.NewCompatible()

	verify := func() {
		for i := 0; i < len(b1.bits); i++ {
			if b1.bits[i] != b2.bits[i] {
				t.Fatalf("error at bit %d!", i)
			}
		}
	}
	for i := 0; i < 1000000; i++ {
		v := hashableUint64(rand.Uint32())
		b1.Add(v)
		b2.AddHash(v.Sum64())
		verify()
		if !b2.Contains(v) {
			t.Fatal("contain error")
		}
	}
}
func BenchmarkContains1kX10kX5(b *testing.B) {
	bf, _ := New(10000)
	for i := 0; i < 1000; i++ {
		bf.Add(hashableUint64(rand.Uint32()))
	}
	b.Run("contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.Contains(hashableUint64(rand.Uint32()))
		}
	})
	b.Run("containsHash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.ContainsHash(uint64(rand.Uint32()))
		}
	})
}

func BenchmarkContains100kX10BX20(b *testing.B) {
	rand.Seed(1337)
	b.StopTimer()
	bf, _ := New(10 * 1000 * 1000 * 1000)
	for i := 0; i < 100*1000; i++ {
		bf.Add(hashableUint64(rand.Uint32()))
	}
	b.Run("contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.Contains(hashableUint64(rand.Uint32()))
		}
	})
	b.Run("containshash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.ContainsHash(uint64(rand.Uint32()))
		}
	})
}

func TestContains(t *testing.T) {
	rand.Seed(1337)
	bf, _ := New(10 * 1000 * 1000)
	for i := 0; i < 100*10000; i++ {
		x := hashableUint64(rand.Uint32())
		bf.Add(x)
		if !bf.Contains(x) {
			t.Fatalf("Did not contain newly added elem: %d", x.Sum64())
		}
	}
}

// BenchmarkUnionInPlace/union-8-6         	   15270	     77848 ns/op
func BenchmarkUnionInPlace(b *testing.B) {
	var filters []*Filter
	b1, _ := New(813129)
	for i := 0; i < 2000; i++ {
		b1.Add(hashableUint64(rand.Uint32()))
	}
	filters = append(filters, b1)
	for i := 0; i < 7; i++ {
		b, _ := b1.NewCompatible()
		filters = append(filters, b)
	}
	b.ResetTimer()
	b.Run("union-8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bx := range filters {
				_ = b1.UnionInPlace(bx)
			}
		}
	})
}

func BenchmarkContains94percentMisses(b *testing.B) {
	// This test should produce about
	// 5.4K hits and 94k misses
	rand.Seed(1337)
	b.StopTimer()
	bf, _ := New(10 * 1000 * 1000)
	for i := 0; i < 100*1000; i++ {
		bf.Add(hashableUint64(rand.Uint32()))
	}
	b.Run("contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.Contains(hashableUint64(rand.Uint32()))
		}
	})
	b.Run("containsHash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bf.ContainsHash(uint64(rand.Uint32()))
		}
	})
}

// This test is quite long-running, thus disabled
func TestHitrate(t *testing.T) {
	t.Skip("Long-running test, use only for sanity-checking")
	/**
	After changes:

	Fill ratio: 9.303936 %
	Theoretical hitrate : 0.007493 %
	Hit rate (100K random tests): 0.009000 % (9 out of 100000)
	Hit rate (100K random tests): 0.009000 % (9 out of 100000)
	Zero-filter Hit rate (100K random tests): 9.373000 % (9373 out of 100000)
	1-filter Hit rate: 9.474021 % (888 out of 9373)

	Original changes:

	Fill ratio: 9.303647 %
	Theoretical hitrate : 0.007492 %
	Hit rate (100K random tests): 2.658000 % (2658 out of 100000)
	Zero-filter Hit rate (100K random tests): 9.456000 % (9456 out of 100000)
	1-filter Hit rate: 53.489848 % (5058 out of 9456)

	*/
	// 512 MB bloom filter
	f, _ := New(512 * 1024 * 1024 * 8)

	// Fill it with 100M items
	for i := 0; i < 100*1024*1024; i++ {
		val := rand.Uint64()
		f.AddHash(val)
		if !f.ContainsHash(val) {
			t.Fatalf("Missing value (just inserted) %d", val)
		}
	}
	// Test individual matches
	numTests := 100000
	hits := 0

	for i := 0; i < numTests; i++ {
		h := rand.Uint64()
		if f.ContainsHash(h) {
			hits++
		}
	}
	fmt.Printf("Error rate: %f %%\n", 100*f.FalsePosititveProbability())
	// With four keys, we should obtain fillrate^4 chance of false positive
	fp := f.PreciseFilledRatio()
	fmt.Printf("Fill ratio: %02f %%\n", 100*fp)
	fmt.Printf("Theoretical hitrate : %02f %%\n", 100*fp*fp*fp*fp)
	fmt.Printf("Hit rate (100K random tests): %02f %% (%d out of %d) \n", 100*float64(hits)/float64(numTests), hits, numTests)
}
