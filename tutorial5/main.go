// +build darwin linux windows

// Note: This demo is an early preview of Go 1.5. In order to build this
// program as an Android APK using the gomobile tool.
//
// See http://godoc.org/golang.org/x/mobile/cmd/gomobile to install gomobile.
//
// Get the basic example and use gomobile to build or install it on your device.
//
//   $ go get -d golang.org/x/mobile/example/basic
//   $ gomobile build golang.org/x/mobile/example/basic # will build an APK
//
//   # plug your Android device to your computer or start an Android emulator.
//   # if you have adb installed on your machine, use gomobile install to
//   # build and deploy the APK to an Android target.
//   $ gomobile install golang.org/x/mobile/example/basic
//
// Switch to your device or emulator to start the Basic application from
// the launcher.
// You can also run the application on your desktop by running the command
// below. (Note: It currently doesn't work on Windows.)
//   $ go install golang.org/x/mobile/example/basic && basic
package main

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/bmatsuo/mobile-gl-tutorial/mobtex"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/app/debug"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

var (
	images  *glutil.Images
	fps     *debug.FPS
	program gl.Program

	position  gl.Attrib
	uv        gl.Attrib
	mvp       gl.Uniform
	textureID gl.Uniform

	texturePath string

	bufD6Vertex gl.Buffer
	bufD6UV     gl.Buffer
	textureD6   gl.Texture
	modelD6     *f32.Mat4
	mvpD6       [16]float32

	mvpMat     *f32.Mat4 // mvpMat is shared because data must be serialized into mvpD6
	view       *f32.Mat4
	viewEye    *f32.Vec3
	viewCenter *f32.Vec3
	viewUp     *f32.Vec3
	projection *f32.Mat4

	green  float32
	touchX float32
	touchY float32
)

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context
		var sz size.Event
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx, _ = e.DrawContext.(gl.Context)
					onStart(glctx)
					a.Send(paint.Event{})
				case lifecycle.CrossOff:
					onStop(glctx)
					glctx = nil
				}
			case size.Event:
				sz = e
				touchX = float32(sz.WidthPx / 2)
				touchY = float32(sz.HeightPx / 2)
			case paint.Event:
				if glctx == nil || e.External {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				onPaint(glctx, sz)
				a.Publish()
				// Drive the animation by preparing to paint the next frame
				// after this one is shown.
				a.Send(paint.Event{})
			case touch.Event:
				touchX = e.X
				touchY = e.Y
			}
		}
	})
}

func onStart(glctx gl.Context) {
	var err error
	program, err = glutil.CreateProgram(glctx, vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	textureD6, err = mobtex.LoadPath(glctx, texturePath)
	if err != nil {
		log.Printf("error loading texture: %v", err)
		return
	}

	// Create a buffer for the die vertex positions
	bufD6Vertex = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.BufferData(gl.ARRAY_BUFFER, d6VertexData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex colors
	bufD6UV = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6UV)
	glctx.BufferData(gl.ARRAY_BUFFER, d6UVData, gl.STATIC_DRAW)

	// Initialize MVP values for the camera
	projection = new(f32.Mat4)
	view = new(f32.Mat4)
	sqrt5 := float32(math.Sqrt(5.0))
	viewEye = &f32.Vec3{-sqrt5, -sqrt5, 3}
	//viewEye = &f32.Vec3{4, 3, 3}
	viewCenter = &f32.Vec3{0, 0, 0}
	viewUp = &f32.Vec3{0, 0, 1}
	//viewUp = &f32.Vec3{0, 1, 0}
	mvpMat = new(f32.Mat4)

	modelD6 = new(f32.Mat4)
	modelD6.Identity()
	//modelD6.Scale(modelD6, 0.5, 0.5, 0.5)
	//modelD6.Rotate(modelD6)
	//modelD6.Translate(modelD6, -1, -1, 1)

	// Initialize shader parameters
	position = glctx.GetAttribLocation(program, "vertexPosition")
	uv = glctx.GetAttribLocation(program, "vertexUV")
	mvp = glctx.GetUniformLocation(program, "MVP")
	textureID = glctx.GetUniformLocation(program, "myTextureSampler")

	// Initialize the depth buffer to make sure faces rendering correctly according to Z
	glctx.Enable(gl.DEPTH_TEST)
	glctx.DepthFunc(gl.LESS)

	images = glutil.NewImages(glctx)
	fps = debug.NewFPS(images)
}

func onStop(glctx gl.Context) {
	glctx.DeleteProgram(program)
	glctx.DeleteBuffer(bufD6Vertex)
	glctx.DeleteBuffer(bufD6UV)
	fps.Release()
	images.Release()
}

func onPaint(glctx gl.Context, sz size.Event) {
	glctx.ClearColor(0, 0, 0.4, 0.4)

	// Re-enable DEPTH_TEST everytime because it must be disabled for rendering
	// the FPS gauge.
	glctx.Enable(gl.DEPTH_TEST)

	// Clear the background and the depth buffer
	glctx.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	glctx.UseProgram(program)

	// Compute the current perspective and camera position.
	setPerspective(projection, 45, float32(float64(sz.WidthPx)/float64(sz.HeightPx)), 0.1, 100.0)
	lookAt(view, viewEye, viewCenter, viewUp)

	// draw the die

	mvpMat.Mul(projection, view)
	mvpMat.Mul(mvpMat, modelD6)
	serialize4(mvpD6[:], mvpMat)
	glctx.UniformMatrix4fv(mvp, mvpD6[:])

	// bind die vertex data
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.EnableVertexAttribArray(position)
	glctx.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, 0, 0)

	// bind die uv vector data
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6UV)
	glctx.EnableVertexAttribArray(uv)
	glctx.VertexAttribPointer(uv, 2, gl.FLOAT, false, 0, 0)

	// bind the die texture
	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, textureD6)
	glctx.Uniform1i(textureID, 0)

	glctx.DrawArrays(gl.TRIANGLES, 0, d6VertexCount)

	glctx.DisableVertexAttribArray(position)
	glctx.DisableVertexAttribArray(uv)

	// Disable the depth test before drawing the FPS gauge.
	glctx.Disable(gl.DEPTH_TEST)
	fps.Draw(sz)
}

var d6UVData = f32.Bytes(binary.LittleEndian,
	0.000059, 0.000004,
	0.000103, 0.336048,
	0.335973, 0.335903,
	1.000023, 0.000013,
	0.667979, 0.335851,
	0.999958, 0.336064,
	0.667979, 0.335851,
	0.336024, 0.671877,
	0.667969, 0.671889,
	1.000023, 0.000013,
	0.668104, 0.000013,
	0.667979, 0.335851,
	0.000059, 0.000004,
	0.335973, 0.335903,
	0.336098, 0.000071,
	0.667979, 0.335851,
	0.335973, 0.335903,
	0.336024, 0.671877,
	1.000004, 0.671847,
	0.999958, 0.336064,
	0.667979, 0.335851,
	0.668104, 0.000013,
	0.335973, 0.335903,
	0.667979, 0.335851,
	0.335973, 0.335903,
	0.668104, 0.000013,
	0.336098, 0.000071,
	0.000103, 0.336048,
	0.000004, 0.671870,
	0.336024, 0.671877,
	0.000103, 0.336048,
	0.336024, 0.671877,
	0.335973, 0.335903,
	0.667969, 0.671889,
	1.000004, 0.671847,
	0.667979, 0.335851,
)

const vertexShader = `#version 100

attribute vec3 vertexPosition;
attribute vec2 vertexUV;

uniform mat4 MVP;

varying vec2 UV;

void main() {
	gl_Position = MVP * (vec4(vertexPosition + vec3(1, 0, 0), 1));
	UV = vertexUV;
}`

const fragmentShader = `#version 100
precision mediump float;

uniform sampler2D myTextureSampler;

varying vec2 UV;

void main() {
	gl_FragColor = texture2D(myTextureSampler, UV);
}`
