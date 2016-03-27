package mobtex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unsafe"

	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/exp/f32"
)

// VBO is an indexed Obj
type VBO struct {
	Index []int
	Obj
}

// IndexVBO builds an index over the given vertices.
func IndexVBO(in *Obj) *VBO {
	return nil
}

// Obj contains the contents of an OBJ file.
type Obj struct {
	V  []f32.Vec3
	VT []Vec2
	VN []f32.Vec3
}

// DecodeObjPath loads an object asset at path using the DecodeObj function as
// a helper.
func DecodeObjPath(path string) (*Obj, error) {
	f, err := asset.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return DecodeObj(f)
}

// DecodeObj loads an object (OBJ) byte stream from r.
func DecodeObj(r io.Reader) (*Obj, error) {
	var err error
	var vxIndices, uvIndices, normIndices []int
	var vxTemp []f32.Vec3
	var uvTemp []Vec2
	var normTemp []f32.Vec3

	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			return nil, fmt.Errorf("blank line")
		}
		if line[0] == '#' {
			continue
		}
		fields := bytes.Fields(line)
		head, vector := fields[0], fields[1:]
		switch {
		case bytes.Equal(head, []byte("v")):
			if len(vector) != 3 {
				return nil, fmt.Errorf("invalid vertex")
			}
			var v f32.Vec3
			v[0], err = parseFloat32(vector[0])
			if err != nil {
				return nil, fmt.Errorf("invalid vertex: %v", err)
			}
			v[1], err = parseFloat32(vector[1])
			if err != nil {
				return nil, fmt.Errorf("invalid vertex: %v", err)
			}
			v[2], err = parseFloat32(vector[2])
			if err != nil {
				return nil, fmt.Errorf("invalid vertex: %v", err)
			}
			vxTemp = append(vxTemp, v)
		case bytes.Equal(head, []byte("vt")):
			if len(vector) != 2 {
				return nil, fmt.Errorf("invalid texture coords")
			}
			var vt Vec2
			vt[0], err = parseFloat32(vector[0])
			if err != nil {
				return nil, fmt.Errorf("invalid texture coords: %v", err)
			}
			vt[1], err = parseFloat32(vector[1])
			if err != nil {
				return nil, fmt.Errorf("invalid texture coords: %v", err)
			}
			uvTemp = append(uvTemp, vt)
		case bytes.Equal(head, []byte("vn")):
			if len(vector) != 3 {
				return nil, fmt.Errorf("invalid normal")
			}
			var vn f32.Vec3
			vn[0], err = parseFloat32(vector[0])
			if err != nil {
				return nil, fmt.Errorf("invalid normal: %v", err)
			}
			vn[1], err = parseFloat32(vector[1])
			if err != nil {
				return nil, fmt.Errorf("invalid normal: %v", err)
			}
			vn[2], err = parseFloat32(vector[2])
			if err != nil {
				return nil, fmt.Errorf("invalid normal: %v", err)
			}
			normTemp = append(normTemp, vn)
		case bytes.Equal(head, []byte("f")):
			if len(vector) != 3 {
				return nil, fmt.Errorf("invalid face %q", vector)
			}
			vx1, uv1, norm1, err := parseIndices(vector[0])
			if err != nil {
				return nil, fmt.Errorf("invalid face: %v", err)
			}
			vx2, uv2, norm2, err := parseIndices(vector[1])
			if err != nil {
				return nil, fmt.Errorf("invalid face: %v", err)
			}
			vx3, uv3, norm3, err := parseIndices(vector[2])
			if err != nil {
				return nil, fmt.Errorf("invalid face: %v", err)
			}

			vxIndices = append(vxIndices, vx1, vx2, vx3)
			uvIndices = append(uvIndices, uv1, uv2, uv3)
			normIndices = append(normIndices, norm1, norm2, norm3)
		case bytes.Equal(head, []byte("s")):
			// ?
			continue
		}
	}
	if s.Err() != nil {
		return nil, err
	}

	obj := &Obj{}
	for i := range vxIndices {
		obj.V = append(obj.V, vxTemp[vxIndices[i]-1])
	}
	for i := range uvIndices {
		obj.VT = append(obj.VT, uvTemp[uvIndices[i]-1])
	}
	for i := range normIndices {
		obj.VN = append(obj.VN, normTemp[normIndices[i]-1])
	}

	return obj, err
}

func parseIndices(b []byte) (int, int, int, error) {
	fields := bytes.SplitN(b, []byte("/"), 3)
	x1, err := strconv.Atoi(*(*string)(unsafe.Pointer(&fields[0])))
	if err != nil {
		return 0, 0, 0, err
	}
	x2, err := strconv.Atoi(*(*string)(unsafe.Pointer(&fields[1])))
	if err != nil {
		return 0, 0, 0, err
	}
	x3, err := strconv.Atoi(*(*string)(unsafe.Pointer(&fields[2])))
	if err != nil {
		return 0, 0, 0, err
	}
	return x1, x2, x3, nil
}

func parseFloat32(b []byte) (float32, error) {
	f64, err := strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 32)
	return float32(f64), err
}

// Vec2 is a 2-dimensional vector with 32 bit precision
type Vec2 [2]float32
