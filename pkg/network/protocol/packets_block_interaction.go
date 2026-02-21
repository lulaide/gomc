package protocol

// Packet14BlockDig translates net.minecraft.src.Packet14BlockDig.
type Packet14BlockDig struct {
	Status    int32
	XPosition int32
	YPosition int32
	ZPosition int32
	Face      int32
}

func (*Packet14BlockDig) PacketID() uint8 { return 14 }

func (p *Packet14BlockDig) ReadPacketData(r *Reader) error {
	status, err := r.ReadUint8()
	if err != nil {
		return err
	}
	x, err := r.ReadInt32()
	if err != nil {
		return err
	}
	y, err := r.ReadUint8()
	if err != nil {
		return err
	}
	z, err := r.ReadInt32()
	if err != nil {
		return err
	}
	face, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.Status = int32(status)
	p.XPosition = x
	p.YPosition = int32(y)
	p.ZPosition = z
	p.Face = int32(face)
	return nil
}

func (p *Packet14BlockDig) WritePacketData(w *Writer) error {
	if err := w.WriteUint8(uint8(p.Status)); err != nil {
		return err
	}
	if err := w.WriteInt32(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(p.YPosition)); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ZPosition); err != nil {
		return err
	}
	return w.WriteUint8(uint8(p.Face))
}

func (*Packet14BlockDig) PacketSize() int { return 11 }

// Packet15Place translates net.minecraft.src.Packet15Place.
type Packet15Place struct {
	XPosition int32
	YPosition int32
	ZPosition int32
	Direction int32
	ItemStack *ItemStack
	XOffset   float32
	YOffset   float32
	ZOffset   float32
}

func (*Packet15Place) PacketID() uint8 { return 15 }

func (p *Packet15Place) ReadPacketData(r *Reader) error {
	x, err := r.ReadInt32()
	if err != nil {
		return err
	}
	y, err := r.ReadUint8()
	if err != nil {
		return err
	}
	z, err := r.ReadInt32()
	if err != nil {
		return err
	}
	direction, err := r.ReadUint8()
	if err != nil {
		return err
	}
	stack, err := r.ReadItemStack()
	if err != nil {
		return err
	}
	xOffsetRaw, err := r.ReadUint8()
	if err != nil {
		return err
	}
	yOffsetRaw, err := r.ReadUint8()
	if err != nil {
		return err
	}
	zOffsetRaw, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.XPosition = x
	p.YPosition = int32(y)
	p.ZPosition = z
	p.Direction = int32(direction)
	p.ItemStack = stack
	p.XOffset = float32(xOffsetRaw) / 16.0
	p.YOffset = float32(yOffsetRaw) / 16.0
	p.ZOffset = float32(zOffsetRaw) / 16.0
	return nil
}

func (p *Packet15Place) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(p.YPosition)); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ZPosition); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(p.Direction)); err != nil {
		return err
	}
	if err := w.WriteItemStack(p.ItemStack); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(int(p.XOffset * 16.0))); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(int(p.YOffset * 16.0))); err != nil {
		return err
	}
	return w.WriteUint8(uint8(int(p.ZOffset * 16.0)))
}

func (*Packet15Place) PacketSize() int { return 19 }

// Packet53BlockChange translates net.minecraft.src.Packet53BlockChange.
type Packet53BlockChange struct {
	XPosition int32
	YPosition int32
	ZPosition int32
	Type      int32
	Metadata  int32
}

func (*Packet53BlockChange) PacketID() uint8 { return 53 }

func (p *Packet53BlockChange) ReadPacketData(r *Reader) error {
	x, err := r.ReadInt32()
	if err != nil {
		return err
	}
	y, err := r.ReadUint8()
	if err != nil {
		return err
	}
	z, err := r.ReadInt32()
	if err != nil {
		return err
	}
	typ, err := r.ReadInt16()
	if err != nil {
		return err
	}
	meta, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.XPosition = x
	p.YPosition = int32(y)
	p.ZPosition = z
	p.Type = int32(typ)
	p.Metadata = int32(meta)
	return nil
}

func (p *Packet53BlockChange) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteUint8(uint8(p.YPosition)); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ZPosition); err != nil {
		return err
	}
	if err := w.WriteInt16(int16(p.Type)); err != nil {
		return err
	}
	return w.WriteUint8(uint8(p.Metadata))
}

func (*Packet53BlockChange) PacketSize() int { return 11 }

func init() {
	// Translation target: Packet#addIdClassMapping for implemented block interaction packets.
	_ = Register(14, false, true, func() Packet { return &Packet14BlockDig{} })
	_ = Register(15, false, true, func() Packet { return &Packet15Place{} })
	_ = Register(53, true, false, func() Packet { return &Packet53BlockChange{} })
}
