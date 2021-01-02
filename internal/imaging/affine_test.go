package imaging

import (
	"math"
	"testing"
)

func TestRotation(t *testing.T) {
	x := 1
	y := 2
	rad := 90 * math.Pi / 180

	rot := rotation(rad)
	tx, ty := transform(rot, float64(x), float64(y))

	if math.Round(tx) != -2 {
		t.Errorf("unexpected value for transformed x: %v", tx)
	}
	if math.Round(ty) != 1 {
		t.Errorf("unexpected value for transformed y: %v", ty)
	}

	// translating around the center should result in the same point
	t0 := translation(float64(-x), float64(-y))
	tx, ty = transform(t0, float64(x), float64(y))
	tx, ty = transform(rot, float64(tx), float64(ty))

	if math.Round(tx) != 0 {
		t.Errorf("unexpected value for transformed x: %v", tx)
	}
	if math.Round(ty) != 0 {
		t.Errorf("unexpected value for transformed y: %v", ty)
	}
}
