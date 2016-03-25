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
	"log"

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

	position gl.Attrib
	color    gl.Attrib
	mvp      gl.Uniform

	bufTriangleVertex gl.Buffer
	bufTriangleColor  gl.Buffer
	modelTriangle     *f32.Mat4
	mvpTriangle       [16]float32

	bufD6Vertex gl.Buffer
	bufD6Color  gl.Buffer
	modelD6     *f32.Mat4
	mvpD6       [16]float32

	mvpMat     *f32.Mat4 // mvpMap is shared because data must be serialized into mvpTriangle or mvpD6
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

	// Create a buffer for the triangle vertex positions
	bufTriangleVertex = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufTriangleVertex)
	glctx.BufferData(gl.ARRAY_BUFFER, triangleVertexData, gl.STATIC_DRAW)

	// Create a buffer for the triangle vertex positions
	bufTriangleColor = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufTriangleColor)
	glctx.BufferData(gl.ARRAY_BUFFER, triangleColorData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex positions
	bufD6Vertex = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.BufferData(gl.ARRAY_BUFFER, d6VertexData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex colors
	bufD6Color = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Color)
	glctx.BufferData(gl.ARRAY_BUFFER, d6ColorData, gl.STATIC_DRAW)

	// Initialize MVP values for the camera
	projection = new(f32.Mat4)
	view = new(f32.Mat4)
	viewEye = &f32.Vec3{0, 5, 3}
	viewCenter = &f32.Vec3{0, 0, 0}
	viewUp = &f32.Vec3{0, -1, 0}
	mvpMat = new(f32.Mat4)

	modelTriangle = new(f32.Mat4)
	modelTriangle.Identity()
	modelTriangle.Mul(modelTriangle, &f32.Mat4{
		{1, 0, 0, -1},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	})

	modelD6 = new(f32.Mat4)
	modelD6.Identity()
	modelD6.Scale(modelD6, 0.5, 0.5, 0.5)
	//modelD6.Rotate(modelD6)
	modelD6.Translate(modelD6, -1, -1, 1)

	// Initialize shader parameters
	position = glctx.GetAttribLocation(program, "vertexPosition")
	color = glctx.GetAttribLocation(program, "vertexColor")
	mvp = glctx.GetUniformLocation(program, "MVP")

	// Initialize the depth buffer to make sure faces rendering correctly according to Z
	glctx.Enable(gl.DEPTH_TEST)
	glctx.DepthFunc(gl.LESS)

	images = glutil.NewImages(glctx)
	fps = debug.NewFPS(images)
}

func onStop(glctx gl.Context) {
	glctx.DeleteProgram(program)
	glctx.DeleteBuffer(bufD6Vertex)
	glctx.DeleteBuffer(bufD6Color)
	fps.Release()
	images.Release()
}

func onPaint(glctx gl.Context, sz size.Event) {
	glctx.ClearColor(0, 0, 0, 0.4)

	// Clear the background and the depth buffer
	glctx.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	glctx.UseProgram(program)

	// Compute the current perspective and camera position.
	setPerspective(projection, 45, float32(float64(sz.WidthPx)/float64(sz.HeightPx)), 0.1, 100.0)
	lookAt(view, viewEye, viewCenter, viewUp)

	// dray the die

	mvpMat.Mul(projection, view)
	mvpMat.Mul(mvpMat, modelD6)
	serialize4(mvpD6[:], mvpMat)
	glctx.UniformMatrix4fv(mvp, mvpD6[:])

	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.EnableVertexAttribArray(position)
	glctx.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, 0, 0)

	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Color)
	glctx.EnableVertexAttribArray(color)
	glctx.VertexAttribPointer(color, coordsPerVertex, gl.FLOAT, false, 0, 0)

	glctx.DrawArrays(gl.TRIANGLES, 0, d6VertexCount)

	glctx.DisableVertexAttribArray(position)
	glctx.DisableVertexAttribArray(color)

	// draw the triangle

	mvpMat.Mul(projection, view)
	mvpMat.Mul(mvpMat, modelTriangle)
	serialize4(mvpTriangle[:], mvpMat)
	glctx.UniformMatrix4fv(mvp, mvpTriangle[:])

	glctx.BindBuffer(gl.ARRAY_BUFFER, bufTriangleVertex)
	glctx.EnableVertexAttribArray(position)
	glctx.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, 0, 0)

	glctx.BindBuffer(gl.ARRAY_BUFFER, bufTriangleColor)
	glctx.EnableVertexAttribArray(color)
	glctx.VertexAttribPointer(color, coordsPerVertex, gl.FLOAT, false, 0, 0)

	glctx.DrawArrays(gl.TRIANGLES, 0, triangleVertexCount)

	glctx.DisableVertexAttribArray(position)
	glctx.DisableVertexAttribArray(color)

	fps.Draw(sz)
}

const vertexShader = `#version 100

attribute vec3 vertexPosition;
attribute vec3 vertexColor;
uniform mat4 MVP;

varying vec3 color;

void main() {
	gl_Position = MVP * (vec4(vertexPosition + vec3(1, 0, 0), 1));
	color = vertexColor;
}`

const fragmentShader = `#version 100
precision mediump float;

varying vec3 color;

void main() {
	gl_FragColor = vec4(color, 1);
}`
