package protocol

import (
	"fmt"
	"math"
)

// WatchableObject mirrors DataWatcher serialized entries used by entity packets.
//
// Translation target:
// - net.minecraft.src.DataWatcher#writeWatchableObject
// - net.minecraft.src.DataWatcher#readWatchableObjects
type WatchableObject struct {
	ObjectType  int8
	DataValueID int8
	Value       any
}

func readWatchableObjects(r *Reader) ([]WatchableObject, error) {
	var out []WatchableObject

	for {
		header, err := r.ReadInt8()
		if err != nil {
			return nil, err
		}
		if header == 127 {
			return out, nil
		}

		headerByte := uint8(header)
		objectType := int8((headerByte & 224) >> 5)
		dataValueID := int8(headerByte & 31)
		entry := WatchableObject{
			ObjectType:  objectType,
			DataValueID: dataValueID,
		}

		switch objectType {
		case 0:
			v, err := r.ReadInt8()
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 1:
			v, err := r.ReadInt16()
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 2:
			v, err := r.ReadInt32()
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 3:
			v, err := r.ReadFloat32()
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 4:
			v, err := r.ReadString(64)
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 5:
			v, err := r.ReadItemStack()
			if err != nil {
				return nil, err
			}
			entry.Value = v
		case 6:
			x, err := r.ReadInt32()
			if err != nil {
				return nil, err
			}
			y, err := r.ReadInt32()
			if err != nil {
				return nil, err
			}
			z, err := r.ReadInt32()
			if err != nil {
				return nil, err
			}
			entry.Value = [3]int32{x, y, z}
		default:
			return nil, fmt.Errorf("unknown watchable object type: %d", objectType)
		}

		out = append(out, entry)
	}
}

func writeWatchableObjects(w *Writer, objects []WatchableObject) error {
	for _, obj := range objects {
		header := uint8((obj.ObjectType << 5) | (obj.DataValueID & 31))
		if err := w.WriteInt8(int8(header)); err != nil {
			return err
		}

		switch obj.ObjectType {
		case 0:
			v, ok := obj.Value.(int8)
			if !ok {
				return fmt.Errorf("watchable type 0 expects int8, got %T", obj.Value)
			}
			if err := w.WriteInt8(v); err != nil {
				return err
			}
		case 1:
			v, ok := obj.Value.(int16)
			if !ok {
				return fmt.Errorf("watchable type 1 expects int16, got %T", obj.Value)
			}
			if err := w.WriteInt16(v); err != nil {
				return err
			}
		case 2:
			v, ok := obj.Value.(int32)
			if !ok {
				return fmt.Errorf("watchable type 2 expects int32, got %T", obj.Value)
			}
			if err := w.WriteInt32(v); err != nil {
				return err
			}
		case 3:
			v, ok := obj.Value.(float32)
			if !ok {
				return fmt.Errorf("watchable type 3 expects float32, got %T", obj.Value)
			}
			if err := w.WriteFloat32(v); err != nil {
				return err
			}
		case 4:
			v, ok := obj.Value.(string)
			if !ok {
				return fmt.Errorf("watchable type 4 expects string, got %T", obj.Value)
			}
			if err := w.WriteString(v); err != nil {
				return err
			}
		case 5:
			v, ok := obj.Value.(*ItemStack)
			if !ok {
				return fmt.Errorf("watchable type 5 expects *ItemStack, got %T", obj.Value)
			}
			if err := w.WriteItemStack(v); err != nil {
				return err
			}
		case 6:
			v, ok := obj.Value.([3]int32)
			if !ok {
				return fmt.Errorf("watchable type 6 expects [3]int32, got %T", obj.Value)
			}
			if err := w.WriteInt32(v[0]); err != nil {
				return err
			}
			if err := w.WriteInt32(v[1]); err != nil {
				return err
			}
			if err := w.WriteInt32(v[2]); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown watchable object type: %d", obj.ObjectType)
		}
	}

	return w.WriteInt8(127)
}

// Packet20NamedEntitySpawn translates net.minecraft.src.Packet20NamedEntitySpawn.
type Packet20NamedEntitySpawn struct {
	EntityID    int32
	Name        string
	XPosition   int32
	YPosition   int32
	ZPosition   int32
	Rotation    int8
	Pitch       int8
	CurrentItem int16
	Metadata    []WatchableObject
}

func (*Packet20NamedEntitySpawn) PacketID() uint8 { return 20 }

func (p *Packet20NamedEntitySpawn) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	name, err := r.ReadString(16)
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
	rotation, err := r.ReadInt8()
	if err != nil {
		return err
	}
	pitch, err := r.ReadInt8()
	if err != nil {
		return err
	}
	currentItem, err := r.ReadInt16()
	if err != nil {
		return err
	}
	metadata, err := readWatchableObjects(r)
	if err != nil {
		return err
	}

	p.EntityID = entityID
	p.Name = name
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	p.Rotation = rotation
	p.Pitch = pitch
	p.CurrentItem = currentItem
	p.Metadata = metadata
	return nil
}

func (p *Packet20NamedEntitySpawn) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	if err := w.WriteString(p.Name); err != nil {
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
	if err := w.WriteInt8(p.Rotation); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Pitch); err != nil {
		return err
	}
	if err := w.WriteInt16(p.CurrentItem); err != nil {
		return err
	}
	return writeWatchableObjects(w, p.Metadata)
}

func (*Packet20NamedEntitySpawn) PacketSize() int {
	// Translation target: Packet20NamedEntitySpawn#getPacketSize returns constant 28.
	return 28
}

// Packet23VehicleSpawn translates net.minecraft.src.Packet23VehicleSpawn.
type Packet23VehicleSpawn struct {
	EntityID int32
	Type     int8

	XPosition int32
	YPosition int32
	ZPosition int32
	Pitch     int8
	Yaw       int8

	ThrowerEntityID int32
	SpeedX          int16
	SpeedY          int16
	SpeedZ          int16
}

func (*Packet23VehicleSpawn) PacketID() uint8 { return 23 }

func (p *Packet23VehicleSpawn) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	entityType, err := r.ReadInt8()
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
	pitch, err := r.ReadInt8()
	if err != nil {
		return err
	}
	yaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	throwerEntityID, err := r.ReadInt32()
	if err != nil {
		return err
	}

	var speedX int16
	var speedY int16
	var speedZ int16
	if throwerEntityID > 0 {
		speedX, err = r.ReadInt16()
		if err != nil {
			return err
		}
		speedY, err = r.ReadInt16()
		if err != nil {
			return err
		}
		speedZ, err = r.ReadInt16()
		if err != nil {
			return err
		}
	}

	p.EntityID = entityID
	p.Type = entityType
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	p.Pitch = pitch
	p.Yaw = yaw
	p.ThrowerEntityID = throwerEntityID
	p.SpeedX = speedX
	p.SpeedY = speedY
	p.SpeedZ = speedZ
	return nil
}

func (p *Packet23VehicleSpawn) WritePacketData(w *Writer) error {
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
	if err := w.WriteInt8(p.Pitch); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Yaw); err != nil {
		return err
	}
	if err := w.WriteInt32(p.ThrowerEntityID); err != nil {
		return err
	}
	if p.ThrowerEntityID > 0 {
		if err := w.WriteInt16(p.SpeedX); err != nil {
			return err
		}
		if err := w.WriteInt16(p.SpeedY); err != nil {
			return err
		}
		if err := w.WriteInt16(p.SpeedZ); err != nil {
			return err
		}
	}
	return nil
}

func (p *Packet23VehicleSpawn) PacketSize() int {
	if p.ThrowerEntityID > 0 {
		return 27
	}
	return 21
}

// Packet28EntityVelocity translates net.minecraft.src.Packet28EntityVelocity.
type Packet28EntityVelocity struct {
	EntityID int32
	MotionX  int16
	MotionY  int16
	MotionZ  int16
}

func NewPacket28EntityVelocity(entityID int32, motionX, motionY, motionZ float64) *Packet28EntityVelocity {
	const maxVelocity = 3.9
	clamp := func(v float64) float64 {
		return math.Max(-maxVelocity, math.Min(maxVelocity, v))
	}
	return &Packet28EntityVelocity{
		EntityID: entityID,
		MotionX:  int16(clamp(motionX) * 8000.0),
		MotionY:  int16(clamp(motionY) * 8000.0),
		MotionZ:  int16(clamp(motionZ) * 8000.0),
	}
}

func (*Packet28EntityVelocity) PacketID() uint8 { return 28 }

func (p *Packet28EntityVelocity) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	motionX, err := r.ReadInt16()
	if err != nil {
		return err
	}
	motionY, err := r.ReadInt16()
	if err != nil {
		return err
	}
	motionZ, err := r.ReadInt16()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.MotionX = motionX
	p.MotionY = motionY
	p.MotionZ = motionZ
	return nil
}

func (p *Packet28EntityVelocity) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	if err := w.WriteInt16(p.MotionX); err != nil {
		return err
	}
	if err := w.WriteInt16(p.MotionY); err != nil {
		return err
	}
	return w.WriteInt16(p.MotionZ)
}

func (*Packet28EntityVelocity) PacketSize() int { return 10 }

// Packet29DestroyEntity translates net.minecraft.src.Packet29DestroyEntity.
type Packet29DestroyEntity struct {
	EntityIDs []int32
}

func (*Packet29DestroyEntity) PacketID() uint8 { return 29 }

func (p *Packet29DestroyEntity) ReadPacketData(r *Reader) error {
	count, err := r.ReadInt8()
	if err != nil {
		return err
	}
	if count < 0 {
		return fmt.Errorf("invalid destroy entity count: %d", count)
	}

	ids := make([]int32, int(count))
	for i := 0; i < len(ids); i++ {
		v, err := r.ReadInt32()
		if err != nil {
			return err
		}
		ids[i] = v
	}
	p.EntityIDs = ids
	return nil
}

func (p *Packet29DestroyEntity) WritePacketData(w *Writer) error {
	if len(p.EntityIDs) > 127 {
		return fmt.Errorf("too many entity ids in destroy packet: %d", len(p.EntityIDs))
	}
	if err := w.WriteInt8(int8(len(p.EntityIDs))); err != nil {
		return err
	}
	for _, id := range p.EntityIDs {
		if err := w.WriteInt32(id); err != nil {
			return err
		}
	}
	return nil
}

func (p *Packet29DestroyEntity) PacketSize() int {
	return 1 + len(p.EntityIDs)*4
}

// Packet30Entity translates net.minecraft.src.Packet30Entity.
type Packet30Entity struct {
	EntityID int32

	XPosition int8
	YPosition int8
	ZPosition int8
	Yaw       int8
	Pitch     int8

	Rotating bool
}

func (*Packet30Entity) PacketID() uint8 { return 30 }

func (p *Packet30Entity) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	return nil
}

func (p *Packet30Entity) WritePacketData(w *Writer) error {
	return w.WriteInt32(p.EntityID)
}

func (*Packet30Entity) PacketSize() int { return 4 }

// Packet31RelEntityMove translates net.minecraft.src.Packet31RelEntityMove.
type Packet31RelEntityMove struct {
	Packet30Entity
}

func (*Packet31RelEntityMove) PacketID() uint8 { return 31 }

func (p *Packet31RelEntityMove) ReadPacketData(r *Reader) error {
	if err := p.Packet30Entity.ReadPacketData(r); err != nil {
		return err
	}
	x, err := r.ReadInt8()
	if err != nil {
		return err
	}
	y, err := r.ReadInt8()
	if err != nil {
		return err
	}
	z, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	return nil
}

func (p *Packet31RelEntityMove) WritePacketData(w *Writer) error {
	if err := p.Packet30Entity.WritePacketData(w); err != nil {
		return err
	}
	if err := w.WriteInt8(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteInt8(p.YPosition); err != nil {
		return err
	}
	return w.WriteInt8(p.ZPosition)
}

func (*Packet31RelEntityMove) PacketSize() int { return 7 }

// Packet32EntityLook translates net.minecraft.src.Packet32EntityLook.
type Packet32EntityLook struct {
	Packet30Entity
}

func NewPacket32EntityLook() *Packet32EntityLook {
	return &Packet32EntityLook{
		Packet30Entity: Packet30Entity{Rotating: true},
	}
}

func (*Packet32EntityLook) PacketID() uint8 { return 32 }

func (p *Packet32EntityLook) ReadPacketData(r *Reader) error {
	if err := p.Packet30Entity.ReadPacketData(r); err != nil {
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
	p.Yaw = yaw
	p.Pitch = pitch
	p.Rotating = true
	return nil
}

func (p *Packet32EntityLook) WritePacketData(w *Writer) error {
	if err := p.Packet30Entity.WritePacketData(w); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Yaw); err != nil {
		return err
	}
	return w.WriteInt8(p.Pitch)
}

func (*Packet32EntityLook) PacketSize() int { return 6 }

// Packet33RelEntityMoveLook translates net.minecraft.src.Packet33RelEntityMoveLook.
type Packet33RelEntityMoveLook struct {
	Packet30Entity
}

func NewPacket33RelEntityMoveLook() *Packet33RelEntityMoveLook {
	return &Packet33RelEntityMoveLook{
		Packet30Entity: Packet30Entity{Rotating: true},
	}
}

func (*Packet33RelEntityMoveLook) PacketID() uint8 { return 33 }

func (p *Packet33RelEntityMoveLook) ReadPacketData(r *Reader) error {
	if err := p.Packet30Entity.ReadPacketData(r); err != nil {
		return err
	}
	x, err := r.ReadInt8()
	if err != nil {
		return err
	}
	y, err := r.ReadInt8()
	if err != nil {
		return err
	}
	z, err := r.ReadInt8()
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

	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	p.Yaw = yaw
	p.Pitch = pitch
	p.Rotating = true
	return nil
}

func (p *Packet33RelEntityMoveLook) WritePacketData(w *Writer) error {
	if err := p.Packet30Entity.WritePacketData(w); err != nil {
		return err
	}
	if err := w.WriteInt8(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteInt8(p.YPosition); err != nil {
		return err
	}
	if err := w.WriteInt8(p.ZPosition); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Yaw); err != nil {
		return err
	}
	return w.WriteInt8(p.Pitch)
}

func (*Packet33RelEntityMoveLook) PacketSize() int { return 9 }

// Packet34EntityTeleport translates net.minecraft.src.Packet34EntityTeleport.
type Packet34EntityTeleport struct {
	EntityID  int32
	XPosition int32
	YPosition int32
	ZPosition int32
	Yaw       int8
	Pitch     int8
}

func (*Packet34EntityTeleport) PacketID() uint8 { return 34 }

func (p *Packet34EntityTeleport) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
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

	p.EntityID = entityID
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	p.Yaw = yaw
	p.Pitch = pitch
	return nil
}

func (p *Packet34EntityTeleport) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
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
	return w.WriteInt8(p.Pitch)
}

func (*Packet34EntityTeleport) PacketSize() int {
	// Translation target: Packet34EntityTeleport#getPacketSize returns 34 in MCP.
	return 34
}

// Packet35EntityHeadRotation translates net.minecraft.src.Packet35EntityHeadRotation.
type Packet35EntityHeadRotation struct {
	EntityID        int32
	HeadRotationYaw int8
}

func (*Packet35EntityHeadRotation) PacketID() uint8 { return 35 }

func (p *Packet35EntityHeadRotation) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	headYaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.HeadRotationYaw = headYaw
	return nil
}

func (p *Packet35EntityHeadRotation) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	return w.WriteInt8(p.HeadRotationYaw)
}

func (*Packet35EntityHeadRotation) PacketSize() int { return 5 }

func init() {
	// Translation target: Packet#addIdClassMapping for implemented entity sync packets.
	_ = Register(20, true, false, func() Packet { return &Packet20NamedEntitySpawn{} })
	_ = Register(23, true, false, func() Packet { return &Packet23VehicleSpawn{} })
	_ = Register(28, true, false, func() Packet { return &Packet28EntityVelocity{} })
	_ = Register(29, true, false, func() Packet { return &Packet29DestroyEntity{} })
	_ = Register(30, true, false, func() Packet { return &Packet30Entity{} })
	_ = Register(31, true, false, func() Packet { return &Packet31RelEntityMove{} })
	_ = Register(32, true, false, func() Packet { return NewPacket32EntityLook() })
	_ = Register(33, true, false, func() Packet { return NewPacket33RelEntityMoveLook() })
	_ = Register(34, true, false, func() Packet { return &Packet34EntityTeleport{} })
	_ = Register(35, true, false, func() Packet { return &Packet35EntityHeadRotation{} })
}
