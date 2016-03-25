package main

import (
	"encoding/binary"
	colour "image/color"

	"golang.org/x/mobile/exp/f32"
)

func init() {
	computeD6ColorData()
}

const (
	d6VertexCount = 3 * 2 * 6
)

var d6VertexData = f32.Bytes(binary.LittleEndian,
	-1.0, -1.0, -1.0, // triangle 1 : begin
	-1.0, -1.0, 1.0,
	-1.0, 1.0, 1.0, // triangle 1 : end
	1.0, 1.0, -1.0, // triangle 2 : begin
	-1.0, -1.0, -1.0,
	-1.0, 1.0, -1.0, // triangle 2 : end
	1.0, -1.0, 1.0,
	-1.0, -1.0, -1.0,
	1.0, -1.0, -1.0,
	1.0, 1.0, -1.0,
	1.0, -1.0, -1.0,
	-1.0, -1.0, -1.0,
	-1.0, -1.0, -1.0,
	-1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0,
	1.0, -1.0, 1.0,
	-1.0, -1.0, 1.0,
	-1.0, -1.0, -1.0,
	-1.0, 1.0, 1.0,
	-1.0, -1.0, 1.0,
	1.0, -1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, -1.0, -1.0,
	1.0, 1.0, -1.0,
	1.0, -1.0, -1.0,
	1.0, 1.0, 1.0,
	1.0, -1.0, 1.0,
	1.0, 1.0, 1.0,
	1.0, 1.0, -1.0,
	-1.0, 1.0, -1.0,
	1.0, 1.0, 1.0,
	-1.0, 1.0, -1.0,
	-1.0, 1.0, 1.0,
	1.0, 1.0, 1.0,
	-1.0, 1.0, 1.0,
	1.0, -1.0, 1.0,
)

var d6VertexColors = [d6VertexCount]colour.Color{
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 255, G: 0, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 255, B: 0},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 0, G: 0, B: 255},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 127, G: 0, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 127, B: 0},
	colour.RGBA{R: 0, G: 0, B: 127},
	colour.RGBA{R: 0, G: 0, B: 127},
	colour.RGBA{R: 0, G: 0, B: 127},
	colour.RGBA{R: 0, G: 0, B: 127},
	colour.RGBA{R: 0, G: 0, B: 127},
	colour.RGBA{R: 0, G: 0, B: 127},
}

var d6VertexColorP [3 * d6VertexCount]float32

var d6ColorData = f32.Bytes(binary.LittleEndian, d6VertexColorP[:]...)

func computeD6ColorP() {
	for i := range d6VertexColors {
		r, g, b, _ := d6VertexColors[i].RGBA()
		d6VertexColorP[3*i+0] = float32(r) / float32(uint16(0xffff))
		d6VertexColorP[3*i+1] = float32(g) / float32(uint16(0xffff))
		d6VertexColorP[3*i+2] = float32(b) / float32(uint16(0xffff))
	}
}

func computeD6ColorData() {
	computeD6ColorP()
	copy(d6ColorData, f32.Bytes(binary.LittleEndian, d6VertexColorP[:]...))
}
