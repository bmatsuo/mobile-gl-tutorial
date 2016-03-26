//+build !linux,!android

package mobtex

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mobile/gl"
)

// LoadPath loads a texture asset at the given path into glctx and
// returns a gl.Texture identifier for the resulting texture.
//
// BUG:
// DDS format is never used because I'm not sure how to generate it or
// which platforms support it properly.
func LoadPath(glctx gl.Context, path string) (gl.Texture, error) {
	// ktx is not supported in general (at least it isn't osx). dds format is
	// untested on ios but I had a lot of trouble getting it to work on osx.  i
	// got it to the point where it didn't totaly fuck up but all the textures
	// were black for some reason.
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bmp":
		return LoadBMP(glctx, path)
	default:
		return gl.Texture{}, fmt.Errorf("unable to open texture asset: %s", path)
	}
}
