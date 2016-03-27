package mobtex

/*
#include <OpenGL/glext.h>
*/
import "C"

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"
)

var zt gl.Texture

// LoadDDSPath loads a DDS asset at path into the given gl.Context and returns
// the gl.Texture identifier for the resulting texture.
func LoadDDSPath(glctx gl.Context, path string) (gl.Texture, error) {
	f, err := asset.Open(path)
	if err != nil {
		return zt, err
	}
	defer f.Close()
	return LoadDDS(glctx, f)
}

var ddsFileCode = []byte("DDS ")

// LoadDDS loads a DDS formatted byte stream from r into the given gl.Context
// and returns the gl.Texture identifier for the resulting texture.
func LoadDDS(glctx gl.Context, r io.Reader) (gl.Texture, error) {
	r = bufio.NewReader(r)

	var (
		header      [124]byte
		fileCode    [4]byte
		height      uint32
		width       uint32
		linearSize  uint32
		mipMapCount uint32
	)

	_, err := io.ReadFull(r, fileCode[:])
	if err != nil {
		return zt, err
	}
	if !bytes.Equal(fileCode[:], ddsFileCode) {
		return zt, fmt.Errorf("not a dds format stream")
	}
	_, err = io.ReadFull(r, header[:])
	if err != nil {
		return zt, fmt.Errorf("failed to read header: %v", err)
	}

	height = binary.LittleEndian.Uint32(header[8:12])
	width = binary.LittleEndian.Uint32(header[12:16])
	linearSize = binary.LittleEndian.Uint32(header[16:20])

	mipMapCount = binary.LittleEndian.Uint32(header[24:28])
	fourCC := string(header[80:84])

	bufSize := linearSize
	if mipMapCount > 1 {
		bufSize = linearSize * 2
	}
	buf := make([]byte, bufSize)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return zt, fmt.Errorf("failed to read data (%d of %d bytes): %v", n, bufSize, err)
	}
	buf = buf[:n]
	var dummy [1]byte
	n, err = io.ReadFull(r, dummy[:])
	if err != io.EOF {
		if err == nil {
			return zt, fmt.Errorf("bytes remaining in stream")
		}
		return zt, err
	}

	var format gl.Enum
	//numComponent := 4
	blockSize := uint32(16)
	switch fourCC {
	case "DXT1":
		format = C.GL_COMPRESSED_RGBA_S3TC_DXT1_EXT
		//numComponent = 3
		blockSize = 8
	case "DXT3":
		format = C.GL_COMPRESSED_RGBA_S3TC_DXT3_EXT
	case "DXT5":
		format = C.GL_COMPRESSED_RGBA_S3TC_DXT5_EXT
	default:
		return zt, fmt.Errorf("invalid dxt identifier")
	}

	texture := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, texture)
	glctx.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	log.Printf(fourCC)
	for level := 0; level < int(mipMapCount) && (width > 0 || height > 0); level++ {
		size := ((width + 3) / 4) * ((height + 3) / 4) * blockSize
		data := buf[:size]
		log.Printf("LEVEL=%d WIDTH=%d HEIGHT=%d SIZE=%d", level, width, height, len(data))
		glctx.CompressedTexImage2D(gl.TEXTURE_2D, level, format, int(width), int(height), 0, data)
		glerr := glctx.GetError()
		if glerr == gl.INVALID_ENUM {
			return zt, fmt.Errorf("invalid internal format: %s (%x)", fourCC, format)
		} else if glerr != 0 {
			return zt, fmt.Errorf("internal gl error: %v", glerr)
		}
		buf = buf[size:]
		width /= 2
		height /= 2
	}

	glctx.Enable(gl.TEXTURE_2D)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	return texture, nil
}
