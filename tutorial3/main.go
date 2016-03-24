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

	bufVertex gl.Buffer

	position gl.Attrib
	mvp      gl.Uniform

	mvpColMaj  [16]float32
	mvpMat     *f32.Mat4
	model      *f32.Mat4
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

	bufVertex = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufVertex)
	glctx.BufferData(gl.ARRAY_BUFFER, triangleVertexData, gl.STATIC_DRAW)

	projection = new(f32.Mat4)
	view = new(f32.Mat4)
	viewEye = &f32.Vec3{4, 3, 3}
	viewCenter = &f32.Vec3{0, 0, 0}
	viewUp = &f32.Vec3{0, 1, 0}
	model = new(f32.Mat4)
	mvpMat = new(f32.Mat4)

	position = glctx.GetAttribLocation(program, "vertexPosition")
	mvp = glctx.GetUniformLocation(program, "MVP")

	images = glutil.NewImages(glctx)
	fps = debug.NewFPS(images)
}

func onStop(glctx gl.Context) {
	glctx.DeleteProgram(program)
	glctx.DeleteBuffer(bufVertex)
	fps.Release()
	images.Release()
}

func onPaint(glctx gl.Context, sz size.Event) {
	glctx.ClearColor(0, 0, 1, 1)
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	glctx.UseProgram(program)

	projection.Perspective(45, float32(float64(sz.WidthPx)/float64(sz.HeightPx)), 0.1, 100.0)
	view.LookAt(viewEye, viewCenter, viewUp)
	model.Identity()
	mvpMat.Mul(model, view)
	mvpMat.Mul(mvpMat, projection)

	// This looks like the correct matrix serialization. Though the documents
	// claim that column major order is required I believe this is row major order.
	mvpColMaj = [16]float32{
		mvpMat[0][0],
		mvpMat[0][1],
		mvpMat[0][2],
		mvpMat[0][3],
		mvpMat[1][0],
		mvpMat[1][1],
		mvpMat[1][2],
		mvpMat[1][3],
		mvpMat[2][0],
		mvpMat[2][1],
		mvpMat[2][2],
		mvpMat[2][3],
		mvpMat[3][0],
		mvpMat[3][1],
		mvpMat[3][2],
		mvpMat[3][3],
	}
	/*
		mvpColMaj = [16]float32{
			mvpMat[0][0],
			mvpMat[1][0],
			mvpMat[2][0],
			mvpMat[3][0],
			mvpMat[0][1],
			mvpMat[1][1],
			mvpMat[2][1],
			mvpMat[3][1],
			mvpMat[0][2],
			mvpMat[1][2],
			mvpMat[2][2],
			mvpMat[3][2],
			mvpMat[0][3],
			mvpMat[1][3],
			mvpMat[2][3],
			mvpMat[3][3],
		}
	*/
	glctx.UniformMatrix4fv(mvp, mvpColMaj[:])

	glctx.BindBuffer(gl.ARRAY_BUFFER, bufVertex)
	glctx.EnableVertexAttribArray(position)
	glctx.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glctx.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	glctx.DisableVertexAttribArray(position)

	fps.Draw(sz)
}

var triangleVertexData = f32.Bytes(binary.LittleEndian,
	-1.0, -1.0, 0.0, // top left
	1.0, -1.0, 0.0, // bottom left
	0.0, 1.0, 0.0, // bottom right
)

const (
	coordsPerVertex = 3
	vertexCount     = 3
)

const vertexShader = `#version 100

attribute vec3 vertexPosition;

uniform mat4 MVP;

void main() {
	gl_Position = MVP * vec4(vertexPosition, 1);
}`

const fragmentShader = `#version 100
precision mediump float;
void main() {
	gl_FragColor = vec4(1, 0, 0, 1);
}`
