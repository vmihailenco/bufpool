package bufpool_test

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"

	"github.com/vmihailenco/bufpool"
)

var syncPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func BenchmarkSyncPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := syncPool.Get().(*bytes.Buffer)
			buf.Reset()

			if err := json.NewEncoder(buf).Encode(b); err != nil {
				b.Fatal(err)
			}

			syncPool.Put(buf)
		}
	})
}

var bufPool bufpool.Pool

func BenchmarkBufPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bufPool.Get()

			if err := json.NewEncoder(buf).Encode(b); err != nil {
				b.Fatal(err)
			}

			bufPool.Put(buf)
		}
	})
}
