package protocol

// Packet24MobSpawn translates net.minecraft.src.Packet24MobSpawn.
type Packet24MobSpawn struct {
	EntityID  int32
	Type      int8
	XPosition int32
	YPosition int32
	ZPosition int32
	Yaw       int8
	Pitch     int8
	HeadYaw   int8
	VelocityX int16
	VelocityY int16
	VelocityZ int16
	Metadata  []WatchableObject
}

func (*Packet24MobSpawn) PacketID() uint8 { return 24 }

func (p *Packet24MobSpawn) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	typ, err := r.ReadInt8()
	if err != nil {
		return err
	}
	x, err := r.ReadInt32()
	if err != nil {
		return err
	}
	y, err := r.ReadInt32()
	if err != nil {
		return err
	}
	z, err := r.ReadInt32()
	if err != nil {
		return err
	}
	yaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	pitch, err := r.ReadInt8()
	if err != nil {
		return err
	}
	headYaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	velX, err := r.ReadInt16()
	if err != nil {
		return err
	}
	velY, err := r.ReadInt16()
	if err != nil {
		return err
	}
	velZ, err := r.ReadInt16()
	if err != nil {
		return err
	}
	meta, err := readWatchableObjects(r)
	if err != nil {
		return err
	}

	p.EntityID = entityID
	p.Type = typ
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	p.Yaw = yaw
	p.Pitch = pitch
	p.HeadYaw = headYaw
	p.VelocityX = velX
	p.VelocityY = velY
	p.VelocityZ = velZ
	p.Metadata = meta
	return nil
}

func (p *Packet24MobSpawn) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Type); err != nil {
		return err
	}
	if err := w.WriteInt32(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteInt32(p.YPosition); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ZPosition); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Yaw); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Pitch); err != nil {
		return err
	}
	if err := w.WriteInt8(p.HeadYaw); err != nil {
		return err
	}
	if err := w.WriteInt16(p.VelocityX); err != nil {
		return err
	}
	if err := w.WriteInt16(p.VelocityY); err != nil {
		return err
	}
	if err := w.WriteInt16(p.VelocityZ); err != nil {
		return err
	}
	return writeWatchableObjects(w, p.Metadata)
}

func (*Packet24MobSpawn) PacketSize() int {
	// Translation target: Packet24MobSpawn#getPacketSize returns 26 in MCP.
	return 26
}

func init() {
	_ = Register(24, true, false, func() Packet { return &Packet24MobSpawn{} })
}
