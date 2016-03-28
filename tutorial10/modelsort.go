package main

import (
	"log"

	"github.com/bmatsuo/mobile-gl-tutorial/mobtex"
	"golang.org/x/mobile/exp/f32"
)

// facesByDepth orders the faces in a VBO by their depth, according to the
// model and camera positions.  It is assumed that faces do not overlap.  If
// faces overlap then there is no way to properly order them for rendering with
// transparency.
type facesByDepth struct {
	vbo *mobtex.VBO
	mv  f32.Mat4
	cam []float32 // face positions relative to the camera
	tmp f32.Vec3
}

// prepareFacesByDepth constructs a index over face depth so that faces can be
// sorted.
func prepareFacesByDepth(vbo *mobtex.VBO, m, v *f32.Mat4) *facesByDepth {
	d := newFacesByDepth(vbo)
	d.Update(m, v)
	return d
}

func newFacesByDepth(vbo *mobtex.VBO) *facesByDepth {
	if len(vbo.Index)%3 != 0 {
		panic("number of vertices is not a multiple of three")
	}

	return &facesByDepth{
		vbo: vbo,
		cam: make([]float32, len(vbo.Index)),
	}
}

// Update must be called any time the model or view matrices change.  It causes
// the cached depth index to be recalculated with the new relative model-camera
// positions.
func (d *facesByDepth) Update(m, v *f32.Mat4) {
	d.mv.Mul(v, m) // order reversed because model transform applied first
	d.update()
}

func (d *facesByDepth) update() {
	for i := range d.cam {
		affine43(&d.tmp, &d.mv, &d.vbo.V[d.vbo.Index[i]])
		log.Printf("%2d    POS=%v", i, d.tmp)
		d.cam[i] = f32.Sqrt(d.tmp[0]*d.tmp[0] + d.tmp[1]*d.tmp[1] + d.tmp[2]*d.tmp[2])
		log.Printf("%2d   DIST=%v", i, d.cam[i])
	}
}

func (d *facesByDepth) Len() int {
	return len(d.cam) / 3
}

func (d *facesByDepth) Swap(i, j int) {
	// swap the three vertex indices associated with faces i and j. relative
	// order of vertices on each face must be maintained.
	d.cam[3*i+0], d.cam[3*j+0] = d.cam[3*j+0], d.cam[3*i+0]
	d.cam[3*i+1], d.cam[3*j+1] = d.cam[3*j+1], d.cam[3*i+1]
	d.cam[3*i+2], d.cam[3*j+2] = d.cam[3*j+2], d.cam[3*i+2]
	d.vbo.Index[3*i+0], d.vbo.Index[3*j+0] = d.vbo.Index[3*j+0], d.vbo.Index[3*i+0]
	d.vbo.Index[3*i+1], d.vbo.Index[3*j+1] = d.vbo.Index[3*j+1], d.vbo.Index[3*i+1]
	d.vbo.Index[3*i+2], d.vbo.Index[3*j+2] = d.vbo.Index[3*j+2], d.vbo.Index[3*i+2]
}

// Less returns true if face i is farther away than face j.
// The resulting sort places the furthest faces in the front of the vbo index list
func (d *facesByDepth) Less(i, j int) bool {
	if d.cam[i+0] > d.cam[j+0] {
		return true
	}
	if d.cam[i+1] > d.cam[j+1] {
		return true
	}
	if d.cam[i+2] > d.cam[j+2] {
		return true
	}
	return false
	//return d.cam[i] > d.cam[j] && d.cam[i]-d.cam[j] > 1e-3
}

// center computes the center of v1, v2, and v3, storing the result in c.
func center(c *f32.Vec3, v1, v2, v3 *f32.Vec3) {
	c.Add(v1, v2)
	c.Add(c, v3)
	c[0] /= 3.0
	c[1] /= 3.0
	c[2] /= 3.0
}

// affine43 computes the affine matrix transformation m on v and stores the
// result in u.  The result expressed using the following using OpenGL vector
// construction and swizzle notation.
//		u = (m * vec4(v, 1)]).xyz
//
// TODO:
// figure out if affine in the right name.
func affine43(u *f32.Vec3, m *f32.Mat4, v *f32.Vec3) {
	*u = f32.Vec3{
		dot43(&m[0], v),
		dot43(&m[1], v),
		dot43(&m[2], v),
	}
}

func dot43(u *f32.Vec4, v *f32.Vec3) float32 {
	return u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
}
