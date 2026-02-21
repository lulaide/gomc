package protocol

// Packet40EntityMetadata translates net.minecraft.src.Packet40EntityMetadata.
type Packet40EntityMetadata struct {
	EntityID int32
	Metadata []WatchableObject
}

func (*Packet40EntityMetadata) PacketID() uint8 { return 40 }

func (p *Packet40EntityMetadata) ReadPacketData(r *Reader) error {
	entityID, err := r.ReadInt32()
	if err != nil {
		return err
	}
	metadata, err := readWatchableObjects(r)
	if err != nil {
		return err
	}
	p.EntityID = entityID
	p.Metadata = metadata
	return nil
}

func (p *Packet40EntityMetadata) WritePacketData(w *Writer) error {
	if err := w.WriteInt32(p.EntityID); err != nil {
		return err
	}
	return writeWatchableObjects(w, p.Metadata)
}

func (*Packet40EntityMetadata) PacketSize() int {
	// Translation target: Packet40EntityMetadata#getPacketSize returns constant 5 in MCP.
	return 5
}

func init() {
	// Translation target: Packet#addIdClassMapping for entity metadata packet.
	_ = Register(40, true, false, func() Packet { return &Packet40EntityMetadata{} })
}
