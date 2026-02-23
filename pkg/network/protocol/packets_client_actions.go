package protocol

// Packet7UseEntity translates net.minecraft.src.Packet7UseEntity.
type Packet7UseEntity struct {
	PlayerEntityID int32
	TargetEntityID int32
	Action         int8
}

func (*Packet7UseEntity) PacketID() uint8 { return 7 }

func (p *Packet7UseEntity) ReadPacketData(r *Reader) error {
	playerID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	targetID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	action, err := r.ReadInt8()
	if err != nil {
		return err
	}

	p.PlayerEntityID = playerID
	p.TargetEntityID = targetID
	p.Action = action
	return nil
}

func (p *Packet7UseEntity) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.PlayerEntityID); err != nil {
		return err
	}
	if err := w.WriteInt32(p.TargetEntityID); err != nil {
		return err
	}
	return w.WriteInt8(p.Action)
}

func (*Packet7UseEntity) PacketSize() int { return 9 }

// Packet18Animation translates net.minecraft.src.Packet18Animation.
type Packet18Animation struct {
	EntityID  int32
	AnimateID int8
}

func (*Packet18Animation) PacketID() uint8 { return 18 }

func (p *Packet18Animation) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	animateID, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.AnimateID = animateID
	return nil
}

func (p *Packet18Animation) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	return w.WriteInt8(p.AnimateID)
}

func (*Packet18Animation) PacketSize() int { return 5 }

// Packet19EntityAction translates net.minecraft.src.Packet19EntityAction.
type Packet19EntityAction struct {
	EntityID int32
	Action   int8
	AuxData  int32
}

func (*Packet19EntityAction) PacketID() uint8 { return 19 }

func (p *Packet19EntityAction) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	action, err := r.ReadInt8()
	if err != nil {
		return err
	}
	auxData, err := r.ReadInt32()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.Action = action
	p.AuxData = auxData
	return nil
}

func (p *Packet19EntityAction) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Action); err != nil {
		return err
	}
	return w.WriteInt32(p.AuxData)
}

func (*Packet19EntityAction) PacketSize() int { return 9 }

// Packet27PlayerInput translates net.minecraft.src.Packet27PlayerInput.
type Packet27PlayerInput struct {
	MoveStrafing float32
	MoveForward  float32
	Jump         bool
	Sneak        bool
}

func (*Packet27PlayerInput) PacketID() uint8 { return 27 }

func (p *Packet27PlayerInput) ReadPacketData(r *Reader) error {
	moveStrafing, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	moveForward, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	jumpRaw, err := r.ReadUint8()
	if err != nil {
		return err
	}
	sneakRaw, err := r.ReadUint8()
	if err != nil {
		return err
	}
	p.MoveStrafing = moveStrafing
	p.MoveForward = moveForward
	p.Jump = jumpRaw != 0
	p.Sneak = sneakRaw != 0
	return nil
}

func (p *Packet27PlayerInput) WritePacketData(w *Writer) error {
	if err := w.WriteFloat32(p.MoveStrafing); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.MoveForward); err != nil {
		return err
	}
	var jump uint8
	if p.Jump {
		jump = 1
	}
	if err := w.WriteUint8(jump); err != nil {
		return err
	}
	var sneak uint8
	if p.Sneak {
		sneak = 1
	}
	return w.WriteUint8(sneak)
}

func (*Packet27PlayerInput) PacketSize() int { return 10 }

func init() {
	// Translation target: Packet#addIdClassMapping for implemented client action packets.
	_ = Register(7, false, true, func() Packet { return &Packet7UseEntity{} })
	_ = Register(18, true, true, func() Packet { return &Packet18Animation{} })
	_ = Register(19, false, true, func() Packet { return &Packet19EntityAction{} })
	_ = Register(27, false, true, func() Packet { return &Packet27PlayerInput{} })
}
