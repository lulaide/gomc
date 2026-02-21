package protocol

import (
	"bytes"
	"testing"
)

func TestPacket2ClientProtocolRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet2ClientProtocol{
		ProtocolVersion: ProtocolVersion,
		Username:        "Steve",
		ServerHost:      "localhost",
		ServerPort:      25565,
	}

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	in, ok := decoded.(*Packet2ClientProtocol)
	if !ok {
		t.Fatalf("decoded packet type mismatch: %T", decoded)
	}
	if in.ProtocolVersion != ProtocolVersion {
		t.Fatalf("protocol version mismatch: got=%d want=%d", in.ProtocolVersion, ProtocolVersion)
	}
	if in.Username != "Steve" || in.ServerHost != "localhost" || in.ServerPort != 25565 {
		t.Fatalf("decoded packet fields mismatch: %#v", in)
	}
}

func TestPacketDirectionValidation(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet2ClientProtocol{
		ProtocolVersion: ProtocolVersion,
		Username:        "Alex",
		ServerHost:      "127.0.0.1",
		ServerPort:      25565,
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	if _, err := ReadPacket(&buf, DirectionClientbound); err == nil {
		t.Fatal("expected direction validation error for packet 2")
	}
}

func TestPacket1LoginReadFallbackWorldTypeAndMode(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	if err := w.WriteUint8(1); err != nil {
		t.Fatalf("write packet id failed: %v", err)
	}
	if err := w.WriteInt32(123); err != nil {
		t.Fatalf("write entity id failed: %v", err)
	}
	if err := w.WriteString("unknown_type"); err != nil {
		t.Fatalf("write world type failed: %v", err)
	}
	if err := w.WriteInt8(9); err != nil { // hardcore bit + creative gametype(1)
		t.Fatalf("write mode failed: %v", err)
	}
	if err := w.WriteInt8(-1); err != nil {
		t.Fatalf("write dimension failed: %v", err)
	}
	if err := w.WriteInt8(2); err != nil {
		t.Fatalf("write difficulty failed: %v", err)
	}
	if err := w.WriteInt8(-128); err != nil {
		t.Fatalf("write world height failed: %v", err)
	}
	if err := w.WriteInt8(20); err != nil {
		t.Fatalf("write max players failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	login, ok := decoded.(*Packet1Login)
	if !ok {
		t.Fatalf("decoded packet type mismatch: %T", decoded)
	}
	if login.TerrainType != "default" {
		t.Fatalf("terrain fallback mismatch: got=%q want=%q", login.TerrainType, "default")
	}
	if !login.HardcoreMode {
		t.Fatal("expected hardcore mode bit to be set")
	}
	if login.GameType != 1 {
		t.Fatalf("game type mismatch: got=%d want=1", login.GameType)
	}
}

func TestPacket205ClientCommandSignedRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet205ClientCommand{ForceRespawn: -1}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	cmd, ok := decoded.(*Packet205ClientCommand)
	if !ok {
		t.Fatalf("decoded packet type mismatch: %T", decoded)
	}
	if cmd.ForceRespawn != -1 {
		t.Fatalf("signed byte mismatch: got=%d want=-1", cmd.ForceRespawn)
	}
}
