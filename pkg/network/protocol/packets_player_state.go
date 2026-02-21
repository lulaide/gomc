package protocol

// Packet43Experience translates net.minecraft.src.Packet43Experience.
type Packet43Experience struct {
	Experience      float32
	ExperienceTotal int16
	ExperienceLevel int16
}

func (*Packet43Experience) PacketID() uint8 { return 43 }

func (p *Packet43Experience) ReadPacketData(r *Reader) error {
	exp, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	level, err := r.ReadInt16()
	if err != nil {
		return err
	}
	total, err := r.ReadInt16()
	if err != nil {
		return err
	}

	p.Experience = exp
	p.ExperienceLevel = level
	p.ExperienceTotal = total
	return nil
}

func (p *Packet43Experience) WritePacketData(w *Writer) error {
	if err := w.WriteFloat32(p.Experience); err != nil {
		return err
	}
	if err := w.WriteInt16(p.ExperienceLevel); err != nil {
		return err
	}
	return w.WriteInt16(p.ExperienceTotal)
}

func (*Packet43Experience) PacketSize() int {
	// Translation target: Packet43Experience#getPacketSize returns constant 4 in MCP.
	return 4
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented player state packets.
	_ = Register(43, true, false, func() Packet { return &Packet43Experience{} })
}
