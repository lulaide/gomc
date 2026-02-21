package protocol

import (
	"bytes"
	"testing"
)

func TestPacket252SharedKeyRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet252SharedKey{
		SharedSecret: []byte{1, 2, 3, 4},
		VerifyToken:  []byte{9, 8, 7, 6},
	}

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet252SharedKey)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if len(in.SharedSecret) != 4 || len(in.VerifyToken) != 4 {
		t.Fatalf("decoded lengths mismatch: secret=%d token=%d", len(in.SharedSecret), len(in.VerifyToken))
	}
}

func TestPacket253ServerAuthDataRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet253ServerAuthData{
		ServerID:    "abc123",
		PublicKey:   []byte{11, 22, 33},
		VerifyToken: []byte{44, 55, 66, 77},
	}
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet253ServerAuthData)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.ServerID != "abc123" {
		t.Fatalf("server id mismatch: got=%q", in.ServerID)
	}
}

func TestPacket254ServerPingRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	out := &Packet254ServerPing{
		ReadSuccessfully: 78,
		ServerHost:       "localhost",
		ServerPort:       25565,
	}

	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet254ServerPing)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.ReadSuccessfully != 78 {
		t.Fatalf("version mismatch: got=%d want=78", in.ReadSuccessfully)
	}
	if in.ServerHost != "localhost" || in.ServerPort != 25565 {
		t.Fatalf("host/port mismatch: host=%q port=%d", in.ServerHost, in.ServerPort)
	}
	if in.IsLegacyPing() {
		t.Fatal("expected non-legacy ping")
	}
}

func TestPacket254ServerPingToleratesTruncatedPayload(t *testing.T) {
	var payload bytes.Buffer
	w := NewWriter(&payload)
	if err := w.WriteUint8(254); err != nil {
		t.Fatalf("write id failed: %v", err)
	}
	if err := w.WriteInt8(1); err != nil {
		t.Fatalf("write first byte failed: %v", err)
	}

	decoded, err := ReadPacket(&payload, DirectionServerbound)
	if err != nil {
		t.Fatalf("ReadPacket should not fail on truncated ping payload: %v", err)
	}
	in, ok := decoded.(*Packet254ServerPing)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.ReadSuccessfully != 1 {
		t.Fatalf("readSuccessfully mismatch: got=%d want=1", in.ReadSuccessfully)
	}
	if in.ServerHost != "" {
		t.Fatalf("expected empty server host for truncated payload, got=%q", in.ServerHost)
	}
}
