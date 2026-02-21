package protocol

import (
	"bytes"
	"testing"

	"github.com/lulaide/gomc/pkg/world/chunk"
)

func buildProtocolTestChunk(x, z int32) *chunk.Chunk {
	ch := chunk.NewChunk(nil, x, z)
	sec0 := chunk.NewExtendedBlockStorage(0, true)
	sec0.SetExtBlockID(1, 1, 1, 1)
	sec0.SetExtBlockMetadata(1, 1, 1, 2)
	sec0.SetExtBlocklightValue(1, 1, 1, 3)
	sec0.SetExtSkylightValue(1, 1, 1, 15)

	sec1 := chunk.NewExtendedBlockStorage(16, true)
	sec1.SetExtBlockID(2, 2, 2, 300) // ensures Add array exists
	sec1.SetExtBlockMetadata(2, 2, 2, 5)
	sec1.SetExtBlocklightValue(2, 2, 2, 6)
	sec1.SetExtSkylightValue(2, 2, 2, 10)

	arr := ch.GetBlockStorageArray()
	arr[0] = sec0
	arr[1] = sec1
	ch.SetBlockStorageArray(arr)
	return ch
}

func TestPacket51MapChunkRoundTrip(t *testing.T) {
	ch := buildProtocolTestChunk(2, 3)
	out, err := NewPacket51MapChunk(ch, true, 65535, false)
	if err != nil {
		t.Fatalf("NewPacket51MapChunk failed: %v", err)
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet51MapChunk)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.XCh != 2 || in.ZCh != 3 {
		t.Fatalf("chunk coords mismatch: got=(%d,%d)", in.XCh, in.ZCh)
	}
	if !in.IncludeInitialize {
		t.Fatal("expected includeInitialize=true")
	}
	if len(in.GetCompressedChunkData()) == 0 {
		t.Fatal("expected decompressed chunk payload")
	}
	if in.PacketSize() <= 17 {
		t.Fatalf("packet size too small: %d", in.PacketSize())
	}
}

func TestPacket56MapChunksRoundTrip(t *testing.T) {
	ch1 := buildProtocolTestChunk(0, 0)
	ch2 := buildProtocolTestChunk(1, 0)
	out, err := NewPacket56MapChunks([]*chunk.Chunk{ch1, ch2}, false)
	if err != nil {
		t.Fatalf("NewPacket56MapChunks failed: %v", err)
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, out); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	decoded, err := ReadPacket(&buf, DirectionClientbound)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}
	in, ok := decoded.(*Packet56MapChunks)
	if !ok {
		t.Fatalf("type mismatch: %T", decoded)
	}
	if in.GetNumberOfChunkInPacket() != 2 {
		t.Fatalf("chunk count mismatch: got=%d want=2", in.GetNumberOfChunkInPacket())
	}
	if in.GetChunkPosX(0) != 0 || in.GetChunkPosX(1) != 1 {
		t.Fatalf("chunk x mismatch: got=(%d,%d)", in.GetChunkPosX(0), in.GetChunkPosX(1))
	}
	if len(in.GetChunkCompressedData(0)) == 0 || len(in.GetChunkCompressedData(1)) == 0 {
		t.Fatal("expected per-chunk payload")
	}
}
