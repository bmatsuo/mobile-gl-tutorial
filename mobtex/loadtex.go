package mobtex

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mobile/gl"
)

// LoadPath loads a texture asset at the given path into glctx and
// returns a gl.Texture identifier for the resulting texture.
func LoadPath(glctx gl.Context, path string) (gl.Texture, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bmp":
		return LoadBMP(glctx, path)
	case ".ktx":
		return LoadKTX(glctx, path)
	case ".dds":
		return LoadDDSPath(glctx, path)
	default:
		return gl.Texture{}, fmt.Errorf("unable to open texture asset: %s", path)
	}
}
