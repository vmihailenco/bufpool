package bufpool

func (b *Buffer) reset(pos int) {
	b.ResetBuf(b.buf[:pos])
}

func (b *Buffer) ResetBuf(buf []byte) {
	b.buf = buf
	b.off = 0
	b.lastRead = opInvalid
}
