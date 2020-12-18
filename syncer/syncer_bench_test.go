package syncer

import (
	"testing"
)

func Benchmark_GetColumns(b *testing.B) {
	bigInt := uint(18446744073709551615)
	model := &syncerTest{
		ID:   1,
		Name: "中文 English Tiếng Việt",
		Blob: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		BiU:  &bigInt,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getColumns(model, true)
	}
}
