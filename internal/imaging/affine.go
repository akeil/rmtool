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

// transform applies an affine transform to the given x,y point.
func transform(m []float64, x, y float64) (float64, float64) {
	tx := m[0]*x + m[1]*y + m[2]
	ty := m[3]*x + m[4]*y + m[5]
	return tx, ty
}
