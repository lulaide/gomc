package protocol

import (
	"fmt"
	"io"
)

// Packet250CustomPayload translates net.minecraft.src.Packet250CustomPayload.
type Packet250CustomPayload struct {
	Channel string
	Length  int16
	Data    []byte
}

func (*Packet250CustomPayload) PacketID() uint8 { return 250 }

func (p *Packet250CustomPayload) ReadPacketData(r *Reader) error {
	channel, err := r.ReadString(20)
	if err != nil {
		return err
	}
	length, err := r.ReadInt16()
	if err != nil {
		return err
	}

	p.Channel = channel
	p.Length = length
	p.Data = nil

	if length > 0 && length < 32767 {
		data := make([]byte, int(length))
		if _, err := io.ReadFull(r.r, data); err != nil {
			return err
		}
		p.Data = data
	}
	return nil
}

func (p *Packet250CustomPayload) WritePacketData(w *Writer) error {
	if err := w.WriteString(p.Channel); err != nil {
		return err
	}
	if err := w.WriteInt16(p.Length); err != nil {
		return err
	}
	if p.Data != nil {
		_, err := w.w.Write(p.Data)
		return err
	}
	return nil
}

func (p *Packet250CustomPayload) PacketSize() int {
	return 2 + utf16UnitsLen(p.Channel)*2 + 2 + int(p.Length)
}

func NewPacket250CustomPayload(channel string, data []byte) (*Packet250CustomPayload, error) {
	p := &Packet250CustomPayload{
		Channel: channel,
		Data:    data,
		Length:  0,
	}
	if data != nil {
		if len(data) > 32767 {
			return nil, fmt.Errorf("payload may not be larger than 32k")
		}
		p.Length = int16(len(data))
	}
	return p, nil
}

func init() {
	// Translation target: Packet#addIdClassMapping
	_ = Register(250, true, true, func() Packet { return &Packet250CustomPayload{} })
}
