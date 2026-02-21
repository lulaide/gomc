package protocol

// Packet62LevelSound translates net.minecraft.src.Packet62LevelSound.
type Packet62LevelSound struct {
	SoundName string
	EffectX   int32 // fixed-point: world * 8
	EffectY   int32 // fixed-point: world * 8
	EffectZ   int32 // fixed-point: world * 8
	Volume    float32
	Pitch     uint8 // 63 == 1.0
}

func (*Packet62LevelSound) PacketID() uint8 { return 62 }

func (p *Packet62LevelSound) ReadPacketData(r *Reader) error {
	soundName, err := r.ReadString(256)
	if err != nil {
		return err
	}
	effectX, err := r.ReadInt32()
	if err != nil {
		return err
	}
	effectY, err := r.ReadInt32()
	if err != nil {
		return err
	}
	effectZ, err := r.ReadInt32()
	if err != nil {
		return err
	}
	volume, err := r.ReadFloat32()
	if err != nil {
		return err
	}
	pitch, err := r.ReadUint8()
	if err != nil {
		return err
	}

	p.SoundName = soundName
	p.EffectX = effectX
	p.EffectY = effectY
	p.EffectZ = effectZ
	p.Volume = volume
	p.Pitch = pitch
	return nil
}

func (p *Packet62LevelSound) WritePacketData(w *Writer) error {
	if err := w.WriteString(p.SoundName); err != nil {
		return err
	}
	if err := w.WriteInt32(p.EffectX); err != nil {
		return err
	}
	if err := w.WriteInt32(p.EffectY); err != nil {
		return err
	}
	if err := w.WriteInt32(p.EffectZ); err != nil {
		return err
	}
	if err := w.WriteFloat32(p.Volume); err != nil {
		return err
	}
	return w.WriteUint8(p.Pitch)
}

func (p *Packet62LevelSound) PacketSize() int {
	// 2-byte utf length + UTF-16 units*2 + 4+4+4 + 4 + 1
	return utf16UnitsLen(p.SoundName)*2 + 19
}

func (p *Packet62LevelSound) EffectXWorld() float64 { return float64(p.EffectX) / 8.0 }
func (p *Packet62LevelSound) EffectYWorld() float64 { return float64(p.EffectY) / 8.0 }
func (p *Packet62LevelSound) EffectZWorld() float64 { return float64(p.EffectZ) / 8.0 }
func (p *Packet62LevelSound) PitchFloat() float32   { return float32(p.Pitch) / 63.0 }

func init() {
	_ = Register(62, true, false, func() Packet { return &Packet62LevelSound{} })
}
