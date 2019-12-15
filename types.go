package buffer

import (
	"io"
	"time"
)

type IoBuffer interface {
	//Read reads next len(buf) bytes from the buffer, util the buffer is drained
	Read(p []byte) (int, error)

	ReadFrom(r io.Reader) (int64, error)

	ReadOnce(r io.Reader, duration time.Duration) (int64, error)

	Write(b []byte) (int, error)

	WriteString(s string) (n int, err error)

	WriteTo(w io.Writer) (n int64, err error)

	Peek(n int) []byte

	Bytes() []byte

	Drain(offset int)

	Alloc(int)

	Free()

	Len() int

	Cap() int

	Reset()

	Clone() IoBuffer

	String() string

	Count(int32) int32

	EOF() bool

	SetEOF(eof bool)

}

