//+build !linux !android

package main

import "golang.org/x/mobile/gl"

const d6TexturePath = "uvtemplate.bmp"

func loadTextureD6(glctx gl.Context) (gl.Texture, error) {
	return loadBMP(glctx, d6TexturePath)
}
