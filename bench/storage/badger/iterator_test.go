package badger

import (
	"context"
	"fmt"
	"testing"
)

func Benchmark_Badger_Simple_Iterator_100(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIteratorKeysOnly(b, ctx, vsz, 100)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator_100000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIteratorKeysOnly(b, ctx, vsz, 100000)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator_WithValues_WithPrefetch_100000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIteratorWithValues(b, ctx, vsz, 100000, true)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator_WithValues_WithoutPrefetch_100000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIteratorWithValues(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator2_WithValues_WithoutPrefetch_100000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIterator2(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator3_WithValues_WithoutPrefetch_10000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIterator3(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator4_WithValues_WithoutPrefetch_10000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIterator4(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator5_WithValues_WithoutPrefetch_10000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIterator5(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func Benchmark_Badger_Simple_Iterator6_WithValues_WithoutPrefetch_10000(b *testing.B) {
	for _, vsz := range valueSize {
		b.Run(fmt.Sprintf("ValueSize:%04d", vsz), func(b *testing.B) {
			ctx := context.Background()
			err := runBadgerIterator6(b, ctx, vsz, 100000, false)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}