package gui

import "math"

type viewFrustum struct {
	planes [6][4]float64
	valid  bool
}

func newViewFrustumFromGL(projection, modelView [16]float32) viewFrustum {
	// OpenGL provides column-major matrices. Build combined clip matrix:
	// clip = projection * modelView (still column-major), then convert to row-major
	// for plane extraction.
	var clipCol [16]float64
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			clipCol[col*4+row] =
				float64(projection[0*4+row])*float64(modelView[col*4+0]) +
					float64(projection[1*4+row])*float64(modelView[col*4+1]) +
					float64(projection[2*4+row])*float64(modelView[col*4+2]) +
					float64(projection[3*4+row])*float64(modelView[col*4+3])
		}
	}

	var m [16]float64 // row-major
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			m[row*4+col] = clipCol[col*4+row]
		}
	}

	f := viewFrustum{valid: true}
	// Left, Right, Bottom, Top, Near, Far
	f.planes[0] = [4]float64{m[3] + m[0], m[7] + m[4], m[11] + m[8], m[15] + m[12]}
	f.planes[1] = [4]float64{m[3] - m[0], m[7] - m[4], m[11] - m[8], m[15] - m[12]}
	f.planes[2] = [4]float64{m[3] + m[1], m[7] + m[5], m[11] + m[9], m[15] + m[13]}
	f.planes[3] = [4]float64{m[3] - m[1], m[7] - m[5], m[11] - m[9], m[15] - m[13]}
	f.planes[4] = [4]float64{m[3] + m[2], m[7] + m[6], m[11] + m[10], m[15] + m[14]}
	f.planes[5] = [4]float64{m[3] - m[2], m[7] - m[6], m[11] - m[10], m[15] - m[14]}

	for i := range f.planes {
		norm := math.Sqrt(
			f.planes[i][0]*f.planes[i][0] +
				f.planes[i][1]*f.planes[i][1] +
				f.planes[i][2]*f.planes[i][2],
		)
		if norm <= 0 {
			f.valid = false
			return f
		}
		inv := 1.0 / norm
		f.planes[i][0] *= inv
		f.planes[i][1] *= inv
		f.planes[i][2] *= inv
		f.planes[i][3] *= inv
	}
	return f
}

func (f viewFrustum) aabbVisible(minX, minY, minZ, maxX, maxY, maxZ float64) bool {
	if !f.valid {
		return true
	}
	for i := range f.planes {
		p := f.planes[i]
		x := minX
		if p[0] >= 0 {
			x = maxX
		}
		y := minY
		if p[1] >= 0 {
			y = maxY
		}
		z := minZ
		if p[2] >= 0 {
			z = maxZ
		}
		if p[0]*x+p[1]*y+p[2]*z+p[3] < 0 {
			return false
		}
	}
	return true
}
