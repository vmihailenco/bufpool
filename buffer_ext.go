package bufpool

// Reset resets the buffer to be empty,
// but it retains the underlying storage for use by future writes.
// Reset is the same as Truncate(0).
func (b *Buffer) Reset(buf []byte) {
	if buf != nil {
		b.buf = buf
	} else {
		b.buf = b.buf[:0]
	}
	b.off = 0
	b.lastRead = opInvalid
}

func (b *Buffer) reset(pos int) {
	b.Reset(b.buf[:pos])
}
