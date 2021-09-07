package sio

import "io"

type decWriterV10 struct {
	authDecV10
	dst io.Writer

	buffer packageV10
	offset int
}

func decryptWriterV10(dst io.Writer, config *Config) (*decWriterV10, error) {
	ad, err := newAuthDecV10(config)
	if err != nil {
		return nil, err
	}
	return &decWriterV10{
		authDecV10: ad,
		dst:        dst,
		buffer:     make(packageV10, maxPackageSize),
	}, nil
}

func (w *decWriterV10) Write(p []byte) (n int, err error) {
	if w.offset > 0 && w.offset < headerSize {
		remaining := headerSize - w.offset
		if len(p) < remaining {
			n = copy(w.buffer[w.offset:], p)
			w.offset += n
			return
		}
		n = copy(w.buffer[w.offset:], p[:remaining])
		p = p[remaining:]
		w.offset += n
	}
	if w.offset >= headerSize {
		remaining := w.buffer.Length() - w.offset
		if len(p) < remaining {
			nn := copy(w.buffer[w.offset:], p)
			w.offset += nn
			return n + nn, err
		}
		n += copy(w.buffer[w.offset:], p[:remaining])
		if err = w.Open(w.buffer.Payload(), w.buffer[:w.buffer.Length()]); err != nil {
			return n, err
		}
		if err = flush(w.dst, w.buffer.Payload()); err != nil {
			return n, err
		}
		p = p[remaining:]
		w.offset = 0
	}
	for len(p) > headerSize {
		packageLen := headerSize + tagSize + headerV10(p).Len()
		if len(p) < packageLen {
			w.offset = copy(w.buffer[:], p)
			n += w.offset
			return n, err
		}
		if err = w.Open(w.buffer[headerSize:packageLen-tagSize], p[:packageLen]); err != nil {
			return n, err
		}
		if err = flush(w.dst, w.buffer[headerSize:packageLen-tagSize]); err != nil {
			return n, err
		}
		p = p[packageLen:]
		n += packageLen
	}
	if len(p) > 0 {
		w.offset = copy(w.buffer[:], p)
		n += w.offset
	}
	return n, nil
}

func (w *decWriterV10) Close() error {
	if w.offset > 0 {
		if w.offset <= headerSize+tagSize {
			return errInvalidPayloadSize
		}
		header := headerV10(w.buffer[:headerSize])
		if w.offset < headerSize+header.Len()+tagSize {
			return errInvalidPayloadSize
		}
		if err := w.Open(w.buffer.Payload(), w.buffer[:w.buffer.Length()]); err != nil {
			return err
		}
		if err := flush(w.dst, w.buffer.Payload()); err != nil {
			return err
		}
	}
	if dst, ok := w.dst.(io.Closer); ok {
		return dst.Close()
	}
	return nil
}

type encWriterV10 struct {
	authEncV10
	dst io.Writer

	buffer      packageV10
	offset      int
	payloadSize int
}

func encryptWriterV10(dst io.Writer, config *Config) (*encWriterV10, error) {
	ae, err := newAuthEncV10(config)
	if err != nil {
		return nil, err
	}
	return &encWriterV10{
		authEncV10:  ae,
		dst:         dst,
		buffer:      make(packageV10, maxPackageSize),
		payloadSize: config.PayloadSize,
	}, nil
}

func (w *encWriterV10) Write(p []byte) (n int, err error) {
	if w.offset > 0 {
		remaining := w.payloadSize - w.offset
		if len(p) < remaining {
			n = copy(w.buffer[headerSize+w.offset:], p)
			w.offset += n
			return
		}
		n = copy(w.buffer[headerSize+w.offset:], p[:remaining])
		w.Seal(w.buffer, w.buffer[headerSize:headerSize+w.payloadSize])
		if err = flush(w.dst, w.buffer[:w.buffer.Length()]); err != nil {
			return n, err
		}
		p = p[remaining:]
		w.offset = 0
	}
	for len(p) >= w.payloadSize {
		w.Seal(w.buffer[:], p[:w.payloadSize])
		if err = flush(w.dst, w.buffer[:w.buffer.Length()]); err != nil {
			return n, err
		}
		p = p[w.payloadSize:]
		n += w.payloadSize
	}
	if len(p) > 0 {
		w.offset = copy(w.buffer[headerSize:], p)
		n += w.offset
	}
	return
}

func (w *encWriterV10) Close() error {
	if w.offset > 0 {
		w.Seal(w.buffer[:], w.buffer[headerSize:headerSize+w.offset])
		if err := flush(w.dst, w.buffer[:w.buffer.Length()]); err != nil {
			return err
		}
	}
	if dst, ok := w.dst.(io.Closer); ok {
		return dst.Close()
	}
	return nil
}

func flush(w io.Writer, p []byte) error {
	n, err := w.Write(p)
	if err != nil {
		return err
	}
	if n != len(p) {
		return io.ErrShortWrite
	}
	return nil
}
