package main

import (
	"encoding/binary"

	"golang.org/x/mobile/exp/f32"
)

var triangleVertexData = f32.Bytes(binary.LittleEndian,
	-1.0, -1.0, 0.0, // top left
	1.0, -1.0, 0.0, // bottom left
	0.0, 1.0, 0.0, // bottom right
)

var triangleColorData = f32.Bytes(binary.LittleEndian,
	1.0, 0.0, 0.0,
	0.0, 1.0, 0.0,
	0.0, 0.0, 1.0,
)

const (
	coordsPerVertex     = 3
	triangleVertexCount = 3
)
