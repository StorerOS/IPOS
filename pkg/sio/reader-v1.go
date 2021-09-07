package sio

import (
	"errors"
	"io"
	"io/ioutil"
	"sync"
)

type encReaderV10 struct {
	authEncV10
	src io.Reader

	buffer      packageV10
	offset      int
	payloadSize int
}

func encryptReaderV10(src io.Reader, config *Config) (*encReaderV10, error) {
	ae, err := newAuthEncV10(config)
	if err != nil {
		return nil, err
	}
	return &encReaderV10{
		authEncV10:  ae,
		src:         src,
		buffer:      make(packageV10, maxPackageSize),
		payloadSize: config.PayloadSize,
	}, nil
}

func (r *encReaderV10) Read(p []byte) (int, error) {
	var n int
	if r.offset > 0 {
		remaining := r.buffer.Length() - r.offset
		if len(p) < remaining {
			n = copy(p, r.buffer[r.offset:r.offset+len(p)])
			r.offset += n
			return n, nil
		}
		n = copy(p, r.buffer[r.offset:r.offset+remaining])
		p = p[remaining:]
		r.offset = 0
	}
	for len(p) >= headerSize+r.payloadSize+tagSize {
		nn, err := io.ReadFull(r.src, p[headerSize:headerSize+r.payloadSize])
		if err != nil && err != io.ErrUnexpectedEOF {
			return n, err
		}
		r.Seal(p, p[headerSize:headerSize+nn])
		n += headerSize + nn + tagSize
		p = p[headerSize+nn+tagSize:]
	}
	if len(p) > 0 {
		nn, err := io.ReadFull(r.src, r.buffer[headerSize:headerSize+r.payloadSize])
		if err != nil && err != io.ErrUnexpectedEOF {
			return n, err
		}
		r.Seal(r.buffer, r.buffer[headerSize:headerSize+nn])
		if length := r.buffer.Length(); length < len(p) {
			r.offset = copy(p, r.buffer[:length])
		} else {
			r.offset = copy(p, r.buffer[:len(p)])
		}
		n += r.offset
	}
	return n, nil
}

type decReaderV10 struct {
	authDecV10
	src io.Reader

	buffer packageV10
	offset int
}

func decryptReaderV10(src io.Reader, config *Config) (*decReaderV10, error) {
	ad, err := newAuthDecV10(config)
	if err != nil {
		return nil, err
	}
	return &decReaderV10{
		authDecV10: ad,
		src:        src,
		buffer:     make(packageV10, maxPackageSize),
	}, nil
}

func (r *decReaderV10) Read(p []byte) (n int, err error) {
	if r.offset > 0 {
		payload := r.buffer.Payload()
		remaining := len(payload) - r.offset
		if len(p) < remaining {
			n = copy(p, payload[r.offset:+r.offset+len(p)])
			r.offset += n
			return
		}
		n = copy(p, payload[r.offset:r.offset+remaining])
		p = p[remaining:]
		r.offset = 0
	}
	for len(p) >= maxPayloadSize {
		if err = r.readPackage(r.buffer); err != nil {
			return n, err
		}
		length := len(r.buffer.Payload())
		if err = r.Open(p[:length], r.buffer[:r.buffer.Length()]); err != nil {
			return n, err
		}
		p = p[length:]
		n += length
	}
	if len(p) > 0 {
		if err = r.readPackage(r.buffer); err != nil {
			return n, err
		}
		payload := r.buffer.Payload()
		if err = r.Open(payload, r.buffer[:r.buffer.Length()]); err != nil {
			return n, err
		}
		if len(payload) < len(p) {
			r.offset = copy(p, payload)
		} else {
			r.offset = copy(p, payload[:len(p)])
		}
		n += r.offset
	}
	return n, nil
}

func (r *decReaderV10) readPackage(dst packageV10) error {
	header := dst.Header()
	_, err := io.ReadFull(r.src, header)
	if err == io.ErrUnexpectedEOF {
		return errInvalidPayloadSize
	}
	if err != nil {
		return err
	}

	_, err = io.ReadFull(r.src, dst.Ciphertext())
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return errInvalidPayloadSize
	}
	if err != nil {
		return err
	}
	return nil
}

type decReaderAtV10 struct {
	src io.ReaderAt

	ad      authDecV10
	bufPool sync.Pool
}

func decryptReaderAtV10(src io.ReaderAt, config *Config) (*decReaderAtV10, error) {
	ad, err := newAuthDecV10(config)
	if err != nil {
		return nil, err
	}
	r := &decReaderAtV10{
		ad:  ad,
		src: src,
	}
	r.bufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, maxPackageSize)
			return &b
		},
	}
	return r, nil
}

func (r *decReaderAtV10) ReadAt(p []byte, offset int64) (n int, err error) {
	if offset < 0 {
		return 0, errors.New("sio: DecReaderAt.ReadAt: offset is negative")
	}

	t := offset / int64(maxPayloadSize)
	if t+1 > (1<<32)-1 {
		return 0, errUnexpectedSize
	}

	buffer := r.bufPool.Get().(*[]byte)
	defer r.bufPool.Put(buffer)
	decReader := decReaderV10{
		authDecV10: r.ad,
		src:        &sectionReader{r.src, t * maxPackageSize},
		buffer:     packageV10(*buffer),
		offset:     0,
	}
	decReader.SeqNum = uint32(t)
	if k := offset % int64(maxPayloadSize); k > 0 {
		if _, err := io.CopyN(ioutil.Discard, &decReader, k); err != nil {
			return 0, err
		}
	}

	for n < len(p) && err == nil {
		var nn int
		nn, err = (&decReader).Read(p[n:])
		n += nn
	}
	if err == io.EOF && n == len(p) {
		err = nil
	}
	return n, err
}
