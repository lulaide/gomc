package protocol

// NewPacket3Chat mirrors Packet3Chat(String, boolean) behavior.
func NewPacket3Chat(message string, isServer bool) *Packet3Chat {
	if len(message) > 32767 {
		message = message[:32767]
	}
	return &Packet3Chat{
		Message:  message,
		IsServer: isServer,
	}
}

// Packet3Chat translates net.minecraft.src.Packet3Chat.
type Packet3Chat struct {
	Message  string
	IsServer bool
}

func (*Packet3Chat) PacketID() uint8 { return 3 }

func (p *Packet3Chat) ReadPacketData(r *Reader) error {
	message, err := r.ReadString(32767)
	if err != nil {
		return err
	}
	p.Message = message
	p.IsServer = true
	return nil
}

func (p *Packet3Chat) WritePacketData(w *Writer) error {
	return w.WriteString(p.Message)
}

func (p *Packet3Chat) PacketSize() int {
	return 2 + 2*utf16UnitsLen(p.Message)
}

// NewPacket4UpdateTime mirrors Packet4UpdateTime(long,long,boolean).
func NewPacket4UpdateTime(worldAge int64, timeOfDay int64, doDaylightCycle bool) *Packet4UpdateTime {
	p := &Packet4UpdateTime{
		WorldAge: worldAge,
		Time:     timeOfDay,
	}

	if !doDaylightCycle {
		p.Time = -p.Time
		if p.Time == 0 {
			p.Time = -1
		}
	}

	return p
}

// Packet4UpdateTime translates net.minecraft.src.Packet4UpdateTime.
type Packet4UpdateTime struct {
	WorldAge int64
	Time     int64
}

func (*Packet4UpdateTime) PacketID() uint8 { return 4 }

func (p *Packet4UpdateTime) ReadPacketData(r *Reader) error {
	worldAge, err := r.ReadInt64()
	if err != nil {
		return err
	}
	time, err := r.ReadInt64()
	if err != nil {
		return err
	}
	p.WorldAge = worldAge
	p.Time = time
	return nil
}

func (p *Packet4UpdateTime) WritePacketData(w *Writer) error {
	if err := w.WriteInt64(p.WorldAge); err != nil {
		return err
	}
	return w.WriteInt64(p.Time)
}

func (*Packet4UpdateTime) PacketSize() int { return 16 }

// Packet6SpawnPosition translates net.minecraft.src.Packet6SpawnPosition.
type Packet6SpawnPosition struct {
	XPosition int32
	YPosition int32
	ZPosition int32
}

func (*Packet6SpawnPosition) PacketID() uint8 { return 6 }

func (p *Packet6SpawnPosition) ReadPacketData(r *Reader) error {
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
	p.XPosition = x
	p.YPosition = y
	p.ZPosition = z
	return nil
}

func (p *Packet6SpawnPosition) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteInt32(p.YPosition); err != nil {
		return err
	}
	return w.WriteInt32(p.ZPosition)
}

func (*Packet6SpawnPosition) PacketSize() int { return 12 }

// Packet8UpdateHealth translates net.minecraft.src.Packet8UpdateHealth.
type Packet8UpdateHealth struct {
	HealthMP       float32
	Food           int16
	FoodSaturation float32
}

func (*Packet8UpdateHealth) PacketID() uint8 { return 8 }

func (p *Packet8UpdateHealth) ReadPacketData(r *Reader) error {
	health, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	food, err := r.ReadInt16()
	if err != nil {
		return err
	}
	saturation, err := r.ReadFloat32()
	if err != nil {
		return err
	}

	p.HealthMP = health
	p.Food = food
	p.FoodSaturation = saturation
	return nil
}

func (p *Packet8UpdateHealth) WritePacketData(w *Writer) error {
	if err := w.WriteFloat32(p.HealthMP); err != nil {
		return err
	}
	if err := w.WriteInt16(p.Food); err != nil {
		return err
	}
	return w.WriteFloat32(p.FoodSaturation)
}

func (*Packet8UpdateHealth) PacketSize() int { return 8 }

// Packet9Respawn translates net.minecraft.src.Packet9Respawn.
type Packet9Respawn struct {
	RespawnDimension int32
	Difficulty       int8
	WorldHeight      int16
	GameType         int8
	TerrainType      string
}

func (*Packet9Respawn) PacketID() uint8 { return 9 }

func (p *Packet9Respawn) ReadPacketData(r *Reader) error {
	respawnDim, err := r.ReadInt32()
	if err != nil {
		return err
	}
	difficulty, err := r.ReadInt8()
	if err != nil {
		return err
	}
	gameTypeRaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	worldHeight, err := r.ReadInt16()
	if err != nil {
		return err
	}
	terrain, err := r.ReadString(16)
	if err != nil {
		return err
	}

	p.RespawnDimension = respawnDim
	p.Difficulty = difficulty
	p.GameType = normalizeGameTypeID(int(gameTypeRaw))
	p.WorldHeight = worldHeight
	p.TerrainType = normalizeWorldTypeName(terrain)
	return nil
}

func (p *Packet9Respawn) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.RespawnDimension); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Difficulty); err != nil {
		return err
	}
	if err := w.WriteInt8(p.GameType); err != nil {
		return err
	}
	if err := w.WriteInt16(p.WorldHeight); err != nil {
		return err
	}
	return w.WriteString(p.TerrainType)
}

func (p *Packet9Respawn) PacketSize() int {
	// Translation target: Packet9Respawn#getPacketSize uses UTF-16 char count, not bytes.
	return 8 + utf16UnitsLen(p.TerrainType)
}

// Packet10Flying translates net.minecraft.src.Packet10Flying.
type Packet10Flying struct {
	XPosition float64
	YPosition float64
	ZPosition float64
	Stance    float64
	Yaw       float32
	Pitch     float32

	OnGround bool
	Moving   bool
	Rotating bool
}

func (*Packet10Flying) PacketID() uint8 { return 10 }

func (p *Packet10Flying) ReadPacketData(r *Reader) error {
	onGround, err := r.ReadUint8()
	if err != nil {
		return err
	}
	p.OnGround = onGround != 0
	return nil
}

func (p *Packet10Flying) WritePacketData(w *Writer) error {
	if p.OnGround {
		return w.WriteUint8(1)
	}
	return w.WriteUint8(0)
}

func (*Packet10Flying) PacketSize() int { return 1 }

// Packet11PlayerPosition translates net.minecraft.src.Packet11PlayerPosition.
type Packet11PlayerPosition struct {
	Packet10Flying
}

func NewPacket11PlayerPosition() *Packet11PlayerPosition {
	p := &Packet11PlayerPosition{}
	p.Moving = true
	return p
}

func (*Packet11PlayerPosition) PacketID() uint8 { return 11 }

func (p *Packet11PlayerPosition) ReadPacketData(r *Reader) error {
	x, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	y, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	stance, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	z, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	p.XPosition = x
	p.YPosition = y
	p.Stance = stance
	p.ZPosition = z
	p.Moving = true
	return p.Packet10Flying.ReadPacketData(r)
}

func (p *Packet11PlayerPosition) WritePacketData(w *Writer) error {
	if err := w.WriteFloat64(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.YPosition); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.Stance); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.ZPosition); err != nil {
		return err
	}
	return p.Packet10Flying.WritePacketData(w)
}

func (*Packet11PlayerPosition) PacketSize() int { return 33 }

// Packet12PlayerLook translates net.minecraft.src.Packet12PlayerLook.
type Packet12PlayerLook struct {
	Packet10Flying
}

func NewPacket12PlayerLook() *Packet12PlayerLook {
	p := &Packet12PlayerLook{}
	p.Rotating = true
	return p
}

func (*Packet12PlayerLook) PacketID() uint8 { return 12 }

func (p *Packet12PlayerLook) ReadPacketData(r *Reader) error {
	yaw, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	pitch, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	p.Yaw = yaw
	p.Pitch = pitch
	p.Rotating = true
	return p.Packet10Flying.ReadPacketData(r)
}

func (p *Packet12PlayerLook) WritePacketData(w *Writer) error {
	if err := w.WriteFloat32(p.Yaw); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.Pitch); err != nil {
		return err
	}
	return p.Packet10Flying.WritePacketData(w)
}

func (*Packet12PlayerLook) PacketSize() int { return 9 }

// Packet13PlayerLookMove translates net.minecraft.src.Packet13PlayerLookMove.
type Packet13PlayerLookMove struct {
	Packet10Flying
}

func NewPacket13PlayerLookMove() *Packet13PlayerLookMove {
	p := &Packet13PlayerLookMove{}
	p.Moving = true
	p.Rotating = true
	return p
}

func (*Packet13PlayerLookMove) PacketID() uint8 { return 13 }

func (p *Packet13PlayerLookMove) ReadPacketData(r *Reader) error {
	x, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	y, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	stance, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	z, err := r.ReadFloat64()
	if err != nil {
		return err
	}
	yaw, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	pitch, err := r.ReadFloat32()
	if err != nil {
		return err
	}

	p.XPosition = x
	p.YPosition = y
	p.Stance = stance
	p.ZPosition = z
	p.Yaw = yaw
	p.Pitch = pitch
	p.Moving = true
	p.Rotating = true
	return p.Packet10Flying.ReadPacketData(r)
}

func (p *Packet13PlayerLookMove) WritePacketData(w *Writer) error {
	if err := w.WriteFloat64(p.XPosition); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.YPosition); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.Stance); err != nil {
		return err
	}
	if err := w.WriteFloat64(p.ZPosition); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.Yaw); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.Pitch); err != nil {
		return err
	}
	return p.Packet10Flying.WritePacketData(w)
}

func (*Packet13PlayerLookMove) PacketSize() int { return 41 }

// Packet202PlayerAbilities translates net.minecraft.src.Packet202PlayerAbilities.
type Packet202PlayerAbilities struct {
	DisableDamage bool
	IsFlying      bool
	AllowFlying   bool
	IsCreative    bool
	FlySpeed      float32
	WalkSpeed     float32
}

func (*Packet202PlayerAbilities) PacketID() uint8 { return 202 }

func (p *Packet202PlayerAbilities) ReadPacketData(r *Reader) error {
	flags, err := r.ReadInt8()
	if err != nil {
		return err
	}
	fly, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	walk, err := r.ReadFloat32()
	if err != nil {
		return err
	}

	p.DisableDamage = (flags & 1) > 0
	p.IsFlying = (flags & 2) > 0
	p.AllowFlying = (flags & 4) > 0
	p.IsCreative = (flags & 8) > 0
	p.FlySpeed = fly
	p.WalkSpeed = walk
	return nil
}

func (p *Packet202PlayerAbilities) WritePacketData(w *Writer) error {
	var flags int8 = 0
	if p.DisableDamage {
		flags |= 1
	}
	if p.IsFlying {
		flags |= 2
	}
	if p.AllowFlying {
		flags |= 4
	}
	if p.IsCreative {
		flags |= 8
	}

	if err := w.WriteInt8(flags); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.FlySpeed); err != nil {
		return err
	}
	return w.WriteFloat32(p.WalkSpeed)
}

func (*Packet202PlayerAbilities) PacketSize() int {
	// Translation target: Packet202PlayerAbilities#getPacketSize() returns 2 in MCP.
	return 2
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented packet subset.
	_ = Register(3, true, true, func() Packet { return &Packet3Chat{IsServer: true} })
	_ = Register(4, true, false, func() Packet { return &Packet4UpdateTime{} })
	_ = Register(6, true, false, func() Packet { return &Packet6SpawnPosition{} })
	_ = Register(8, true, false, func() Packet { return &Packet8UpdateHealth{} })
	_ = Register(9, true, true, func() Packet { return &Packet9Respawn{} })
	_ = Register(10, true, true, func() Packet { return &Packet10Flying{} })
	_ = Register(11, true, true, func() Packet { return NewPacket11PlayerPosition() })
	_ = Register(12, true, true, func() Packet { return NewPacket12PlayerLook() })
	_ = Register(13, true, true, func() Packet { return NewPacket13PlayerLookMove() })
	_ = Register(202, true, true, func() Packet { return &Packet202PlayerAbilities{} })
}
