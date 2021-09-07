package ioutil

import (
	"io"
	"os"

	humanize "github.com/dustin/go-humanize"
)

const defaultAppendBufferSize = humanize.MiByte

type WriteOnCloser struct {
	io.Writer
	hasWritten bool
}

func (w *WriteOnCloser) Write(p []byte) (int, error) {
	w.hasWritten = true
	return w.Writer.Write(p)
}

func (w *WriteOnCloser) Close() error {
	if !w.hasWritten {
		_, err := w.Write(nil)
		if err != nil {
			return err
		}
	}
	if closer, ok := w.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (w *WriteOnCloser) HasWritten() bool { return w.hasWritten }

func WriteOnClose(w io.Writer) *WriteOnCloser {
	return &WriteOnCloser{w, false}
}

type LimitWriter struct {
	io.Writer
	skipBytes int64
	wLimit    int64
}

func (w *LimitWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	var n1 int
	if w.skipBytes > 0 {
		if w.skipBytes >= int64(len(p)) {
			w.skipBytes = w.skipBytes - int64(len(p))
			return n, nil
		}
		p = p[w.skipBytes:]
		w.skipBytes = 0
	}
	if w.wLimit == 0 {
		return n, nil
	}
	if w.wLimit < int64(len(p)) {
		n1, err = w.Writer.Write(p[:w.wLimit])
		w.wLimit = w.wLimit - int64(n1)
		return n, err
	}
	n1, err = w.Writer.Write(p)
	w.wLimit = w.wLimit - int64(n1)
	return n, err
}

func (w *LimitWriter) Close() error {
	if closer, ok := w.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func LimitedWriter(w io.Writer, skipBytes int64, limit int64) *LimitWriter {
	return &LimitWriter{w, skipBytes, limit}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

type SkipReader struct {
	io.Reader

	skipCount int64
}

func (s *SkipReader) Read(p []byte) (int, error) {
	l := int64(len(p))
	if l == 0 {
		return 0, nil
	}
	for s.skipCount > 0 {
		if l > s.skipCount {
			l = s.skipCount
		}
		n, err := s.Reader.Read(p[:l])
		if err != nil {
			return 0, err
		}
		s.skipCount -= int64(n)
	}
	return s.Reader.Read(p)
}

func NewSkipReader(r io.Reader, n int64) io.Reader {
	return &SkipReader{r, n}
}

func SameFile(fi1, fi2 os.FileInfo) bool {
	if !os.SameFile(fi1, fi2) {
		return false
	}
	if !fi1.ModTime().Equal(fi2.ModTime()) {
		return false
	}
	if fi1.Mode() != fi2.Mode() {
		return false
	}
	if fi1.Size() != fi2.Size() {
		return false
	}
	return true
}
