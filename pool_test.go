package bufpool

import (
	"math/rand"
	"testing"
	"time"
)

func TestIndexSize(t *testing.T) {
	tests := []struct {
		idx  int
		size int
	}{
		{0, 64},
		{1, 128},
		{2, 256},
		{steps - 1, maxSize},
	}

	for _, test := range tests {
		got := indexSize(test.idx)
		if got != test.size {
			t.Fatalf("got %d, wanted %d", got, test.size)
		}
	}
}

func TestIndex(t *testing.T) {
	testIndex(t, 0, 0)
	testIndex(t, 1, 0)

	testIndex(t, minSize-1, 0)
	testIndex(t, minSize, 0)
	testIndex(t, minSize+1, 1)

	testIndex(t, 2*minSize-1, 1)
	testIndex(t, 2*minSize, 1)
	testIndex(t, 2*minSize+1, 2)

	testIndex(t, maxSize-1, steps-1)
	testIndex(t, maxSize, steps-1)
	testIndex(t, maxSize+1, steps-1)
}

func testIndex(t *testing.T, n, expectedIdx int) {
	idx := index(n)
	if idx != expectedIdx {
		t.Fatalf("unexpected idx for n=%d: %d. Expecting %d", n, idx, expectedIdx)
	}
}

func TestPoolCalibrate(t *testing.T) {
	var p Pool
	for i := 0; i < steps*calibrateCallsThreshold; i++ {
		n := 1004
		if i%15 == 0 {
			n = rand.Intn(15234)
		}
		testGetPut(t, &p, n)
	}
}

func TestPoolVariousSizesSerial(t *testing.T) {
	testPoolVariousSizes(t)
}

func TestPoolVariousSizesConcurrent(t *testing.T) {
	concurrency := 5
	ch := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func() {
			testPoolVariousSizes(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testPoolVariousSizes(t *testing.T) {
	var p Pool
	for i := 0; i <= steps; i++ {
		n := (1 << uint32(i))

		testGetPut(t, &p, n)
		testGetPut(t, &p, n+1)
		testGetPut(t, &p, n-1)

		for j := 0; j < 10; j++ {
			testGetPut(t, &p, j+n)
		}
	}
}

func testGetPut(t *testing.T, p *Pool, n int) {
	buf := p.Get()

	bb := buf.Bytes()
	bb = allocNBytes(bb, n)

	buf.ResetBuf(bb)
	p.Put(buf)
}

func allocNBytes(dst []byte, n int) []byte {
	diff := n - cap(dst)
	if diff <= 0 {
		return dst[:n]
	}
	return append(dst, make([]byte, diff)...)
}
