package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"
)

func loadBMP(glctx gl.Context, path string) (gl.Texture, error) {
	var (
		header  [54]byte
		dataPos uint32
		width   uint32
		height  uint32
		size    uint32 // width * height * 3
		data    []byte
	)

	f, err := asset.Open(path)
	if err != nil {
		return gl.Texture{}, err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	_, err = io.ReadFull(r, header[:])
	if err != nil {
		return gl.Texture{}, fmt.Errorf("failed to read header: %v", err)
	}

	if header[0] != 'B' || header[1] != 'M' {
		return gl.Texture{}, fmt.Errorf("not a BMP format file: %v", path)
	}

	dataPos = binary.LittleEndian.Uint32(header[10:14])
	width = binary.LittleEndian.Uint32(header[18:22])
	height = binary.LittleEndian.Uint32(header[22:26])
	size = binary.LittleEndian.Uint32(header[34:38])

	log.Printf("BITMAP DATA w=%d h=%d size=%d", width, height, size)

	if size == 0 {
		size = width * height * 3
	}
	if dataPos == 0 {
		dataPos = uint32(len(header))
	}

	data = make([]byte, size)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return gl.Texture{}, err
	}
	for i := 0; i < len(data); i += 3 {
		data[i], data[i+2] = data[i+2], data[i]
	}

	texture := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, texture)

	// TexImage2D does not take internalFormat or border parameters in golang.org/x/mobile/gl
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(width), int(height), gl.RGB, gl.UNSIGNED_BYTE, data)

	// Generating a mipmap for the texture provides
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	glctx.GenerateMipmap(gl.TEXTURE_2D)
	// Replace the above mipmap filtering code the following for faster texture
	// generation at the expense of good anti-aliasing.
	//glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	//glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return texture, nil
}
