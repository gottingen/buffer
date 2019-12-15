package buffer

import "sync"

const minShift = 6
const maxShift = 18
const errSlot = -1

var bbPool *byteBufferPool

func init() {
	bbPool = newByteBufferPool()
}


type bufferSlot struct {
	defaultSize int
	pool        sync.Pool
}

type byteBufferPool struct {
	minShift int
	minSize  int
	maxSize  int

	pool []*bufferSlot
}

func newBytes(size int) []byte {
	return make([]byte, size)
}

// newByteBufferPool returns byteBufferPool
func newByteBufferPool() *byteBufferPool {
	p := &byteBufferPool{
		minShift: minShift,
		minSize:  1 << minShift,
		maxSize:  1 << maxShift,
	}
	for i := 0; i <= maxShift-minShift; i++ {
		slab := &bufferSlot{
			defaultSize: 1 << (uint)(i+minShift),
		}
		p.pool = append(p.pool, slab)
	}

	return p
}


func (p *byteBufferPool) slot(size int) int {
	if size > p.maxSize {
		return errSlot
	}
	slot := 0
	shift := 0
	if size > p.minSize {
		size--
		for size > 0 {
			size = size >> 1
			shift++
		}
		slot = shift - p.minShift
	}

	return slot
}



// take returns *[]byte from byteBufferPool
func (p *byteBufferPool) take(size int) *[]byte {
	slot := p.slot(size)
	if slot == errSlot {
		b := newBytes(size)
		return &b
	}
	v := p.pool[slot].pool.Get()
	if v == nil {
		b := newBytes(p.pool[slot].defaultSize)
		b = b[0:size]
		return &b
	}
	b := v.(*[]byte)
	*b = (*b)[0:size]
	return b
}


// give returns *[]byte to byteBufferPool
func (p *byteBufferPool) give(buf *[]byte) {
	if buf == nil {
		return
	}
	size := cap(*buf)
	slot := p.slot(size)
	if slot == errSlot {
		return
	}
	if size != int(p.pool[slot].defaultSize) {
		return
	}
	p.pool[slot].pool.Put(buf)
}


// GetBytes returns *[]byte from byteBufferPool
func GetBytes(size int) *[]byte {
	return bbPool.take(size)
}

// PutBytes Put *[]byte to byteBufferPool
func PutBytes(buf *[]byte) {
	bbPool.give(buf)
}

