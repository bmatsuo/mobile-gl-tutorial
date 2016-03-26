/*
Package mobtex wraps generic texture asset decoders so they can be loaded into
a gl.Context from golang.org/x/mobile/gl.  If the texture does not supply
mipmaps, as in the BMP format, then mipmaps will be generated automatically.

Typically an application will just make use of the generic function LoadPath.

	texturePath := "myasset.ktx"
	texture, err := mobtex.LoadPath(glctx, texturePath)
	if err != nil {
		log.Printf("texture asset %s failed to load : %v", texturePath, err)
	}
*/
package mobtex
