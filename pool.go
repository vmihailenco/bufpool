package bufpool

import (
	"math/bits"
	"sync/atomic"
)

const (
	minBitSize = 6 // 2**6=64 is a CPU cache line size
	steps      = 20

	minSize     = 1 << minBitSize               // 64 bytes
	maxSize     = 1 << (minBitSize + steps - 1) // 32 mb
	maxPoolSize = maxSize << 1                  // 64 mb

	defaultServePctile      = 0.9
	defaultDiscardPctile    = 0.95
	calibrateCallsThreshold = 42000
	defaultSize             = 4096
)

// Pool represents byte buffer pool.
//
// Distinct pools may be used for distinct types of byte buffers.
// Properly determined byte buffer types with their own pools may help reducing
// memory waste.
type Pool struct {
	calls       [steps]uint64
	calibrating uint64

	ServePctile   float64
	DiscardPctile float64

	serveSize   uint64
	discardSize uint64
}

func (p *Pool) getServeSize() int {
	size := atomic.LoadUint64(&p.serveSize)
	if size > 0 {
		return int(size)
	}

	// Use first size until first calibration.
	for i := 0; i < len(p.calls); i++ {
		size = atomic.LoadUint64(&p.calls[i])
		if size > 0 {
			atomic.CompareAndSwapUint64(&p.serveSize, 0, size)
			return int(size)
		}
	}

	return defaultSize
}

func (p *Pool) Get() *Buffer {
	buf := Get(p.getServeSize())
	return buf
}

func (p *Pool) Put(buf *Buffer) {
	length := buf.Len()
	if length == 0 {
		length = buf.Cap()
	}

	idx := index(length)
	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	discardSize := int(atomic.LoadUint64(&p.discardSize))
	if discardSize == 0 || buf.Cap() <= discardSize {
		Put(buf)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	var sumCall uint64
	calls := make([]uint64, len(p.calls))

	for i := 0; i < len(p.calls); i++ {
		n := atomic.SwapUint64(&p.calls[i], 0)
		calls[i] = n
		sumCall += n
	}

	serveSum := uint64(float64(sumCall) * p.getServePctile())
	discardSum := uint64(float64(sumCall) * p.getDiscardPctile())

	var serveSize int
	var discardSize int

	sumCall = 0
	for i, numCall := range calls {
		sumCall += numCall

		if serveSize == 0 && sumCall >= serveSum {
			serveSize = indexSize(i)
		} else if sumCall >= discardSum {
			discardSize = indexSize(i)
			break
		}
	}

	atomic.StoreUint64(&p.serveSize, uint64(serveSize))
	atomic.StoreUint64(&p.discardSize, uint64(discardSize))
	atomic.StoreUint64(&p.calibrating, 0)
}

func (p *Pool) getServePctile() float64 {
	if p.ServePctile > 0 {
		return p.ServePctile
	}
	return defaultServePctile
}

func (p *Pool) getDiscardPctile() float64 {
	if p.DiscardPctile > 0 {
		return p.DiscardPctile
	}
	return defaultDiscardPctile
}

func index(n int) int {
	if n == 0 {
		return 0
	}
	idx := bits.Len32(uint32((n - 1) >> minBitSize))
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}

func prevIndex(n int) int {
	next := index(n)
	if next == 0 || n == indexSize(n) {
		return next
	}
	return next - 1
}

func indexSize(idx int) int {
	return minSize << idx
}
