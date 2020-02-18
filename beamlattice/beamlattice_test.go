package beamlattice

import (
	"reflect"
	"testing"
)

func TestCapMode_String(t *testing.T) {
	tests := []struct {
		name string
		b    CapMode
	}{
		{"sphere", CapModeSphere},
		{"hemisphere", CapModeHemisphere},
		{"butt", CapModeButt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.String(); got != tt.name {
				t.Errorf("CapMode.String() = %v, want %v", got, tt.name)
			}
		})
	}
}

func Test_newCapMode(t *testing.T) {
	tests := []struct {
		name   string
		wantT  CapMode
		wantOk bool
	}{
		{"sphere", CapModeSphere, true},
		{"hemisphere", CapModeHemisphere, true},
		{"butt", CapModeButt, true},
		{"empty", CapModeSphere, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotOk := newCapMode(tt.name)
			if !reflect.DeepEqual(gotT, tt.wantT) {
				t.Errorf("newCapMode() gotT = %v, want %v", gotT, tt.wantT)
			}
			if gotOk != tt.wantOk {
				t.Errorf("newCapMode() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_newClipMode(t *testing.T) {
	tests := []struct {
		name   string
		wantC  ClipMode
		wantOk bool
	}{
		{"none", ClipNone, true},
		{"inside", ClipInside, true},
		{"outside", ClipOutside, true},
		{"empty", ClipNone, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, gotOk := newClipMode(tt.name)
			if !reflect.DeepEqual(gotC, tt.wantC) {
				t.Errorf("newClipMode() gotC = %v, want %v", gotC, tt.wantC)
			}
			if gotOk != tt.wantOk {
				t.Errorf("newClipMode() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestClipMode_String(t *testing.T) {
	tests := []struct {
		name string
		c    ClipMode
	}{
		{"none", ClipNone},
		{"inside", ClipInside},
		{"outside", ClipOutside},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.name {
				t.Errorf("ClipMode.String() = %v, want %v", got, tt.name)
			}
		})
	}
}

func TestBeamLattice_checkSanity(t *testing.T) {
	type args struct {
		nodeCount uint32
	}
	tests := []struct {
		name string
		m    *BeamLattice
		args args
		want bool
	}{
		{"eq", &BeamLattice{Beams: []Beam{{NodeIndices: [2]uint32{1, 1}}}}, args{0}, false},
		{"high1", &BeamLattice{Beams: []Beam{{NodeIndices: [2]uint32{2, 1}}}}, args{2}, false},
		{"high2", &BeamLattice{Beams: []Beam{{NodeIndices: [2]uint32{1, 2}}}}, args{2}, false},
		{"good", &BeamLattice{Beams: []Beam{{NodeIndices: [2]uint32{1, 2}}}}, args{3}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.checkSanity(tt.args.nodeCount); got != tt.want {
				t.Errorf("BeamLattice.checkSanity() = %v, want %v", got, tt.want)
			}
		})
	}
}