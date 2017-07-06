package experiment

import (
	"testing"
)

func BenchmarkUseChannel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UseChannel(i)
	}
}

func BenchmarkUseMutex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UseMutex(i)
	}
}
