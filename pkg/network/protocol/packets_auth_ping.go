package protocol

const packetIDCustomPayload = uint8(250)

// Packet252SharedKey translates net.minecraft.src.Packet252SharedKey stream format.
type Packet252SharedKey struct {
	SharedSecret []byte
	VerifyToken  []byte
}

func (*Packet252SharedKey) PacketID() uint8 { return 252 }

func (p *Packet252SharedKey) ReadPacketData(r *Reader) error {
	secret, err := r.ReadByteArray()
	if err != nil {
		return err
	}
	token, err := r.ReadByteArray()
	if err != nil {
		return err
	}
	p.SharedSecret = secret
	p.VerifyToken = token
	return nil
}

func (p *Packet252SharedKey) WritePacketData(w *Writer) error {
	if err := w.WriteByteArray(p.SharedSecret); err != nil {
		return err
	}
	return w.WriteByteArray(p.VerifyToken)
}

func (p *Packet252SharedKey) PacketSize() int {
	return 2 + len(p.SharedSecret) + 2 + len(p.VerifyToken)
}

// Packet253ServerAuthData translates net.minecraft.src.Packet253ServerAuthData stream format.
type Packet253ServerAuthData struct {
	ServerID    string
	PublicKey   []byte
	VerifyToken []byte
}

func (*Packet253ServerAuthData) PacketID() uint8 { return 253 }

func (p *Packet253ServerAuthData) ReadPacketData(r *Reader) error {
	serverID, err := r.ReadString(20)
	if err != nil {
		return err
	}
	pubKey, err := r.ReadByteArray()
	if err != nil {
		return err
	}
	token, err := r.ReadByteArray()
	if err != nil {
		return err
	}
	p.ServerID = serverID
	p.PublicKey = pubKey
	p.VerifyToken = token
	return nil
}

func (p *Packet253ServerAuthData) WritePacketData(w *Writer) error {
	if err := w.WriteString(p.ServerID); err != nil {
		return err
	}
	if err := w.WriteByteArray(p.PublicKey); err != nil {
		return err
	}
	return w.WriteByteArray(p.VerifyToken)
}

func (p *Packet253ServerAuthData) PacketSize() int {
	return 2 + utf16UnitsLen(p.ServerID)*2 + 2 + len(p.PublicKey) + 2 + len(p.VerifyToken)
}

// Packet254ServerPing translates net.minecraft.src.Packet254ServerPing.
type Packet254ServerPing struct {
	ReadSuccessfully int8
	ServerHost       string
	ServerPort       int32
}

func (*Packet254ServerPing) PacketID() uint8 { return 254 }

func (p *Packet254ServerPing) ReadPacketData(r *Reader) error {
	first, err := r.ReadInt8()
	if err != nil {
		p.ReadSuccessfully = 0
		p.ServerHost = ""
		return nil
	}
	p.ReadSuccessfully = first

	// Tolerant legacy ping parser, mirrors nested try/catch in MCP Packet254ServerPing#readPacketData.
	if _, err := r.ReadInt8(); err != nil {
		p.ServerHost = ""
		return nil
	}
	if _, err := r.ReadString(255); err != nil {
		p.ServerHost = ""
		return nil
	}
	if _, err := r.ReadInt16(); err != nil {
		p.ServerHost = ""
		return nil
	}
	version, err := r.ReadInt8()
	if err != nil {
		p.ServerHost = ""
		return nil
	}
	p.ReadSuccessfully = version

	if p.ReadSuccessfully >= 73 {
		host, err := r.ReadString(255)
		if err != nil {
			p.ServerHost = ""
			return nil
		}
		port, err := r.ReadInt32()
		if err != nil {
			p.ServerHost = ""
			return nil
		}
		p.ServerHost = host
		p.ServerPort = port
	}

	return nil
}

func (p *Packet254ServerPing) WritePacketData(w *Writer) error {
	if err := w.WriteInt8(1); err != nil {
		return err
	}
	if err := w.WriteUint8(packetIDCustomPayload); err != nil {
		return err
	}
	if err := w.WriteString("MC|PingHost"); err != nil {
		return err
	}
	payloadLength := 3 + utf16UnitsLen(p.ServerHost)*2 + 4
	if err := w.WriteInt16(int16(payloadLength)); err != nil {
		return err
	}
	if err := w.WriteInt8(p.ReadSuccessfully); err != nil {
		return err
	}
	if err := w.WriteString(p.ServerHost); err != nil {
		return err
	}
	return w.WriteInt32(p.ServerPort)
}

func (p *Packet254ServerPing) PacketSize() int {
	return 3 + utf16UnitsLen(p.ServerHost)*2 + 4
}

// IsLegacyPing mirrors Packet254ServerPing#func_140050_d.
func (p *Packet254ServerPing) IsLegacyPing() bool {
	return p.ReadSuccessfully == 0
}

func init() {
	// Translation target: Packet#addIdClassMapping for implemented auth/ping packets.
	_ = Register(252, true, true, func() Packet { return &Packet252SharedKey{} })
	_ = Register(253, true, false, func() Packet { return &Packet253ServerAuthData{} })
	_ = Register(254, false, true, func() Packet { return &Packet254ServerPing{} })
}
