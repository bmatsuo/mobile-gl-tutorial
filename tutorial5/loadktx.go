package main

import (
	"fmt"
	"log"

	"github.com/bmatsuo/mobile-gl-tutorial/texture/ktx"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/gl"
)

func loadKTX(glctx gl.Context, path string) (gl.Texture, error) {
	f, err := asset.Open(path)
	if err != nil {
		return gl.Texture{}, err
	}
	defer f.Close()

	prevUnpackAlignment := glctx.GetInteger(gl.UNPACK_ALIGNMENT)
	if prevUnpackAlignment != 4 {
		glctx.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
		defer glctx.PixelStorei(gl.UNPACK_ALIGNMENT, int32(prevUnpackAlignment))
	}

	header, metadata, data, err := ktx.Read(f)
	if err != nil {
		return gl.Texture{}, err
	}
	log.Printf("%#v", header)

	metamap, err := ktx.DecodeMetadata(header, metadata)
	if err != nil {
		return gl.Texture{}, err
	}
	for k, vs := range metamap {
		for _, v := range vs {
			log.Printf("%s=%q", k, v)
		}
	}

	log.Printf("%d levels of texture", len(data))

	texture := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, texture)

	width := int(header.PixelWidth)
	height := int(header.PixelHeight)
	for level, mipdata := range data {
		log.Printf("LEVEL=%d WIDTH=%d HEIGHT=%d LEN=%d SIZE=%d",
			level, width, height, len(mipdata), ((width+3)/4)*((height+3)/4)*8)
		glctx.CompressedTexImage2D(gl.TEXTURE_2D, level, gl.Enum(header.GLInternalFormat), width, height, 0, mipdata)
		glerr := glctx.GetError()
		if glerr == gl.INVALID_ENUM {
			formats := make([]int32, 32)
			glctx.GetIntegerv(formats, gl.COMPRESSED_TEXTURE_FORMATS)
			for i := range formats {
				if formats[i] == 0 {
					formats = formats[:i]
					break
				}
			}
			log.Printf("AVAILABLE COMPRESSED TEXTURE FORMATS: %x", formats)
			return gl.Texture{}, fmt.Errorf("invalid compressed texture format: %x", header.GLInternalFormat)
		} else if glerr != 0 {
			return gl.Texture{}, fmt.Errorf("GL ERROR: %x", glerr)
		}
		width /= 2
		height /= 2
	}

	glctx.Enable(gl.TEXTURE_2D)

	//glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	//glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	return texture, nil
}
