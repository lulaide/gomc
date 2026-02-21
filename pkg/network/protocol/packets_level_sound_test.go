package protocol

import (
	"bytes"
	"testing"
)

func TestPacket62LevelSoundRoundTrip(t *testing.T) {
	out := &Packet62LevelSound{
		SoundName: "step.grass",
		EffectX:   12 * 8,
		EffectY:   65 * 8,
		EffectZ:   -4 * 8,
		Volume:    0.25,
		Pitch:     63,
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	inPkt, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := inPkt.(*Packet62LevelSound)
	if !ok {
		t.Fatalf("expected Packet62LevelSound, got %T", inPkt)
	}
	if in.SoundName != out.SoundName {
		t.Fatalf("sound mismatch: got=%q want=%q", in.SoundName, out.SoundName)
	}
	if in.EffectX != out.EffectX || in.EffectY != out.EffectY || in.EffectZ != out.EffectZ {
		t.Fatalf("effect mismatch: got=(%d,%d,%d) want=(%d,%d,%d)",
			in.EffectX, in.EffectY, in.EffectZ, out.EffectX, out.EffectY, out.EffectZ)
	}
	if in.Volume != out.Volume || in.Pitch != out.Pitch {
		t.Fatalf("audio mismatch: got=(%f,%d) want=(%f,%d)", in.Volume, in.Pitch, out.Volume, out.Pitch)
	}
	if in.PitchFloat() != 1.0 {
		t.Fatalf("pitch float mismatch: got=%f want=1.0", in.PitchFloat())
	}
}
