package sio

import (
	"bytes"
	"io"
)

type decWriter struct {
	config Config
	dst    io.Writer

	firstWrite bool
}

func decryptWriter(w io.Writer, config *Config) *decWriter {
	return &decWriter{
		config:     *config,
		dst:        w,
		firstWrite: true,
	}
}

func (w *decWriter) Write(p []byte) (n int, err error) {
	if w.firstWrite {
		if len(p) == 0 {
			return 0, nil
		}
		w.firstWrite = false
		switch p[0] {
		default:
			return 0, errUnsupportedVersion
		case Version10:
			w.dst, err = decryptWriterV10(w.dst, &w.config)
			if err != nil {
				return 0, err
			}
		case Version20:
			w.dst, err = decryptWriterV20(w.dst, &w.config)
			if err != nil {
				return 0, err
			}
		}
	}
	return w.dst.Write(p)
}

func (w *decWriter) Close() error {
	if closer, ok := w.dst.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type decReader struct {
	config Config
	src    io.Reader

	firstRead bool
}

func decryptReader(r io.Reader, config *Config) *decReader {
	return &decReader{
		config:    *config,
		src:       r,
		firstRead: true,
	}
}

func (r *decReader) Read(p []byte) (n int, err error) {
	if r.firstRead {
		if len(p) == 0 {
			return 0, nil
		}
		var version [1]byte
		if _, err = io.ReadFull(r.src, version[:]); err != nil {
			return 0, err
		}
		r.firstRead = false
		r.src = io.MultiReader(bytes.NewReader(version[:]), r.src)
		switch version[0] {
		default:
			return 0, errUnsupportedVersion
		case Version10:
			r.src, err = decryptReaderV10(r.src, &r.config)
			if err != nil {
				return 0, err
			}
		case Version20:
			r.src, err = decryptReaderV20(r.src, &r.config)
			if err != nil {
				return 0, err
			}
		}
	}
	return r.src.Read(p)
}

type decReaderAt struct {
	config Config
	src    io.ReaderAt

	firstRead bool
}

func decryptReaderAt(r io.ReaderAt, config *Config) *decReaderAt {
	return &decReaderAt{
		config:    *config,
		src:       r,
		firstRead: true,
	}
}

func (r *decReaderAt) ReadAt(p []byte, offset int64) (n int, err error) {
	if r.firstRead {
		if len(p) == 0 {
			return 0, nil
		}
		var version [1]byte
		if _, err = r.src.ReadAt(version[:], 0); err != nil {
			return 0, err
		}
		r.firstRead = false
		switch version[0] {
		default:
			return 0, errUnsupportedVersion
		case Version10:
			r.src, err = decryptReaderAtV10(r.src, &r.config)
			if err != nil {
				return 0, err
			}
		case Version20:
			r.src, err = decryptReaderAtV20(r.src, &r.config)
			if err != nil {
				return 0, err
			}
		}
	}
	return r.src.ReadAt(p, offset)
}
