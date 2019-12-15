package buffer


import (
	"sync"
	"errors"
)

var ibPool IoBufferPool

// IoBufferPool is Iobuffer Pool
type IoBufferPool struct {
	pool sync.Pool
}

// take returns IoBuffer from IoBufferPool
func (p *IoBufferPool) take(size int) (buf IoBuffer) {
	v := p.pool.Get()
	if v == nil {
		buf = NewIoBuffer(size)
	} else {
		buf = v.(IoBuffer)
		buf.Alloc(size)
		buf.Count(1)
	}
	return
}

// give returns IoBuffer to IoBufferPool
func (p *IoBufferPool) give(buf IoBuffer) {
	buf.Free()
	p.pool.Put(buf)
}

// GetIoBuffer returns IoBuffer from pool
func GetIoBuffer(size int) IoBuffer {
	return ibPool.take(size)
}

// PutIoBuffer returns IoBuffer to pool
func PutIoBuffer(buf IoBuffer) error {
	count := buf.Count(-1)
	if count > 0 {
		return nil
	} else if count < 0 {
		return errors.New("PutIoBuffer duplicate")
	}
	ibPool.give(buf)
	return nil
}

