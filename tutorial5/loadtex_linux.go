package main

import "golang.org/x/mobile/gl"

const d6TexturePath = "uvtemplate.ktx"

func loadTextureD6(glctx gl.Context) (gl.Texture, error) {
	return loadKTX(glctx, d6TexturePath)
}
