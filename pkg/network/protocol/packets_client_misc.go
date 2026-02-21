package protocol

// Packet16BlockItemSwitch translates net.minecraft.src.Packet16BlockItemSwitch.
type Packet16BlockItemSwitch struct {
	ID int16
}

func (*Packet16BlockItemSwitch) PacketID() uint8 { return 16 }

func (p *Packet16BlockItemSwitch) ReadPacketData(r *Reader) error {
	v, err := r.ReadInt16()
	if err != nil {
		return err
	}
	p.ID = v
	return nil
}

func (p *Packet16BlockItemSwitch) WritePacketData(w *Writer) error {
	return w.WriteInt16(p.ID)
}

func (*Packet16BlockItemSwitch) PacketSize() int { return 2 }

// Packet204ClientInfo translates net.minecraft.src.Packet204ClientInfo.
type Packet204ClientInfo struct {
	Language       string
	RenderDistance int8
	ChatVisible    int8
	ChatColours    bool
	GameDifficulty int8
	ShowCape       bool
}

func (*Packet204ClientInfo) PacketID() uint8 { return 204 }

func (p *Packet204ClientInfo) ReadPacketData(r *Reader) error {
	lang, err := r.ReadString(7)
	if err != nil {
		return err
	}
	renderDistance, err := r.ReadInt8()
	if err != nil {
		return err
	}
	flags, err := r.ReadInt8()
	if err != nil {
		return err
	}
	difficulty, err := r.ReadInt8()
	if err != nil {
		return err
	}
	showCapeByte, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.Language = lang
	p.RenderDistance = renderDistance
	p.ChatVisible = flags & 7
	p.ChatColours = (flags & 8) == 8
	p.GameDifficulty = difficulty
	p.ShowCape = showCapeByte != 0
	return nil
}

func (p *Packet204ClientInfo) WritePacketData(w *Writer) error {
	if err := w.WriteString(p.Language); err != nil {
		return err
	}
	if err := w.WriteInt8(p.RenderDistance); err != nil {
		return err
	}
	flags := p.ChatVisible | (boolToBit(p.ChatColours) << 3)
	if err := w.WriteInt8(flags); err != nil {
		return err
	}
	if err := w.WriteInt8(p.GameDifficulty); err != nil {
		return err
	}
	if p.ShowCape {
		return w.WriteUint8(1)
	}
	return w.WriteUint8(0)
}

func boolToBit(v bool) int8 {
	if v {
		return 1
	}
	return 0
}

func (*Packet204ClientInfo) PacketSize() int {
	// Translation target: Packet204ClientInfo#getPacketSize() returns constant 7.
	return 7
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented misc packets.
	_ = Register(16, true, true, func() Packet { return &Packet16BlockItemSwitch{} })
	_ = Register(204, false, true, func() Packet { return &Packet204ClientInfo{} })
}
