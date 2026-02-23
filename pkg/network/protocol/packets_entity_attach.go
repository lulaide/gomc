package protocol

// Packet39AttachEntity translates net.minecraft.src.Packet39AttachEntity.
//
// AttachState:
// - 0: riding
// - 1: leashed
type Packet39AttachEntity struct {
	RidingEntityID  int32
	VehicleEntityID int32
	AttachState     uint8
}

func (*Packet39AttachEntity) PacketID() uint8 { return 39 }

func (p *Packet39AttachEntity) ReadPacketData(r *Reader) error {
	ridingID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	vehicleID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	attachState, err := r.ReadUint8()
	if err != nil {
		return err
	}
	p.RidingEntityID = ridingID
	p.VehicleEntityID = vehicleID
	p.AttachState = attachState
	return nil
}

func (p *Packet39AttachEntity) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.RidingEntityID); err != nil {
		return err
	}
	if err := w.WriteInt32(p.VehicleEntityID); err != nil {
		return err
	}
	return w.WriteUint8(p.AttachState)
}

func (*Packet39AttachEntity) PacketSize() int { return 9 }

func init() {
	// Translation target: Packet#addIdClassMapping(39, true, false, Packet39AttachEntity.class)
	_ = Register(39, true, false, func() Packet { return &Packet39AttachEntity{} })
}
