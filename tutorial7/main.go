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
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatsuo/mobile-gl-tutorial/f32hack"
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

	// BUG:
	// The DDS compressed texture format is never used because I'm not sure how
	// to generate it or which platforms support it properly.
	texturePath string

	bufD6Vertex gl.Buffer
	bufD6UV     gl.Buffer
	textureD6   gl.Texture
	modelD6     *f32.Mat4
	mvpD6       [16]float32

	decel f32.Radian // decelleration in radians/sec

	viewAngle         f32.Radian
	viewAngleSpeed    f32.Radian
	viewAngleMaxSpeed f32.Radian

	fov         f32.Radian
	fovSpeed    f32.Radian
	fovMaxSpeed f32.Radian

	mvpMat     *f32.Mat4 // mvpMat is shared because data must be serialized into mvpD6
	view       *f32.Mat4
	viewEye    *f32.Vec3
	viewCenter *f32.Vec3
	viewUp     *f32.Vec3
	projection *f32.Mat4

	drawTime time.Time

	screen    size.Event
	touchDown bool
	touchTime time.Time
	vX        float32
	vY        float32
	deltaX    float32
	deltaY    float32
	touchX    float32
	touchY    float32
)

// PI is a low order approximation of PI
const PI = 3.14159

func computePV(sz size.Event, deltat float32, force bool) {
	if !touchDown && !force {
		return
	}
	minDim := sz.HeightPx
	if sz.WidthPx < sz.HeightPx {
		minDim = sz.WidthPx
	}
	aspect := float32(sz.WidthPx) / float32(sz.HeightPx)
	if minDim == 0 {
		minDim = 768
		aspect = float32(4) / float32(3)
	}

	// two rotations is equivalent to one screen movement in the minimum screen
	// dimention
	angleScale := f32.Radian(2 * PI / float32(minDim))
	viewAngleSpeed = f32.Radian(vX) * angleScale
	if viewAngleSpeed > viewAngleMaxSpeed {
		viewAngleSpeed = viewAngleMaxSpeed
	} else if viewAngleSpeed < -viewAngleMaxSpeed {
		viewAngleSpeed = -viewAngleMaxSpeed
	}
	viewAngle += viewAngleSpeed * f32.Radian(deltat)

	*viewEye = f32.Vec3{5 * f32.Cos(float32(viewAngle)), -5 * f32.Sin(float32(viewAngle)), 3}
	*viewUp = f32.Vec3{f32.Cos(float32(viewAngle)), -f32.Sin(float32(viewAngle)), 3}
	f32hack.LookAt(view, viewEye, viewCenter, viewUp)

	//log.Printf("ANGLE=%.02f SPEED=%.02f/s", viewAngle, viewAngleSpeed)
	newViewAngleSpeed := viewAngleSpeed
	if viewAngleSpeed > 0 {
		newViewAngleSpeed = viewAngleSpeed - decel*f32.Radian(deltat)
	} else if viewAngleSpeed < 0 {
		newViewAngleSpeed = viewAngleSpeed + decel*f32.Radian(deltat)
	}
	if viewAngleSpeed*newViewAngleSpeed < 0 {
		newViewAngleSpeed = 0
	}
	//log.Printf("NEW=%.02f/s OLD=%.02f/s", newViewAngleSpeed, viewAngleSpeed)
	viewAngleSpeed = newViewAngleSpeed
	vX = float32(viewAngleSpeed / angleScale)

	// no transformation applied to fovSpeed because honestly I don't know what
	// you would do
	fovSpeed = f32.Radian(vY)
	if fovSpeed > fovMaxSpeed {
		fovSpeed = fovMaxSpeed
	} else if fovSpeed < -fovMaxSpeed {
		fovSpeed = -fovMaxSpeed
	}
	fov += fovSpeed * f32.Radian(deltat)

	f32hack.SetPerspective(projection, fov, aspect, 0.1, 100.0)
	//log.Printf("FOV=%.02f SPEED=%.02f/s", fov, fovSpeed)
	newFovSpeed := fovSpeed
	if fovSpeed > 0 {
		newFovSpeed = fovSpeed - decel*f32.Radian(deltat)
	} else if fovSpeed < 0 {
		newFovSpeed = fovSpeed + decel*f32.Radian(deltat)
	}
	if fovSpeed*newFovSpeed < 0 {
		newFovSpeed = 0
	}
	//log.Printf("NEW=%.02f/s OLD=%.02f/s", newFovSpeed, fovSpeed)
	fovSpeed = newFovSpeed
	vY = float32(fovSpeed) // TODO: undo whatever transformation I decide is right
}

// invertUV determines if the hardcoded vertex UV matrix needs to have its y
// values inverted.
func invertUV() bool {
	return strings.ToLower(filepath.Ext(texturePath)) == ".ktx"
}

func main() {
	app.Main(func(a app.App) {
		var glctx gl.Context
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
				screen = e

				// the following is not truly correct for but handling an
				// uncommon corner case.
				touchClear()
				touchX = float32(screen.WidthPx / 2)
				touchY = float32(screen.HeightPx / 2)
			case paint.Event:
				if glctx == nil || e.External {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				onPaint(glctx, screen)
				a.Publish()
				// Drive the animation by preparing to paint the next frame
				// after this one is shown.
				a.Send(paint.Event{})
			case touch.Event:
				now := time.Now()
				switch e.Type {
				case touch.TypeBegin:
					touchDown = true
				case touch.TypeMove:
					touchDown = true
					deltaX = e.X - touchX
					deltaY = e.Y - touchY
					durus := now.Sub(touchTime) / time.Microsecond
					vX = (deltaX / float32(durus)) * 1e6
					vY = (deltaY / float32(durus)) * 1e6
				case touch.TypeEnd:
					touchClear()
				}
				touchTime = now
				touchX = e.X
				touchY = e.Y
				log.Printf("TOUCH=%t X=%.02f Y=%.02f DX=%.02f DY=%.02f VX=%.02f/s VX=%0.02g/s", touchDown, touchX, touchY, deltaX, deltaY, vX, vY)
			}
		}
	})
}

func touchClear() {
	touchDown = false
	deltaX = 0
	deltaY = 0
	vX = 0
	vY = 0
}

func onStart(glctx gl.Context) {
	// initialize touchTime just so that it isn't the zero time. it's not a big
	// deal.
	touchTime = time.Now()
	drawTime = time.Now()

	// Decellerate at 2*PI RAD/S^2 becasue that seems about right..
	decel = 2 * PI

	// allow two rotations around the model per second
	viewAngleMaxSpeed = 2 * PI

	// fov max speed is pretty arbitrary.. not even sure if it is a linear scale
	// allow one radian (degree?) of change per second.
	fov = 45
	fovMaxSpeed = 1

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

	obj, err := mobtex.DecodeObjPath("cube2.obj")
	if err != nil {
		log.Printf("error loading object: %v", err)
		return
	}
	d6VertexData = d6VertexData[:0]
	d6UVData = d6UVData[:0]
	uvinvert := invertUV()
	for i := range obj.V {
		d6VertexData = append(d6VertexData, f32.Bytes(binary.LittleEndian, obj.V[i][:]...)...)
	}
	for i := range obj.VT {
		if uvinvert {
			obj.VT[i][1] = 1 - obj.VT[i][1]
		}
		d6UVData = append(d6UVData, f32.Bytes(binary.LittleEndian, obj.VT[i][:]...)...)
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
	viewEye = &f32.Vec3{5, 0, 3}
	viewCenter = &f32.Vec3{0, 0, 0}
	viewUp = &f32.Vec3{1, 0, 0}
	mvpMat = new(f32.Mat4)

	computePV(size.Event{}, 0, true)

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
	now := time.Now()
	elapsed := now.Sub(drawTime)
	deltat := float32(float64(elapsed) / float64(time.Second))
	drawTime = now

	glctx.ClearColor(0, 0, 0.4, 0.4)

	// Re-enable flags which must be reset everytime because they must be
	// disabled for rendering the FPS gauge.
	glctx.Enable(gl.CULL_FACE)
	glctx.Enable(gl.DEPTH_TEST)

	// Clear the background and the depth buffer
	glctx.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	glctx.UseProgram(program)

	// Compute the current perspective and camera position.
	computePV(sz, deltat, false)

	// draw the die

	mvpMat.Mul(projection, view)
	mvpMat.Mul(mvpMat, modelD6)
	f32hack.Serialize4(mvpD6[:], mvpMat)
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

	// Disable certain flags before drawing the FPS gauge because they will
	// cause the gauge to be invisible.
	glctx.Disable(gl.CULL_FACE)
	glctx.Disable(gl.DEPTH_TEST)
	fps.Draw(sz)
}

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
