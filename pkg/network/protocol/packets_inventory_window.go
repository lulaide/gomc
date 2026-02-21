package protocol

// Packet103SetSlot translates net.minecraft.src.Packet103SetSlot.
type Packet103SetSlot struct {
	WindowID  int8
	ItemSlot  int16
	ItemStack *ItemStack
}

func (*Packet103SetSlot) PacketID() uint8 { return 103 }

func (p *Packet103SetSlot) ReadPacketData(r *Reader) error {
	windowID, err := r.ReadInt8()
	if err != nil {
		return err
	}
	itemSlot, err := r.ReadInt16()
	if err != nil {
		return err
	}
	stack, err := r.ReadItemStack()
	if err != nil {
		return err
	}

	p.WindowID = windowID
	p.ItemSlot = itemSlot
	p.ItemStack = stack
	return nil
}

func (p *Packet103SetSlot) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(p.WindowID); err != nil {
		return err
	}
	if err := w.WriteInt16(p.ItemSlot); err != nil {
		return err
	}
	return w.WriteItemStack(p.ItemStack)
}

func (*Packet103SetSlot) PacketSize() int {
	// Translation target: Packet103SetSlot#getPacketSize returns constant 8 in MCP.
	return 8
}

// Packet104WindowItems translates net.minecraft.src.Packet104WindowItems.
type Packet104WindowItems struct {
	WindowID   int8
	ItemStacks []*ItemStack
}

func (*Packet104WindowItems) PacketID() uint8 { return 104 }

func (p *Packet104WindowItems) ReadPacketData(r *Reader) error {
	windowID, err := r.ReadInt8()
	if err != nil {
		return err
	}
	count, err := r.ReadInt16()
	if err != nil {
		return err
	}
	if count < 0 {
		count = 0
	}

	stacks := make([]*ItemStack, int(count))
	for i := 0; i < len(stacks); i++ {
		stack, err := r.ReadItemStack()
		if err != nil {
			return err
		}
		stacks[i] = stack
	}

	p.WindowID = windowID
	p.ItemStacks = stacks
	return nil
}

func (p *Packet104WindowItems) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(p.WindowID); err != nil {
		return err
	}
	if err := w.WriteInt16(int16(len(p.ItemStacks))); err != nil {
		return err
	}
	for _, stack := range p.ItemStacks {
		if err := w.WriteItemStack(stack); err != nil {
			return err
		}
	}
	return nil
}

func (p *Packet104WindowItems) PacketSize() int {
	// Translation target: Packet104WindowItems#getPacketSize is dynamic.
	size := 3
	for _, stack := range p.ItemStacks {
		if stack == nil {
			size += 2
			continue
		}
		size += 5
		if stack.Tag != nil {
			// NBT payload size is encoded by helper; exact size is not required for current pipeline.
			size += 2
		}
	}
	return size
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented inventory packets.
	_ = Register(103, true, false, func() Packet { return &Packet103SetSlot{} })
	_ = Register(104, true, false, func() Packet { return &Packet104WindowItems{} })
}
