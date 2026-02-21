package protocol

// Packet38EntityStatus translates net.minecraft.src.Packet38EntityStatus.
type Packet38EntityStatus struct {
	EntityID     int32
	EntityStatus int8
}

func (*Packet38EntityStatus) PacketID() uint8 { return 38 }

func (p *Packet38EntityStatus) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	status, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.EntityStatus = status
	return nil
}

func (p *Packet38EntityStatus) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	return w.WriteInt8(p.EntityStatus)
}

func (*Packet38EntityStatus) PacketSize() int { return 5 }

func init() {
	// Translation target: Packet#addIdClassMapping for entity status packet.
	_ = Register(38, true, false, func() Packet { return &Packet38EntityStatus{} })
}
