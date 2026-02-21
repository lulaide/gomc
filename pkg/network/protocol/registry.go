package protocol

import (
	"fmt"
	"io"
)

type packetRegistration struct {
	clientbound bool
	serverbound bool
	newPacket   func() Packet
}

// Registry translates Packet id<->class registration and direction checks.
//
// Translation target:
// - net.minecraft.src.Packet#addIdClassMapping
// - net.minecraft.src.Packet#readPacket
type Registry struct {
	byID map[uint8]packetRegistration
}

func NewRegistry() *Registry {
	return &Registry{
		byID: make(map[uint8]packetRegistration),
	}
}

func (r *Registry) Register(packetID uint8, clientbound bool, serverbound bool, ctor func() Packet) error {
	if ctor == nil {
		return fmt.Errorf("packet constructor is nil for id %d", packetID)
	}
	if _, exists := r.byID[packetID]; exists {
		return fmt.Errorf("duplicate packet id %d", packetID)
	}
	r.byID[packetID] = packetRegistration{
		clientbound: clientbound,
		serverbound: serverbound,
		newPacket:   ctor,
	}
	return nil
}

func (r *Registry) ensureDirection(packetID uint8, direction Direction) error {
	reg, ok := r.byID[packetID]
	if !ok {
		return fmt.Errorf("bad packet id %d", packetID)
	}
	if direction == DirectionClientbound && !reg.clientbound {
		return fmt.Errorf("bad packet id %d for clientbound stream", packetID)
	}
	if direction == DirectionServerbound && !reg.serverbound {
		return fmt.Errorf("bad packet id %d for serverbound stream", packetID)
	}
	return nil
}

func (r *Registry) ReadPacket(in io.Reader, direction Direction) (Packet, error) {
	rd := NewReader(in)
	packetID, err := rd.ReadUint8()
	if err != nil {
		return nil, err
	}

	if err := r.ensureDirection(packetID, direction); err != nil {
		return nil, err
	}

	reg := r.byID[packetID]
	packet := reg.newPacket()
	if packet == nil {
		return nil, fmt.Errorf("constructor returned nil packet for id %d", packetID)
	}
	if err := packet.ReadPacketData(rd); err != nil {
		return nil, err
	}
	return packet, nil
}

func (r *Registry) WritePacket(out io.Writer, packet Packet) error {
	if packet == nil {
		return fmt.Errorf("packet is nil")
	}

	packetID := packet.PacketID()
	if _, ok := r.byID[packetID]; !ok {
		return fmt.Errorf("unregistered packet id %d", packetID)
	}

	wr := NewWriter(out)
	if err := wr.WriteUint8(packetID); err != nil {
		return err
	}
	return packet.WritePacketData(wr)
}

var DefaultRegistry = NewRegistry()

func Register(packetID uint8, clientbound bool, serverbound bool, ctor func() Packet) error {
	return DefaultRegistry.Register(packetID, clientbound, serverbound, ctor)
}

func ReadPacket(in io.Reader, direction Direction) (Packet, error) {
	return DefaultRegistry.ReadPacket(in, direction)
}

func WritePacket(out io.Writer, packet Packet) error {
	return DefaultRegistry.WritePacket(out, packet)
}
