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

	"github.com/bmatsuo/mobile-gl-tutorial/f32hack"

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

	mvpColMaj          [16]float32
	mvpMat             *f32.Mat4
	model              *f32.Mat4
	modelCenter        *f32.Vec3  // this is used to center the model on its center of gravity prior to scaling
	modelScale         *f32.Vec3  // scaling applied along each axis
	modelRotationAxis  *f32.Vec3  // primary axis of model rotation -- in this application that is the y axis
	modelRotationDelta f32.Radian // amount rotation changes each frame
	modelRotationTotal f32.Radian // the current amount of rotation around the modelRotationAxis
	modelRotationMax   f32.Radian = 3.1415 / 8
	modelRotation      *f32.Mat4  // the model rotation axis, the result of rotating Total radian around Axis
	modelPos           *f32.Vec3  // spacial position of the (rotated) model.
	viewAngle          float32    // might be better for this to have higher resolution for reproduceability
	view               *f32.Mat4
	viewEye            *f32.Vec3
	viewCenter         *f32.Vec3
	viewUp             *f32.Vec3
	projection         *f32.Mat4

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
	modelCenter = &f32.Vec3{0, 1.0 / 3.0, 0}
	modelScale = &f32.Vec3{2, 0.5, 1}
	modelPos = &f32.Vec3{2, 0, 0}
	modelRotationAxis = &f32.Vec3{0, 1, 0}
	modelRotationDelta = 0.025
	modelRotation = new(f32.Mat4)
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
	glctx.ClearColor(0, 0, 0.4, 1)
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	glctx.UseProgram(program)

	viewAngle += 0.05
	*viewEye = f32.Vec3{5 * f32.Cos(viewAngle), -5 * f32.Sin(viewAngle), 3}
	*viewUp = f32.Vec3{f32.Cos(viewAngle), -f32.Sin(viewAngle), 3}

	modelRotationTotal += modelRotationDelta
	if modelRotationTotal >= modelRotationMax {
		modelRotationTotal = modelRotationMax
		modelRotationDelta *= -1
	} else if -modelRotationTotal >= modelRotationMax {
		modelRotationTotal = -modelRotationMax
		modelRotationDelta *= -1
	}

	modelRotation.Identity()
	modelRotation.Rotate(modelRotation, modelRotationTotal, modelRotationAxis)

	model.Identity()
	model.Translate(model, modelPos[0], modelPos[1], modelPos[2])
	model.Mul(model, modelRotation)
	model.Scale(model, modelScale[0], modelScale[1], modelScale[2])
	model.Translate(model, modelCenter[0], modelCenter[1], modelCenter[2])

	f32hack.SetPerspective(projection, 45, float32(float64(sz.WidthPx)/float64(sz.HeightPx)), 0.1, 100.0)
	f32hack.LookAt(view, viewEye, viewCenter, viewUp)
	mvpMat.Mul(projection, view)
	mvpMat.Mul(mvpMat, model)
	f32hack.Serialize4(mvpColMaj[:], mvpMat)

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
