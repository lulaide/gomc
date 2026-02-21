package protocol

import "unicode/utf16"

func utf16UnitsLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func normalizeGameTypeID(id int) int8 {
	switch id {
	case 0, 1, 2:
		return int8(id)
	default:
		return 0
	}
}

func normalizeWorldTypeName(name string) string {
	switch name {
	case "default", "flat", "largeBiomes", "default_1_1":
		return name
	default:
		return "default"
	}
}

// Packet0KeepAlive translates net.minecraft.src.Packet0KeepAlive.
type Packet0KeepAlive struct {
	RandomID int32
}

func (*Packet0KeepAlive) PacketID() uint8 { return 0 }

func (p *Packet0KeepAlive) ReadPacketData(r *Reader) error {
	v, err := r.ReadInt32()
	if err != nil {
		return err
	}
	p.RandomID = v
	return nil
}

func (p *Packet0KeepAlive) WritePacketData(w *Writer) error {
	return w.WriteInt32(p.RandomID)
}

func (*Packet0KeepAlive) PacketSize() int { return 4 }

// Packet1Login translates net.minecraft.src.Packet1Login.
type Packet1Login struct {
	ClientEntityID    int32
	TerrainType       string
	HardcoreMode      bool
	GameType          int8
	Dimension         int8
	DifficultySetting int8
	WorldHeight       int8
	MaxPlayers        int8
}

func (*Packet1Login) PacketID() uint8 { return 1 }

func (p *Packet1Login) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	terrain, err := r.ReadString(16)
	if err != nil {
		return err
	}
	modeRaw, err := r.ReadInt8()
	if err != nil {
		return err
	}
	dimension, err := r.ReadInt8()
	if err != nil {
		return err
	}
	difficulty, err := r.ReadInt8()
	if err != nil {
		return err
	}
	worldHeight, err := r.ReadInt8()
	if err != nil {
		return err
	}
	maxPlayers, err := r.ReadInt8()
	if err != nil {
		return err
	}

	mode := int(modeRaw)
	p.ClientEntityID = entityID
	p.TerrainType = normalizeWorldTypeName(terrain)
	p.HardcoreMode = (mode & 8) == 8
	p.GameType = normalizeGameTypeID(mode &^ 8)
	p.Dimension = dimension
	p.DifficultySetting = difficulty
	p.WorldHeight = worldHeight
	p.MaxPlayers = maxPlayers
	return nil
}

func (p *Packet1Login) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.ClientEntityID); err != nil {
		return err
	}

	terrain := p.TerrainType
	if terrain == "" {
		terrain = ""
	}
	if err := w.WriteString(terrain); err != nil {
		return err
	}

	mode := int(p.GameType)
	if p.HardcoreMode {
		mode |= 8
	}

	if err := w.WriteInt8(int8(mode)); err != nil {
		return err
	}
	if err := w.WriteInt8(p.Dimension); err != nil {
		return err
	}
	if err := w.WriteInt8(p.DifficultySetting); err != nil {
		return err
	}
	if err := w.WriteInt8(p.WorldHeight); err != nil {
		return err
	}
	return w.WriteInt8(p.MaxPlayers)
}

func (p *Packet1Login) PacketSize() int {
	terrainLen := 0
	if p.TerrainType != "" {
		terrainLen = utf16UnitsLen(p.TerrainType)
	}
	return 6 + 2*terrainLen + 4 + 4 + 1 + 1 + 1
}

// Packet2ClientProtocol translates net.minecraft.src.Packet2ClientProtocol.
type Packet2ClientProtocol struct {
	ProtocolVersion int8
	Username        string
	ServerHost      string
	ServerPort      int32
}

func (*Packet2ClientProtocol) PacketID() uint8 { return 2 }

func (p *Packet2ClientProtocol) ReadPacketData(r *Reader) error {
	version, err := r.ReadInt8()
	if err != nil {
		return err
	}
	username, err := r.ReadString(16)
	if err != nil {
		return err
	}
	host, err := r.ReadString(255)
	if err != nil {
		return err
	}
	port, err := r.ReadInt32()
	if err != nil {
		return err
	}

	p.ProtocolVersion = version
	p.Username = username
	p.ServerHost = host
	p.ServerPort = port
	return nil
}

func (p *Packet2ClientProtocol) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(p.ProtocolVersion); err != nil {
		return err
	}
	if err := w.WriteString(p.Username); err != nil {
		return err
	}
	if err := w.WriteString(p.ServerHost); err != nil {
		return err
	}
	return w.WriteInt32(p.ServerPort)
}

func (p *Packet2ClientProtocol) PacketSize() int {
	return 3 + 2*utf16UnitsLen(p.Username)
}

// Packet205ClientCommand translates net.minecraft.src.Packet205ClientCommand.
type Packet205ClientCommand struct {
	ForceRespawn int8
}

func (*Packet205ClientCommand) PacketID() uint8 { return 205 }

func (p *Packet205ClientCommand) ReadPacketData(r *Reader) error {
	v, err := r.ReadInt8()
	if err != nil {
		return err
	}
	p.ForceRespawn = v
	return nil
}

func (p *Packet205ClientCommand) WritePacketData(w *Writer) error {
	return w.WriteUint8(uint8(p.ForceRespawn))
}

func (*Packet205ClientCommand) PacketSize() int { return 1 }

// Packet255KickDisconnect translates net.minecraft.src.Packet255KickDisconnect.
type Packet255KickDisconnect struct {
	Reason string
}

func (*Packet255KickDisconnect) PacketID() uint8 { return 255 }

func (p *Packet255KickDisconnect) ReadPacketData(r *Reader) error {
	s, err := r.ReadString(256)
	if err != nil {
		return err
	}
	p.Reason = s
	return nil
}

func (p *Packet255KickDisconnect) WritePacketData(w *Writer) error {
	return w.WriteString(p.Reason)
}

func (p *Packet255KickDisconnect) PacketSize() int {
	return utf16UnitsLen(p.Reason)
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented packet subset.
	_ = Register(0, true, true, func() Packet { return &Packet0KeepAlive{} })
	_ = Register(1, true, true, func() Packet { return &Packet1Login{} })
	_ = Register(2, false, true, func() Packet { return &Packet2ClientProtocol{} })
	_ = Register(205, false, true, func() Packet { return &Packet205ClientCommand{} })
	_ = Register(255, true, true, func() Packet { return &Packet255KickDisconnect{} })
}
