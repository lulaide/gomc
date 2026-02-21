package protocol

import (
	"bytes"
	"testing"
)

func TestPacket16BlockItemSwitchRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet16BlockItemSwitch{ID: 5}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet16BlockItemSwitch)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.ID != 5 {
		t.Fatalf("id mismatch: got=%d want=5", in.ID)
	}
}

func TestPacket204ClientInfoRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet204ClientInfo{
		Language:       "zh_CN",
		RenderDistance: 2,
		ChatVisible:    1,
		ChatColours:    true,
		GameDifficulty: 2,
		ShowCape:       true,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet204ClientInfo)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.Language != "zh_CN" || !in.ChatColours || !in.ShowCape || in.ChatVisible != 1 {
		t.Fatalf("decoded mismatch: %#v", in)
	}
	if in.PacketSize() != 7 {
		t.Fatalf("packet size mismatch: got=%d want=7", in.PacketSize())
	}
}
