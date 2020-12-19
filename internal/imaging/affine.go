package imaging

import (
	"math"
)

func identity() []float64 {
	return []float64{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
}

// Rotation Matrix (CCW)
//
//  cos(angle)   -sin(angle)    0
//  sin(angle)    cos(angle)    0
//  0             0             1
//
func rotation(angle float64) []float64 {
	m := identity()
	m[0] = math.Cos(angle)
	m[1] = math.Sin(angle) * -1

	m[3] = math.Sin(angle)
	m[4] = math.Cos(angle)

	return m
}

// Translation Matrix:
//
//  1  0  dx
//  0  1  dy
//  0  0  1
//
func translation(dx, dy float64) []float64 {
	m := identity()

	m[2] = dx
	m[5] = dy

	return m
}

// multiply combines two affine transforms
func multiply(a, b []float64) []float64 {
	m := make([]float64, 9)

	m[0] = a[0]*b[0] + a[1]*b[3] + a[2]*b[6]
	m[1] = a[0]*b[1] + a[1]*b[4] + a[2]*b[7]
	m[2] = a[0]*b[2] + a[1]*b[5] + a[2]*b[8]

	m[3] = a[3]*b[0] + a[4]*b[3] + a[5]*b[6]
	m[4] = a[3]*b[1] + a[4]*b[4] + a[5]*b[7]
	m[5] = a[3]*b[2] + a[4]*b[5] + a[5]*b[8]

	m[6] = a[6]*b[0] + a[7]*b[3] + a[8]*b[6]
	m[7] = a[6]*b[1] + a[7]*b[4] + a[8]*b[7]
	m[8] = a[6]*b[2] + a[7]*b[5] + a[8]*b[8]

	return m
}

// transform applies an affine transform to the given x,y point.
func transform(m []float64, x, y float64) (float64, float64) {
	tx := m[0]*x + m[1]*y + m[2]
	ty := m[3]*x + m[4]*y + m[5]
	return tx, ty
}
