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
	"image/color"
	"log"
	"math"
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

	glPosition   gl.Attrib
	glUV         gl.Attrib
	glNorm       gl.Attrib
	glMVP        gl.Uniform
	glM          gl.Uniform
	glV          gl.Uniform
	glTexture    gl.Uniform
	glLightPos   gl.Uniform
	glLightPosMP gl.Uniform
	glLightColor gl.Uniform
	glLightPower gl.Uniform

	// BUG:
	// The DDS compressed texture format is never used because I'm not sure how
	// to generate it or which platforms support it properly.
	texturePath string

	bufD6Vertex gl.Buffer
	bufD6UV     gl.Buffer
	bufD6Norm   gl.Buffer
	bufD6Index  gl.Buffer
	textureD6   gl.Texture
	modelD6     *f32.Mat4
	mvpD6       [16]float32
	mD6         [16]float32
	_view       [16]float32

	decel f32.Radian // decelleration in radians/sec

	viewAngle         f32.Radian
	viewAngleSpeed    f32.Radian
	viewAngleMaxSpeed f32.Radian

	fov         f32.Radian
	fovSpeed    f32.Radian
	fovMaxSpeed f32.Radian
	fovMin      f32.Radian
	fovMax      f32.Radian

	lightPos   *f32.Vec3
	lightColor color.RGBA
	lightPower float32

	mvpMat     *f32.Mat4 // mvpMat is shared because data must be serialized into mvpD6
	view       *f32.Mat4
	viewEye    *f32.Vec3
	viewCenter *f32.Vec3
	viewUp     *f32.Vec3
	projection *f32.Mat4

	drawTime time.Time
	numDraw  uint64
	fpsTime  time.Time

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
	viewAngleOld := viewAngle
	viewAngle += viewAngleSpeed * f32.Radian(deltat)
	if math.IsNaN(float64(viewAngle)) {
		// TODO:
		// figure out where these NaNs are coming from and stop avoiding them
		// like a sloppy programmer.
		viewAngle = viewAngleOld
	}

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
	fovOld := fov
	fov += fovSpeed * f32.Radian(deltat)
	if math.IsNaN(float64(fov)) {
		fov = fovOld
	} else if fov > fovMax {
		fov = fovMax
	} else if fov < fovMin {
		fov = fovMin
	}

	f32hack.SetPerspective(projection, fov, aspect, 0.1, 100.0)
	newFovSpeed := fovSpeed
	if fovSpeed > 0 {
		newFovSpeed = fovSpeed - decel*f32.Radian(deltat)
	} else if fovSpeed < 0 {
		newFovSpeed = fovSpeed + decel*f32.Radian(deltat)
	}
	if fovSpeed*newFovSpeed <= 0 {
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
	now := time.Now()
	touchTime = now
	drawTime = now
	fpsTime = now

	// Decellerate at 2*PI RAD/S^2 becasue that seems about right..
	decel = 2 * PI

	// allow two rotations around the model per second
	viewAngleMaxSpeed = 2 * PI

	// fov max speed is pretty arbitrary.. not even sure if it is a linear scale
	// allow one radian (degree?) of change per second.
	fov = PI / 4.0
	fovMaxSpeed = 2
	fovMin = PI / 18.0
	fovMax = PI * 5 / 6

	lightPos = &f32.Vec3{5, 5, 5}
	lightColor = color.RGBA{R: 255, G: 255, B: 255}
	lightPower = 50.0

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
	vbo := mobtex.IndexVBO(obj)
	d6VertexData = d6VertexData[:0]
	d6UVData = d6UVData[:0]
	d6NormData = d6NormData[:0]
	d6IndexData = d6IndexData[:0]
	uvinvert := invertUV()
	for i := range vbo.V {
		d6VertexData = append(d6VertexData, f32.Bytes(binary.LittleEndian, vbo.V[i][:]...)...)
	}
	for i := range vbo.VT {
		if uvinvert {
			vbo.VT[i][1] = 1 - vbo.VT[i][1]
		}
		d6UVData = append(d6UVData, f32.Bytes(binary.LittleEndian, vbo.VT[i][:]...)...)
	}
	for i := range vbo.VN {
		d6NormData = append(d6NormData, f32.Bytes(binary.LittleEndian, vbo.VN[i][:]...)...)
	}
	for i := range vbo.Index {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], vbo.Index[i])
		d6IndexData = append(d6IndexData, buf[:]...)
	}

	// Create a buffer for the die vertex positions
	bufD6Vertex = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.BufferData(gl.ARRAY_BUFFER, d6VertexData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex UV vectors
	bufD6UV = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6UV)
	glctx.BufferData(gl.ARRAY_BUFFER, d6UVData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex normals
	bufD6Norm = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Norm)
	glctx.BufferData(gl.ARRAY_BUFFER, d6NormData, gl.STATIC_DRAW)

	// Create a buffer for the die vertex index
	bufD6Index = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, bufD6Index)
	glctx.BufferData(gl.ELEMENT_ARRAY_BUFFER, d6IndexData, gl.STATIC_DRAW)

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

	// Initialize shader parameters
	glPosition = glctx.GetAttribLocation(program, "vertexPosition")
	glUV = glctx.GetAttribLocation(program, "vertexUV")
	glNorm = glctx.GetAttribLocation(program, "vertexNormal")
	glMVP = glctx.GetUniformLocation(program, "MVP")
	glM = glctx.GetUniformLocation(program, "M")
	glV = glctx.GetUniformLocation(program, "V")
	glTexture = glctx.GetUniformLocation(program, "myTextureSampler")
	glLightPos = glctx.GetUniformLocation(program, "lightPosition")
	glLightPosMP = glctx.GetUniformLocation(program, "lightPosition_mp")
	glLightColor = glctx.GetUniformLocation(program, "lightColor")
	glLightPower = glctx.GetUniformLocation(program, "lightPower")

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
	numDraw++

	now := time.Now()
	elapsed := now.Sub(drawTime)
	deltat := float32(float64(elapsed) / float64(time.Second))
	drawTime = now
	if now.Sub(fpsTime) > time.Second {
		log.Printf("TOUCH=%t X=%.02f Y=%.02f DX=%.02f DY=%.02f VX=%.02f/s VX=%0.02g/s", touchDown, touchX, touchY, deltaX, deltaY, vX, vY)
		log.Printf("ANGLE=%.03f VIEW=\n%v", viewAngle, view)
		log.Printf("FOV=%.03f PROJETION=\n%v", fov, projection)
		log.Printf("LATENCY=%.03f ms/frame", 1000/float64(numDraw))
		numDraw = 0
		fpsTime = now
	}

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
	f32hack.Serialize4(mD6[:], modelD6)
	f32hack.Serialize4(_view[:], view)
	glctx.UniformMatrix4fv(glMVP, mvpD6[:])
	glctx.UniformMatrix4fv(glM, mD6[:])
	glctx.UniformMatrix4fv(glV, _view[:])
	glctx.Uniform3f(glLightPos, lightPos[0], lightPos[1], lightPos[2])
	glctx.Uniform3f(glLightPosMP, lightPos[0], lightPos[1], lightPos[2])
	rlight := float32(lightColor.R) / float32(255)
	glight := float32(lightColor.G) / float32(255)
	blight := float32(lightColor.B) / float32(255)
	glctx.Uniform3f(glLightColor, rlight, glight, blight)
	glctx.Uniform1f(glLightPower, lightPower)

	// bind die vertex data
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Vertex)
	glctx.EnableVertexAttribArray(glPosition)
	glctx.VertexAttribPointer(glPosition, coordsPerVertex, gl.FLOAT, false, 0, 0)

	// bind die uv vector data
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6UV)
	glctx.EnableVertexAttribArray(glUV)
	glctx.VertexAttribPointer(glUV, 2, gl.FLOAT, false, 0, 0)

	// bind die normal vector data
	glctx.BindBuffer(gl.ARRAY_BUFFER, bufD6Norm)
	glctx.EnableVertexAttribArray(glNorm)
	glctx.VertexAttribPointer(glNorm, 3, gl.FLOAT, false, 0, 0)

	// bind die vector index data
	glctx.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, bufD6Index)

	// bind the die texture
	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, textureD6)
	glctx.Uniform1i(glTexture, 0)

	glctx.DrawElements(gl.TRIANGLES, len(d6IndexData)/2, gl.UNSIGNED_SHORT, 0)

	glctx.DisableVertexAttribArray(glPosition)
	glctx.DisableVertexAttribArray(glUV)
	glctx.DisableVertexAttribArray(glNorm)

	// Disable certain flags before drawing the FPS gauge because they will
	// cause the gauge to be invisible.
	glctx.Disable(gl.CULL_FACE)
	glctx.Disable(gl.DEPTH_TEST)
	fps.Draw(sz)
}

const vertexShader = `#version 100

attribute vec3 vertexPosition;
attribute vec2 vertexUV;
attribute vec3 vertexNormal;

varying vec2 UV;
varying vec3 position;
varying vec3 normalCamera;
varying vec3 eyeDirectionCamera;
varying vec3 lightDirectionCamera;

uniform mat4 MVP;
uniform mat4 V;
uniform mat4 M;
uniform vec3 lightPosition;

void main() {
	gl_Position = MVP * (vec4(vertexPosition + vec3(1, 0, 0), 1));

	position = (M * vec4(vertexPosition, 1)).xyz;

	vec3 vertexPositionCamera = (V * M * vec4(vertexPosition, 1)).xyz;
	eyeDirectionCamera = vec3(0, 0, 0) - vertexPositionCamera;

	vec3 lightPositionCamera = (V * vec4(lightPosition, 1)).xyz;
	lightDirectionCamera = lightPositionCamera + eyeDirectionCamera;

	normalCamera = (V * M * vec4(vertexNormal, 0)).xyz; // Only correct if ModelMatrix does not scale the model ! Use its inverse transpose scaling occurs.

	// this is as it has always been
	UV = vertexUV;
}`

const fragmentShader = `#version 100
precision mediump float;

varying vec2 UV;
varying vec3 position;
varying vec3 normalCamera;
varying vec3 eyeDirectionCamera;
varying vec3 lightDirectionCamera;

uniform sampler2D myTextureSampler;
uniform vec3 lightPosition_mp;
uniform vec3 lightColor;
uniform float lightPower;

void main() {
	vec3 materialDiffuseColor = texture2D(myTextureSampler, UV).rgb;
	vec3 materialAmbientColor = vec3(0.1, 0.1, 0.1) * materialDiffuseColor;
	vec3 materialSpecularColor = vec3(0.3, 0.3, 0.3);

	float lightDistance = length(lightPosition_mp - position);

	vec3 n = normalize(normalCamera);
	vec3 l = normalize(lightDirectionCamera);
	// Cosine of the angle between the normal and the light direction, 
	// clamp above 0
	//  - light is at the vertical of the triangle -> 1
	//  - light is perpendicular to the triangle -> 0
	//  - light is behind the triangle -> 0
	float cosTheta = clamp(dot(n, l), 0.0, 1.0);

	vec3 E = normalize(eyeDirectionCamera);
	vec3 R = reflect(-l, n);
	// Cosine of the angle between the Eye vector and the Reflect vector,
	// clamp to 0
	//  - Looking into the reflection -> 1
	//  - Looking elsewhere -> < 1
	float cosAlpha = clamp(dot(E, R ), 0.0, 1.0);

	gl_FragColor = vec4(
			materialAmbientColor +
				materialDiffuseColor * lightColor * lightPower * cosTheta / (lightDistance * lightDistance) + 
				materialSpecularColor * lightColor * lightPower * pow(cosAlpha, 5.0) / (lightDistance * lightDistance),
			0);
}`
