package buffer

import (
	"io"
	"strconv"
)

// Buffer provides byte buffer, which can be used for minimizing
// memory allocations.
//
// Buffer may be used with functions appending data to the given []byte
// slice. See example code for details.
//
// Use Get for obtaining an empty byte buffer.
type Buffer struct {
	// B is a byte buffer to use in append-like workloads.
	// See example code for details.
	B []byte
}

// ReadFrom implements io.ReaderFrom.
//
// The function appends all the data read from r to b.
func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	p := b.B
	nStart := int64(len(p))
	nMax := int64(cap(p))
	n := nStart
	if nMax == 0 {
		nMax = 64
		p = make([]byte, nMax)
	} else {
		p = p[:nMax]
	}
	for {
		if n == nMax {
			nMax *= 2
			bNew := make([]byte, nMax)
			copy(bNew, p)
			p = bNew
		}
		nn, err := r.Read(p[n:])
		n += int64(nn)
		if err != nil {
			b.B = p[:n]
			n -= nStart
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
	}
}

// WriteTo implements io.WriterTo.
func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.B)
	return int64(n), err
}

// Bytes returns b.B, i.e. all the bytes accumulated in the buffer.
//
// The purpose of this function is bytes.Buffer compatibility.
func (b *Buffer) Bytes() []byte {
	return b.B
}

// Write implements io.Writer - it appends p to Buffer.B
func (b *Buffer) Write(p []byte) (int, error) {
	b.B = append(b.B, p...)
	return len(p), nil
}

// WriteByte appends the byte c to the buffer.
//
// The purpose of this function is bytes.Buffer compatibility.
//
// The function always returns nil.
func (b *Buffer) WriteByte(c byte) error {
	b.B = append(b.B, c)
	return nil
}

// WriteString appends s to Buffer.B.
func (b *Buffer) WriteString(s string) (int, error) {
	b.B = append(b.B, s...)
	return len(s), nil
}

// Set sets Buffer.B to p.
func (b *Buffer) Set(p []byte) {
	b.B = append(b.B[:0], p...)
}

// SetString sets Buffer.B to s.
func (b *Buffer) SetString(s string) {
	b.B = append(b.B[:0], s...)
}

// String returns string representation of Buffer.B.
func (b *Buffer) String() string {
	return string(b.B)
}

// Reset makes Buffer.B empty.
func (b *Buffer) Reset() {
	b.B = b.B[:0]
}

func (b *Buffer) WriteInt(n int64) {
	b.B = strconv.AppendInt(b.B, n, 10)
}

func (b *Buffer) WriteUint(n uint64) {
	b.B = strconv.AppendUint(b.B, n, 10)
}

func (b *Buffer) WriteBool(v bool) {
	b.B = strconv.AppendBool(b.B, v)
}

func (b *Buffer) WriteFloat(f float64, bitSize int) {
	b.B = strconv.AppendFloat(b.B, f, 'f', -1, bitSize)
}

// Len returns the size of the byte buffer.
func (b *Buffer) Len() int {
	return len(b.B)
}

func (b *Buffer) Cap() int {
	return cap(b.B)
}

// TrimNewline trims any final "\n" byte from the end of the buffer.
func (b *Buffer) TrimNewline() {
	if i := len(b.B) - 1; i >= 0 {
		if b.B[i] == '\n' {
			b.B = b.B[:i]
		}
	}
}
