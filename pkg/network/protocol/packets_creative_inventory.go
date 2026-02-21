package protocol

// Packet107CreativeSetSlot translates net.minecraft.src.Packet107CreativeSetSlot.
type Packet107CreativeSetSlot struct {
	Slot      int16
	ItemStack *ItemStack
}

func (*Packet107CreativeSetSlot) PacketID() uint8 { return 107 }

func (p *Packet107CreativeSetSlot) ReadPacketData(r *Reader) error {
	slot, err := r.ReadInt16()
	if err != nil {
		return err
	}
	stack, err := r.ReadItemStack()
	if err != nil {
		return err
	}
	p.Slot = slot
	p.ItemStack = stack
	return nil
}

func (p *Packet107CreativeSetSlot) WritePacketData(w *Writer) error {
	if err := w.WriteInt16(p.Slot); err != nil {
		return err
	}
	return w.WriteItemStack(p.ItemStack)
}

func (*Packet107CreativeSetSlot) PacketSize() int {
	// Translation target: Packet107CreativeSetSlot#getPacketSize returns 8 in MCP.
	return 8
}

func init() {
	// Translation target: Packet#addIdClassMapping for creative set slot packet.
	_ = Register(107, false, true, func() Packet { return &Packet107CreativeSetSlot{} })
}
