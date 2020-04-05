package bufpool_test

import (
	"fmt"

	"github.com/vmihailenco/bufpool"
)

var jsonPool bufpool.Pool

func ExamplePool() {
	const avgPayloadSize = 1000

	for i := 0; i < 100000; i++ {
		buf := jsonPool.Get()

		buf.Reset()
		_, _ = buf.Write(make([]byte, avgPayloadSize))

		jsonPool.Put(buf)
	}

	buf := jsonPool.Get()

	fmt.Println("len", buf.Cap() >= 1000 && buf.Cap() <= 1200)
	// Output: len true

	jsonPool.Put(buf)
}
