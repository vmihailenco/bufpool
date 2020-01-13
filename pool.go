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

	defaultServePctile      = 0.95
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
	calls       [steps]uint32
	calibrating uint32

	ServePctile   float64 // default is 0.95
	DiscardPctile float64 // default is 0.95

	serveSize   uint32
	discardSize uint32
}

func (p *Pool) getServeSize() int {
	size := atomic.LoadUint32(&p.serveSize)
	if size > 0 {
		return int(size)
	}

	// Use first size until first calibration.
	for i := 0; i < len(p.calls); i++ {
		size = atomic.LoadUint32(&p.calls[i])
		if size > 0 {
			atomic.CompareAndSwapUint32(&p.serveSize, 0, size)
			return int(size)
		}
	}

	return defaultSize
}

func (p *Pool) Get() *Buffer {
	buf := Get(p.getServeSize())
	buf.Reset()
	return buf
}

func (p *Pool) Put(buf *Buffer) {
	length := buf.Len()
	if length == 0 {
		length = buf.Cap()
	}

	idx := index(length)
	if atomic.AddUint32(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	discardSize := int(atomic.LoadUint32(&p.discardSize))
	if discardSize == 0 || buf.Cap() <= discardSize {
		Put(buf)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint32(&p.calibrating, 0, 1) {
		return
	}

	var callSum uint64
	var calls [steps]uint32

	for i := 0; i < len(p.calls); i++ {
		n := atomic.SwapUint32(&p.calls[i], 0)
		calls[i] = n
		callSum += uint64(n)
	}

	serveSum := uint64(float64(callSum) * p.getServePctile())
	discardSum := uint64(float64(callSum) * p.getDiscardPctile())

	var serveSize int
	var discardSize int

	callSum = 0
	for i, numCall := range &calls {
		callSum += uint64(numCall)

		if serveSize == 0 && callSum >= serveSum {
			serveSize = indexSize(i)
		} else if callSum >= discardSum {
			discardSize = indexSize(i)
			break
		}
	}

	if discardSize == 0 {
		discardSize <<= 1
	}

	atomic.StoreUint32(&p.serveSize, uint32(serveSize))
	atomic.StoreUint32(&p.discardSize, uint32(discardSize))
	atomic.StoreUint32(&p.calibrating, 0)
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
	return minSize << uint(idx)
}
