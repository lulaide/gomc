package protocol

// Packet201PlayerInfo translates net.minecraft.src.Packet201PlayerInfo.
type Packet201PlayerInfo struct {
	PlayerName  string
	IsConnected bool
	Ping        int16
}

func (*Packet201PlayerInfo) PacketID() uint8 { return 201 }

func (p *Packet201PlayerInfo) ReadPacketData(r *Reader) error {
	name, err := r.ReadString(16)
	if err != nil {
		return err
	}
	connected, err := r.ReadInt8()
	if err != nil {
		return err
	}
	ping, err := r.ReadInt16()
	if err != nil {
		return err
	}

	p.PlayerName = name
	p.IsConnected = connected != 0
	p.Ping = ping
	return nil
}

func (p *Packet201PlayerInfo) WritePacketData(w *Writer) error {
	if err := w.WriteString(p.PlayerName); err != nil {
		return err
	}
	if p.IsConnected {
		if err := w.WriteInt8(1); err != nil {
			return err
		}
	} else {
		if err := w.WriteInt8(0); err != nil {
			return err
		}
	}
	return w.WriteInt16(p.Ping)
}

func (p *Packet201PlayerInfo) PacketSize() int {
	return utf16UnitsLen(p.PlayerName) + 5
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented player info packet.
	_ = Register(201, true, false, func() Packet { return &Packet201PlayerInfo{} })
}
