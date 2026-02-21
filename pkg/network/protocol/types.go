package protocol

import "github.com/lulaide/gomc/pkg/nbt"

const ProtocolVersion = 78

// Direction indicates the side currently decoding packets.
//
// - DirectionClientbound: packets the client is allowed to receive.
// - DirectionServerbound: packets the server is allowed to receive.
type Direction int

const (
	DirectionClientbound Direction = iota
	DirectionServerbound
)

// Packet translates net.minecraft.src.Packet raw serialization contract.
type Packet interface {
	PacketID() uint8
	ReadPacketData(r *Reader) error
	WritePacketData(w *Writer) error
	PacketSize() int
}

// ItemStack is packet-level stack representation for 1.6.4 stream helpers.
//
// Translation target:
// - net.minecraft.src.Packet#readItemStack
// - net.minecraft.src.Packet#writeItemStack
type ItemStack struct {
	ItemID     int16
	StackSize  int8
	ItemDamage int16
	Tag        *nbt.CompoundTag
}
