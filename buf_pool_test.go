package bufpool_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/bufpool"
)

func TestGetZero(t *testing.T) {
	buf := bufpool.NewBuffer(nil)
	bufpool.Put(buf)

	buf = bufpool.NewBuffer(make([]byte, 0))
	bufpool.Put(buf)

	buf = bufpool.Get(0)
	require.Equal(t, buf.Len(), 0)
	require.Equal(t, buf.Cap(), 64)
}

func TestUseAfterPut(t *testing.T) {
	buf := bufpool.Get(10)
	bufpool.Put(buf)

	require.Equal(t, buf.Len(), -1)
	require.Equal(t, buf.Cap(), 64)

	require.Panics(t, func() {
		buf.WriteByte(0)
	}, "Write")
	require.Panics(t, func() {
		buf.Bytes()
	})
	require.Panics(t, func() {
		buf.String()
	})
}
