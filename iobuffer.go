package buffer

import (
	"errors"
	"github.com/gottingen/atomic"
	"io"
	"net"
	"time"
)

const MinRead = 1 << 9
const MaxRead = 1 << 17
const ResetOffMark = -1
const DefaultSize = 1 << 4

var nullByte []byte

var (
	EOF                  = errors.New("EOF")
	ErrTooLarge          = errors.New("io buffer: too large")
	ErrNegativeCount     = errors.New("io buffer: negative count")
	ErrInvalidWriteCount = errors.New("io buffer: invalid write count")
)

// ioBuffer
type ioBuffer struct {
	buf     []byte // contents: buf[off : len(buf)]
	off     int    // read from &buf[off], write to &buf[len(buf)]
	offMark int
	count   *atomic.Int32
	eof     bool

	b *[]byte
}

func (b *ioBuffer) Read(p []byte) (n int, err error) {
	if b.off >= len(b.buf) {
		b.Reset()

		if len(p) == 0 {
			return
		}

		return 0, io.EOF
	}

	n = copy(p, b.buf[b.off:])
	b.off += n

	return
}

func (b *ioBuffer) ReadOnce(r io.Reader, duration time.Duration) (n int64, e error) {
	var (
		m               int
		zeroTime        time.Time
		conn            net.Conn
		loop, ok, first = true, true, true
	)

	if conn, ok = r.(net.Conn); !ok {
		loop = false
	}

	if b.off >= len(b.buf) {
		b.Reset()
	}

	if b.off > 0 && len(b.buf)-b.off < 4*MinRead {
		b.copy(0)
	}

	if cap(b.buf) == len(b.buf) {
		b.copy(MinRead)
	}

	for {
		if first == false {
			if free := cap(b.buf) - len(b.buf); free < MinRead {
				// not enough space at end
				if b.off+free < MinRead {
					// not enough space using beginning of buffer;
					// double buffer capacity
					b.copy(MinRead)
				} else {
					b.copy(0)
				}
			}
		}

		l := cap(b.buf) - len(b.buf)

		if conn != nil {
			if first {
				// TODO: support configure
				conn.SetReadDeadline(time.Now().Add(duration))
			} else {
				conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			}

			m, e = r.Read(b.buf[len(b.buf):cap(b.buf)])

			// Reset read deadline
			conn.SetReadDeadline(zeroTime)

		} else {
			m, e = r.Read(b.buf[len(b.buf):cap(b.buf)])
		}

		if m > 0 {
			b.buf = b.buf[0 : len(b.buf)+m]
			n += int64(m)
		}

		if e != nil {
			if te, ok := e.(net.Error); ok && te.Timeout() && !first {
				return n, nil
			}
			return n, e
		}

		if l != m {
			loop = false
		}

		if n > MaxRead {
			loop = false
		}

		if !loop {
			break
		}

		first = false
	}

	return n, nil
}

func (b *ioBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	if b.off >= len(b.buf) {
		b.Reset()
	}

	for {
		if free := cap(b.buf) - len(b.buf); free < MinRead {
			// not enough space at end
			if b.off+free < MinRead {
				// not enough space using beginning of buffer;
				// double buffer capacity
				b.copy(MinRead)
			} else {
				b.copy(0)
			}
		}

		m, e := r.Read(b.buf[len(b.buf):cap(b.buf)])

		b.buf = b.buf[0 : len(b.buf)+m]
		n += int64(m)

		if e == io.EOF {
			break
		}

		if m == 0 {
			break
		}

		if e != nil {
			return n, e
		}
	}

	return
}

func (b *ioBuffer) Write(p []byte) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(p))

	if !ok {
		m = b.grow(len(p))
	}

	return copy(b.buf[m:], p), nil
}

func (b *ioBuffer) WriteString(s string) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(s))

	if !ok {
		m = b.grow(len(s))
	}

	return copy(b.buf[m:], s), nil
}

func (b *ioBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); l+n <= cap(b.buf) {
		b.buf = b.buf[:l+n]

		return l, true
	}

	return 0, false
}

func (b *ioBuffer) grow(n int) int {
	m := b.Len()

	// If buffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Reset()
	}

	// Try to grow by means of a reslice.
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}

	if m+n <= cap(b.buf)/2 {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= cap(b.buf) to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		b.copy(0)
	} else {
		// Not enough space anywhere, we need to allocate.
		b.copy(n)
	}

	// Restore b.off and len(b.buf).
	b.off = 0
	b.buf = b.buf[:m+n]

	return m
}

func (b *ioBuffer) WriteTo(w io.Writer) (n int64, err error) {
	for b.off < len(b.buf) {
		nBytes := b.Len()
		m, e := w.Write(b.buf[b.off:])

		if m > nBytes {
			panic(ErrInvalidWriteCount)
		}

		b.off += m
		n += int64(m)

		if e != nil {
			return n, e
		}

		if m == 0 || m == nBytes {
			return n, nil
		}
	}

	return
}

func (b *ioBuffer) Append(data []byte) error {
	if b.off >= len(b.buf) {
		b.Reset()
	}

	dataLen := len(data)

	if free := cap(b.buf) - len(b.buf); free < dataLen {
		// not enough space at end
		if b.off+free < dataLen {
			// not enough space using beginning of buffer;
			// double buffer capacity
			b.copy(dataLen)
		} else {
			b.copy(0)
		}
	}

	m := copy(b.buf[len(b.buf):len(b.buf)+dataLen], data)
	b.buf = b.buf[0 : len(b.buf)+m]

	return nil
}

func (b *ioBuffer) AppendByte(data byte) error {
	return b.Append([]byte{data})
}

func (b *ioBuffer) Peek(n int) []byte {
	if len(b.buf)-b.off < n {
		return nil
	}

	return b.buf[b.off : b.off+n]
}

func (b *ioBuffer) Mark() {
	b.offMark = b.off
}

func (b *ioBuffer) Restore() {
	if b.offMark != ResetOffMark {
		b.off = b.offMark
		b.offMark = ResetOffMark
	}
}

func (b *ioBuffer) Bytes() []byte {
	return b.buf[b.off:]
}

func (b *ioBuffer) Cut(offset int) IoBuffer {
	if b.off+offset > len(b.buf) {
		return nil
	}

	buf := make([]byte, offset)

	copy(buf, b.buf[b.off:b.off+offset])
	b.off += offset
	b.offMark = ResetOffMark

	return &ioBuffer{
		buf: buf,
		off: 0,
	}
}

func (b *ioBuffer) Drain(offset int) {
	if b.off+offset > len(b.buf) {
		return
	}

	b.off += offset
	b.offMark = ResetOffMark
}

func (b *ioBuffer) String() string {
	return string(b.buf[b.off:])
}

func (b *ioBuffer) Len() int {
	return len(b.buf) - b.off
}

func (b *ioBuffer) Cap() int {
	return cap(b.buf)
}

func (b *ioBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
	b.offMark = ResetOffMark
	b.eof = false
}

func (b *ioBuffer) available() int {
	return len(b.buf) - b.off
}

func (b *ioBuffer) Clone() IoBuffer {
	buf := GetIoBuffer(b.Len())
	buf.Write(b.Bytes())

	buf.SetEOF(b.EOF())

	return buf
}

func (b *ioBuffer) Free() {
	b.Reset()
	b.giveSlice()
}

func (b *ioBuffer) Alloc(size int) {
	if b.buf != nil {
		b.Free()
	}
	if size <= 0 {
		size = DefaultSize
	}
	b.b = b.makeSlice(size)
	b.buf = *b.b
	b.buf = b.buf[:0]
}

func (b *ioBuffer) Count(count int32) int32 {
	return b.count.Add(count)
}

func (b *ioBuffer) EOF() bool {
	return b.eof
}

func (b *ioBuffer) SetEOF(eof bool) {
	b.eof = eof
}

func (b *ioBuffer) copy(expand int) {
	var newBuf []byte
	var bufp *[]byte

	if expand > 0 {
		bufp = b.makeSlice(2*cap(b.buf) + expand)
		newBuf = *bufp
		copy(newBuf, b.buf[b.off:])
		PutBytes(b.b)
		b.b = bufp
	} else {
		newBuf = b.buf
		copy(newBuf, b.buf[b.off:])
	}
	b.buf = newBuf[:len(b.buf)-b.off]
	b.off = 0
}

func (b *ioBuffer) makeSlice(n int) *[]byte {
	return GetBytes(n)
}

func (b *ioBuffer) giveSlice() {
	if b.b != nil {
		PutBytes(b.b)
		b.b = nil
		b.buf = nullByte
	}
}

func NewIoBuffer(capacity int) IoBuffer {
	buffer := &ioBuffer{
		offMark: ResetOffMark,
		count:   atomic.NewInt32(1),
	}
	if capacity <= 0 {
		capacity = DefaultSize
	}
	buffer.b = GetBytes(capacity)
	buffer.buf = (*buffer.b)[:0]
	return buffer
}

func NewIoBufferString(s string) IoBuffer {
	if s == "" {
		return NewIoBuffer(0)
	}
	return &ioBuffer{
		buf:     []byte(s),
		offMark: ResetOffMark,
		count:   atomic.NewInt32(1),
	}
}

func NewIoBufferBytes(bytes []byte) IoBuffer {
	if bytes == nil {
		return NewIoBuffer(0)
	}
	return &ioBuffer{
		buf:     bytes,
		offMark: ResetOffMark,
		count:   atomic.NewInt32(1),
	}
}

func NewIoBufferEOF() IoBuffer {
	buf := NewIoBuffer(0)
	buf.SetEOF(true)
	return buf
}
