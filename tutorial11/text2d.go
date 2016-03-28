package main

import (
	"encoding/binary"
	"math"

	"github.com/bmatsuo/mobile-gl-tutorial/mobtex"

	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

type text2D struct {
	texturePath string

	v        []Vec2
	vb       []byte
	uv       []Vec2
	uvb      []byte
	vBuffer  gl.Buffer
	uvBuffer gl.Buffer

	gl        gl.Context
	program   gl.Program
	texture   gl.Texture
	vertexPos gl.Attrib
	vertexUV  gl.Attrib
	sampler   gl.Uniform
}

func newText2D(glctx gl.Context, texturePath string) (*text2D, error) {
	t2d := new(text2D)
	err := t2d.init(glctx, texturePath)
	if err != nil {
		return nil, err
	}
	return t2d, nil
}

// init reads a font texture (ASCII printable minimum).
func (t2d *text2D) init(glctx gl.Context, texturePath string) error {
	t2d.gl = glctx

	program, err := glutil.CreateProgram(t2d.gl, textVertexShader, textFragmentShader)
	if err != nil {
		return err
	}
	t2d.program = program

	t2d.texturePath = texturePath
	texture, err := mobtex.LoadPath(t2d.gl, t2d.texturePath)
	if err != nil {
		t2d.cleanup()
		return err
	}
	t2d.texture = texture

	t2d.vBuffer = t2d.gl.CreateBuffer()
	t2d.uvBuffer = t2d.gl.CreateBuffer()

	t2d.vertexPos = t2d.gl.GetAttribLocation(program, "vertexPosition")
	t2d.vertexUV = t2d.gl.GetAttribLocation(program, "vertexUV")
	t2d.sampler = t2d.gl.GetUniformLocation(program, "myTextureSampler")

	return nil
}

// write draws text at the specified (artificial) coordinates.
func (t2d *text2D) write(text string, x, y, size int) {
	// reduce allocations by reusing old vector buffers
	v := t2d.v[:0]
	uv := t2d.uv[:0]

	const uvOffset = 1.0 / 16.0

	for i, char := range text {
		qUpLeft := Vec2{float32(x + i*size), float32(y + size)}
		qUpRight := Vec2{float32(x + i*size + size), float32(y + size)}
		qDownRight := Vec2{float32(x + i*size + size), float32(y)}
		qDownLeft := Vec2{float32(x + i*size), float32(y)}

		v = append(v, qUpLeft)
		v = append(v, qDownLeft)
		v = append(v, qUpRight)

		v = append(v, qDownRight)
		v = append(v, qUpRight)
		v = append(v, qDownLeft)

		uvX := float32(char%16) / 16.0
		uvY := float32(char/16) / 16.0

		qUpLeft = Vec2{float32(uvX), float32(uvY)}
		qUpRight = Vec2{float32(uvX + uvOffset), float32(uvY)}
		qDownRight = Vec2{float32(uvX + uvOffset), float32(uvY + uvOffset)}
		qDownLeft = Vec2{float32(uvX), float32(uvY + uvOffset)}

		uv = append(uv, qUpLeft)
		uv = append(uv, qDownLeft)
		uv = append(uv, qUpRight)

		uv = append(uv, qDownRight)
		uv = append(uv, qUpRight)
		uv = append(uv, qDownLeft)
	}

	// store v and uv for future calls
	t2d.v = v
	t2d.uv = uv

	// fill and bind buffers
	t2d.vb = appendVec2(t2d.vb[:0], binary.LittleEndian, v...)
	t2d.uvb = appendVec2(t2d.uvb[:0], binary.LittleEndian, uv...)
	t2d.gl.BindBuffer(gl.ARRAY_BUFFER, t2d.vBuffer)
	t2d.gl.BufferData(gl.ARRAY_BUFFER, t2d.vb, gl.STATIC_DRAW)
	t2d.gl.BindBuffer(gl.ARRAY_BUFFER, t2d.uvBuffer)
	t2d.gl.BufferData(gl.ARRAY_BUFFER, t2d.uvb, gl.STATIC_DRAW)

	t2d.gl.UseProgram(t2d.program)

	// use TEXTURE0 for the fragment shader texture sampler
	t2d.gl.ActiveTexture(gl.TEXTURE0)
	t2d.gl.BindTexture(gl.TEXTURE_2D, t2d.texture)
	t2d.gl.Uniform1i(t2d.sampler, 0)

	// setup vertex position and uv attributes
	t2d.gl.EnableVertexAttribArray(t2d.vertexPos)
	t2d.gl.BindBuffer(gl.ARRAY_BUFFER, t2d.vBuffer)
	t2d.gl.VertexAttribPointer(t2d.vertexPos, 2, gl.FLOAT, false, 0, 0)
	defer t2d.gl.DisableVertexAttribArray(t2d.vertexPos)
	t2d.gl.EnableVertexAttribArray(t2d.vertexUV)
	t2d.gl.BindBuffer(gl.ARRAY_BUFFER, t2d.uvBuffer)
	t2d.gl.VertexAttribPointer(t2d.vertexUV, 2, gl.FLOAT, false, 0, 0)
	defer t2d.gl.DisableVertexAttribArray(t2d.vertexUV)

	// temporarily enable blending to handle transparencies.
	t2d.gl.Enable(gl.BLEND)
	t2d.gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	defer t2d.gl.Disable(gl.BLEND)

	// draw the text
	t2d.gl.DrawArrays(gl.TRIANGLES, 0, len(t2d.v))
}

// clear internal structures so they may be garbage collected.  init can
// re-initialize a text2D object.
func (t2d *text2D) cleanup() {
	if t2d.gl == nil {
		return
	}
	defer func() {
		*t2d = text2D{}
	}()

	if t2d.uvBuffer.Value != 0 {
		t2d.gl.DeleteBuffer(t2d.uvBuffer)
		t2d.uvBuffer = gl.Buffer{}
	}
	if t2d.vBuffer.Value != 0 {
		t2d.gl.DeleteBuffer(t2d.vBuffer)
		t2d.vBuffer = gl.Buffer{}
	}

	if t2d.texture.Value != 0 {
		t2d.gl.DeleteTexture(t2d.texture)
		t2d.texture = gl.Texture{}
	}

	if t2d.program.Value != 0 {
		t2d.gl.DeleteProgram(t2d.program)
		t2d.program = gl.Program{}
	}
}

func appendVec2(b []byte, bo binary.ByteOrder, v ...Vec2) []byte {
	for i := range v {
		var buf [4]byte
		bo.PutUint32(buf[:], math.Float32bits(v[i][0]))
		b = append(b, buf[:]...)
		bo.PutUint32(buf[:], math.Float32bits(v[i][1]))
		b = append(b, buf[:]...)
	}
	return b
}

// Vec2 is a two-dimensional vector of real numbers with 32-bits of precision
// in each dimension.
type Vec2 [2]float32

var textVertexShader = `#version 100

attribute vec2 vertexPosition;
attribute vec2 vertexUV;

varying vec2 UV;

void main(){

	// Output position of the vertex, in clip space
	// map [0..800][0..600] to [-1..1][-1..1]
	vec2 vertexPosition_homoneneousspace = vertexPosition - vec2(400,300); // [0..800][0..600] -> [-400..400][-300..300]
	vertexPosition_homoneneousspace /= vec2(400,300);
	gl_Position =  vec4(vertexPosition_homoneneousspace, 0, 1);

	UV = vertexUV;
}`

var textFragmentShader = `#version 100
precision mediump float;

varying vec2 UV;

uniform sampler2D myTextureSampler;

void main(){
	// just sample the color from the texture. make sure to preserve
	// transparency.
	gl_FragColor = texture2D( myTextureSampler, UV );
}`
