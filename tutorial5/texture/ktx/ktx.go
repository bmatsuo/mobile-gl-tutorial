package ktx

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var fileID = [12]byte{0xAB, 0x4B, 0x54, 0x58, 0x20, 0x31, 0x31, 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}

// headerSize is the size of an encoded ktx file header -- 12 byte identifier +
// 52 bytes of uint32 metadata
const headerSize = 64

// endiannessValue is contained in the 4 bytes following the fileID and is used
// to determine the encoding for the rest of the file.
const endiannessValue = 0x04030201

// Endianness describes the byte order of the encoded data once a ktx file has
// been read completely this value is no longer important (probably?).
type Endianness int

// Possible values for a ktx file's Endianness.
const (
	BigEndian Endianness = iota
	LittleEndian
)

// Header contains ktx file header metadata
type Header struct {
	Endianness
	GLType                uint32
	GLTypeSize            uint32
	GLFormat              uint32
	GLInternalFormat      uint32
	GLBaseInternalFormat  uint32
	PixelWidth            uint32
	PixelHeight           uint32
	PixelDepth            uint32
	NumberOfArrayElements uint32
	NumberOfFaces         uint32
	NumberOfMipmapLevels  uint32
	BytesOfKeyValueData   uint32
}

// Read reads a ktx format byte stream from r.
func Read(r io.Reader) (h *Header, meta []byte, data [][]byte, err error) {
	r = bufio.NewReader(r)
	h, err = DecodeHeader(r)
	if err != nil {
		return nil, nil, nil, err
	}
	meta = make([]byte, h.BytesOfKeyValueData)
	_, err = io.ReadFull(r, meta)
	if err != nil {
		return nil, nil, nil, err
	}
	for level := uint32(0); level < h.NumberOfMipmapLevels; level++ {
		var bufSize [4]byte
		_, err = io.ReadFull(r, bufSize[:])
		if err != nil {
			return nil, nil, nil, err
		}
		size, _ := decodeUint32(h.Endianness, bufSize[:])
		mipPad := 3 - (size+3)%4
		b := make([]byte, size+mipPad)
		_, err = io.ReadFull(r, b)
		if err != nil {
			return nil, nil, nil, err
		}
		b = b[:size]
		data = append(data, b)
	}

	var dummy [1]byte
	_, err = r.Read(dummy[:])
	if err != io.EOF {
		if err == nil {
			return nil, nil, nil, fmt.Errorf("bytes renamining in ktx stream")
		}
		return nil, nil, nil, err
	}
	return h, meta, data, nil
}

// DecodeMetadata decodes metadata key-value pairs given in a ktx file.  The
// specification is quite confused about how to handle the value and while it
// should typically be a utf-8 string it may be binary and it will include any
// terminating null included in the keyAndValueByteSize.
func DecodeMetadata(h *Header, meta []byte) (map[string][][]byte, error) {
	if len(meta) == 0 {
		return nil, nil
	}
	m := map[string][][]byte{}
	for len(meta) > 0 {
		var kvsize uint32
		kvsize, meta = decodeUint32(h.Endianness, meta)
		padsize := (3 - int(kvsize)%4)
		kvdata := meta[:int(kvsize)+padsize]
		meta = meta[int(kvsize)+padsize:]
		klen := bytes.Index(kvdata, []byte{'0'})
		if klen < 0 {
			return nil, fmt.Errorf("metadata pair missing null terminated key")
		}
		vlen := int(kvsize) - klen - 1
		if vlen < 0 {
			return nil, fmt.Errorf("invalid metadata pair")
		}
		k := string(meta[:klen])
		meta = meta[klen+1:] // skip the terminating null
		v := meta[:vlen]
		meta = meta[vlen:]
		for i := range meta {
			if meta[i] != 0 {
				return nil, fmt.Errorf("sanity check failure")
			}
		}
		m[k] = append(m[k], v)
	}
	return m, nil
}

// DecodeHeader decodes a ktx file header from r.
func DecodeHeader(r io.Reader) (*Header, error) {
	b := make([]byte, headerSize)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}

	id, b := b[:12], b[12:]
	if !bytes.Equal(id, fileID[:]) {
		return nil, fmt.Errorf("not a ktx header")
	}

	endianness, ok := byteOrder(b[:4])
	b = b[4:]
	if !ok {
		return nil, fmt.Errorf("cannot determine endianness")
	}

	h := &Header{}
	h.Endianness = endianness
	h.GLType, b = decodeUint32(h.Endianness, b)
	h.GLTypeSize, b = decodeUint32(h.Endianness, b)
	h.GLFormat, b = decodeUint32(h.Endianness, b)
	h.GLInternalFormat, b = decodeUint32(h.Endianness, b)
	h.GLBaseInternalFormat, b = decodeUint32(h.Endianness, b)
	h.PixelWidth, b = decodeUint32(h.Endianness, b)
	h.PixelHeight, b = decodeUint32(h.Endianness, b)
	h.PixelDepth, b = decodeUint32(h.Endianness, b)
	h.NumberOfArrayElements, b = decodeUint32(h.Endianness, b)
	h.NumberOfFaces, b = decodeUint32(h.Endianness, b)
	h.NumberOfMipmapLevels, b = decodeUint32(h.Endianness, b)
	h.BytesOfKeyValueData, b = decodeUint32(h.Endianness, b)
	if len(b) != 0 {
		panic(fmt.Sprintf("leftover bytes %q", b))
	}
	return h, nil
}

func decodeUint32(end Endianness, b []byte) (uint32, []byte) {
	if end == BigEndian {
		return binary.BigEndian.Uint32(b[:4]), b[4:]
	}
	return binary.LittleEndian.Uint32(b[:4]), b[4:]
}

func byteOrder(endianness []byte) (end Endianness, ok bool) {
	be := binary.BigEndian.Uint32(endianness)
	if be == endiannessValue {
		return BigEndian, true
	}
	le := binary.LittleEndian.Uint32(endianness)
	if le == endiannessValue {
		return LittleEndian, true
	}
	return 0, false
}
