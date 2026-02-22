package protocol

// Packet22Collect translates net.minecraft.src.Packet22Collect.
type Packet22Collect struct {
	CollectedEntityID int32
	CollectorEntityID int32
}

func (*Packet22Collect) PacketID() uint8 { return 22 }

func (p *Packet22Collect) ReadPacketData(r *Reader) error {
	collected, err := r.ReadInt32()
	if err != nil {
		return err
	}
	collector, err := r.ReadInt32()
	if err != nil {
		return err
	}
	p.CollectedEntityID = collected
	p.CollectorEntityID = collector
	return nil
}

func (p *Packet22Collect) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.CollectedEntityID); err != nil {
		return err
	}
	return w.WriteInt32(p.CollectorEntityID)
}

func (*Packet22Collect) PacketSize() int { return 8 }

func init() {
	_ = Register(22, true, false, func() Packet { return &Packet22Collect{} })
}
