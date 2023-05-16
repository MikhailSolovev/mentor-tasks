package src

import (
	"errors"
	"io"
)

type Reader interface {
	Read(p []byte) (int, error)
	ReadAll(bufSize int) (string, error)
	BytesRead() int64
}

type CountingToLowerReaderImpl struct {
	Reader         io.Reader
	TotalBytesRead int64
}

func toLowerCase(p []byte, n int) {
	for i := 0; i < n; i++ {
		p[i] = p[i] | 32
	}
}

func (cr *CountingToLowerReaderImpl) Read(p []byte) (int, error) {
	n, err := cr.Reader.Read(p)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return 0, nil
		}
	}

	toLowerCase(p, n)

	cr.TotalBytesRead += int64(n)
	return n, err
}

func (cr *CountingToLowerReaderImpl) ReadAll(bufSize int) (string, error) {
	buffer := make([]byte, bufSize)
	_str := ""
	for {
		n, err := cr.Read(buffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return _str, err
			}
			_str += string(buffer[:n])
			break
		}
		_str += string(buffer[:n])
	}

	return _str, nil
}

func (cr *CountingToLowerReaderImpl) BytesRead() int64 {
	return cr.TotalBytesRead
}

func NewCountingReader(r io.Reader) *CountingToLowerReaderImpl {
	return &CountingToLowerReaderImpl{
		Reader: r,
	}
}
