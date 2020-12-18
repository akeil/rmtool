package rm

import (
	"testing"
)

func TestValidateDrawing(t *testing.T) {
	var d Drawing
	err := d.Validate()
	if err != nil {
		t.Log("newly initialized drawing should be valid")
		t.Error(err)
	}

	d.Version = Version(100)
	err = d.Validate()
	if err == nil {
		t.Errorf("Invalid version should not be accepted")
	}

	d.Version = V5
	err = d.Validate()
	if err != nil {
		t.Errorf("valid version should be accepted")
	}
}

func TestValidateLayer(t *testing.T) {
	var l Layer
	err := l.Validate()
	if err != nil {
		t.Log("newly initialized layer should be valid")
		t.Error(err)
	}
}

func TestValidateStroke(t *testing.T) {
	var s Stroke
	// TODO: this is the only field that is not initialized to a valid value
	s.BrushSize = Medium

	err := s.Validate()
	if err != nil {
		t.Log("newly initialized stroke should be valid")
		t.Error(err)
	}

	s.BrushType = BrushType(100)
	err = s.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid brush type %v", s.BrushType)
	}
	s.BrushType = BallpointV5
	err = s.Validate()
	if err != nil {
		t.Errorf("valid brush type %v was not accepted: %v", s.BrushType, err)
	}

	s.BrushColor = BrushColor(100)
	err = s.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid brush color %v", s.BrushColor)
	}
	s.BrushColor = Gray
	err = s.Validate()
	if err != nil {
		t.Errorf("valid brush color %v was not accepted: %v", s.BrushColor, err)
	}

	s.BrushSize = BrushSize(0.1)
	err = s.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid brush size %v", s.BrushSize)
	}
	s.BrushSize = Large
	err = s.Validate()
	if err != nil {
		t.Errorf("valid brush size %v was not accepted: %v", s.BrushSize, err)
	}

}

func TestValidateDot(t *testing.T) {
	var d Dot
	err := d.Validate()
	if err != nil {
		t.Log("newly initialized dot should be valid")
		t.Error(err)
	}

	d.Speed = -1
	err = d.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid speed value %v", d.Speed)
	}
	d.Speed = 5.5
	err = d.Validate()
	if err != nil {
		t.Errorf("valid speed value %v was not accepted: %v", d.Speed, err)
	}

	d.Tilt = 3.0
	err = d.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid tilt value %v", d.Tilt)
	}
	d.Tilt = rad(45)
	err = d.Validate()
	if err != nil {
		t.Errorf("valid tilt value %v was not accepted: %v", d.Tilt, err)
	}

	d.Width = -1
	err = d.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid width value %v", d.Width)
	}
	d.Width = 5.5
	err = d.Validate()
	if err != nil {
		t.Errorf("valid width value %v was not accepted: %v", d.Width, err)
	}

	d.Pressure = -1
	err = d.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid pressure value %v", d.Pressure)
	}
	d.Pressure = 2
	err = d.Validate()
	if err == nil {
		t.Errorf("failed to detect invalid pressure value %v", d.Pressure)
	}
	d.Pressure = 0.7
	err = d.Validate()
	if err != nil {
		t.Errorf("valid pressure value %v was not accepted: %v", d.Pressure, err)
	}
}
