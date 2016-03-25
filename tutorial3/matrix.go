package main

import "golang.org/x/mobile/exp/f32"

func setPerspective(m *f32.Mat4, r f32.Radian, aspect, near, far float32) {
	m.Perspective(r, aspect, near, far)
	transpose4(m)
}

func lookAt(m *f32.Mat4, eye, center, up *f32.Vec3) {
	m.LookAt(eye, center, up)
	transpose4(m)
}

func transpose4(m *f32.Mat4) {
	*m = f32.Mat4{
		{m[0][0], m[1][0], m[2][0], m[3][0]},
		{m[0][1], m[1][1], m[2][1], m[3][1]},
		{m[0][2], m[1][2], m[2][2], m[3][2]},
		{m[0][3], m[1][3], m[2][3], m[3][3]},
	}
}

// serialize4 returns a slice containing m serialized into column-major order.
// If len(dst) is at least 16 then the returned slice will slice of dst will be
// used to serialize the data and reduce allocations.
func serialize4(dst []float32, m *f32.Mat4) []float32 {
	// this serialization considers the matrix vectors to define its rows, the
	// natural representation and how the package documents the type to behave.
	if len(dst) < 16 {
		dst = make([]float32, 16)
	}
	dst = dst[:16]
	dst[0] = m[0][0]
	dst[1] = m[1][0]
	dst[2] = m[2][0]
	dst[3] = m[3][0]
	dst[4] = m[0][1]
	dst[5] = m[1][1]
	dst[6] = m[2][1]
	dst[7] = m[3][1]
	dst[8] = m[0][2]
	dst[9] = m[1][2]
	dst[10] = m[2][2]
	dst[11] = m[3][2]
	dst[12] = m[0][3]
	dst[13] = m[1][3]
	dst[14] = m[2][3]
	dst[15] = m[3][3]
	return dst
}