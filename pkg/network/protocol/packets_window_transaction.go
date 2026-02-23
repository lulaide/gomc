package protocol

// Packet101CloseWindow translates net.minecraft.src.Packet101CloseWindow.
type Packet101CloseWindow struct {
	WindowID int8
}

func (*Packet101CloseWindow) PacketID() uint8 { return 101 }

func (p *Packet101CloseWindow) ReadPacketData(r *Reader) error {
	id, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.WindowID = id
	return nil
}

func (p *Packet101CloseWindow) WritePacketData(w *Writer) error {
	return w.WriteInt8(p.WindowID)
}

func (*Packet101CloseWindow) PacketSize() int { return 1 }

// Packet102WindowClick translates net.minecraft.src.Packet102WindowClick.
type Packet102WindowClick struct {
	WindowID      int8
	InventorySlot int16
	MouseClick    int8
	ActionNumber  int16
	// Mode maps to vanilla click mode byte:
	// 0=normal, 1=shift, 2=hotbar swap, 3=pick block, 4=drop, 5=drag, 6=double click.
	Mode         int8
	HoldingShift bool
	ItemStack    *ItemStack
}

func (*Packet102WindowClick) PacketID() uint8 { return 102 }

func (p *Packet102WindowClick) ReadPacketData(r *Reader) error {
	windowID, err := r.ReadInt8()
	if err != nil {
		return err
	}
	slot, err := r.ReadInt16()
	if err != nil {
		return err
	}
	mouse, err := r.ReadInt8()
	if err != nil {
		return err
	}
	action, err := r.ReadInt16()
	if err != nil {
		return err
	}
	mode, err := r.ReadInt8()
	if err != nil {
		return err
	}
	stack, err := r.ReadItemStack()
	if err != nil {
		return err
	}

	p.WindowID = windowID
	p.InventorySlot = slot
	p.MouseClick = mouse
	p.ActionNumber = action
	p.Mode = mode
	p.HoldingShift = mode == 1
	p.ItemStack = stack
	return nil
}

func (p *Packet102WindowClick) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(p.WindowID); err != nil {
		return err
	}
	if err := w.WriteInt16(p.InventorySlot); err != nil {
		return err
	}
	if err := w.WriteInt8(p.MouseClick); err != nil {
		return err
	}
	if err := w.WriteInt16(p.ActionNumber); err != nil {
		return err
	}
	mode := p.Mode
	if mode == 0 && p.HoldingShift {
		mode = 1
	}
	if err := w.WriteInt8(mode); err != nil {
		return err
	}
	return w.WriteItemStack(p.ItemStack)
}

func (*Packet102WindowClick) PacketSize() int {
	// Translation target: Packet102WindowClick#getPacketSize returns constant 11 in MCP.
	return 11
}

// Packet106Transaction translates net.minecraft.src.Packet106Transaction.
type Packet106Transaction struct {
	WindowID     int8
	ActionNumber int16
	Accepted     bool
}

func (*Packet106Transaction) PacketID() uint8 { return 106 }

func (p *Packet106Transaction) ReadPacketData(r *Reader) error {
	windowID, err := r.ReadInt8()
	if err != nil {
		return err
	}
	action, err := r.ReadInt16()
	if err != nil {
		return err
	}
	acceptedByte, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.WindowID = windowID
	p.ActionNumber = action
	p.Accepted = acceptedByte != 0
	return nil
}

func (p *Packet106Transaction) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(p.WindowID); err != nil {
		return err
	}
	if err := w.WriteInt16(p.ActionNumber); err != nil {
		return err
	}
	if p.Accepted {
		return w.WriteUint8(1)
	}
	return w.WriteUint8(0)
}

func (*Packet106Transaction) PacketSize() int { return 4 }

func init() {
	// Translation target: Packet#addIdClassMapping for implemented inventory/window transaction packets.
	_ = Register(101, true, true, func() Packet { return &Packet101CloseWindow{} })
	_ = Register(102, false, true, func() Packet { return &Packet102WindowClick{} })
	_ = Register(106, true, true, func() Packet { return &Packet106Transaction{} })
}
