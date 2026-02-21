package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"

	"github.com/lulaide/gomc/pkg/nbt"
)

// Reader wraps packet binary reads.
type Reader struct {
	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (r *Reader) ReadInt8() (int8, error) {
	var v int8
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadUint8() (uint8, error) {
	var v uint8
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadInt16() (int16, error) {
	var v int16
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadUint16() (uint16, error) {
	var v uint16
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadInt32() (int32, error) {
	var v int32
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadInt64() (int64, error) {
	var v int64
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadFloat32() (float32, error) {
	var v float32
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

func (r *Reader) ReadFloat64() (float64, error) {
	var v float64
	err := binary.Read(r.r, binary.BigEndian, &v)
	return v, err
}

// ReadString translates Packet#readString(DataInput,int).
func (r *Reader) ReadString(maxLen int) (string, error) {
	length, err := r.ReadInt16()
	if err != nil {
		return "", err
	}
	if int(length) > maxLen {
		return "", fmt.Errorf("received string length longer than maximum allowed (%d > %d)", length, maxLen)
	}
	if length < 0 {
		return "", fmt.Errorf("received string length is less than zero")
	}

	units := make([]uint16, int(length))
	for i := 0; i < len(units); i++ {
		u, err := r.ReadUint16()
		if err != nil {
			return "", err
		}
		units[i] = u
	}
	return string(utf16.Decode(units)), nil
}

// ReadByteArray translates Packet#readBytesFromStream(DataInput).
func (r *Reader) ReadByteArray() ([]byte, error) {
	length, err := r.ReadInt16()
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, fmt.Errorf("key was smaller than zero")
	}

	out := make([]byte, int(length))
	if _, err := io.ReadFull(r.r, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReadNBTTagCompound translates Packet#readNBTTagCompound(DataInput).
func (r *Reader) ReadNBTTagCompound() (*nbt.CompoundTag, error) {
	length, err := r.ReadInt16()
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, nil
	}

	data := make([]byte, int(length))
	if _, err := io.ReadFull(r.r, data); err != nil {
		return nil, err
	}
	return nbt.ReadCompressed(bytes.NewReader(data))
}

// ReadItemStack translates Packet#readItemStack(DataInput).
func (r *Reader) ReadItemStack() (*ItemStack, error) {
	itemID, err := r.ReadInt16()
	if err != nil {
		return nil, err
	}
	if itemID < 0 {
		return nil, nil
	}

	stackSize, err := r.ReadInt8()
	if err != nil {
		return nil, err
	}
	damage, err := r.ReadInt16()
	if err != nil {
		return nil, err
	}
	tag, err := r.ReadNBTTagCompound()
	if err != nil {
		return nil, err
	}

	return &ItemStack{
		ItemID:     itemID,
		StackSize:  stackSize,
		ItemDamage: damage,
		Tag:        tag,
	}, nil
}

// Writer wraps packet binary writes.
type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) WriteInt8(v int8) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteUint8(v uint8) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteInt16(v int16) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteUint16(v uint16) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteInt32(v int32) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteInt64(v int64) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteFloat32(v float32) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

func (w *Writer) WriteFloat64(v float64) error {
	return binary.Write(w.w, binary.BigEndian, v)
}

// WriteString translates Packet#writeString(String,DataOutput).
func (w *Writer) WriteString(s string) error {
	units := utf16.Encode([]rune(s))
	if len(units) > 32767 {
		return fmt.Errorf("string too big")
	}

	if err := w.WriteInt16(int16(len(units))); err != nil {
		return err
	}
	for _, u := range units {
		if err := w.WriteUint16(u); err != nil {
			return err
		}
	}
	return nil
}

// WriteByteArray translates Packet#writeByteArray(DataOutput, byte[]).
func (w *Writer) WriteByteArray(data []byte) error {
	if len(data) > 32767 {
		return fmt.Errorf("byte array too long for short length prefix: %d", len(data))
	}
	if err := w.WriteInt16(int16(len(data))); err != nil {
		return err
	}
	_, err := w.w.Write(data)
	return err
}

// WriteNBTTagCompound translates Packet#writeNBTTagCompound.
func (w *Writer) WriteNBTTagCompound(tag *nbt.CompoundTag) error {
	if tag == nil {
		return w.WriteInt16(-1)
	}

	var buf bytes.Buffer
	if err := nbt.WriteCompressed(tag, &buf); err != nil {
		return err
	}
	data := buf.Bytes()
	if len(data) > 32767 {
		return fmt.Errorf("compressed nbt too long for short length prefix: %d", len(data))
	}
	if err := w.WriteInt16(int16(len(data))); err != nil {
		return err
	}
	_, err := w.w.Write(data)
	return err
}

// WriteItemStack translates Packet#writeItemStack(ItemStack,DataOutput).
func (w *Writer) WriteItemStack(stack *ItemStack) error {
	if stack == nil {
		return w.WriteInt16(-1)
	}

	if err := w.WriteInt16(stack.ItemID); err != nil {
		return err
	}
	if err := w.WriteInt8(stack.StackSize); err != nil {
		return err
	}
	if err := w.WriteInt16(stack.ItemDamage); err != nil {
		return err
	}
	return w.WriteNBTTagCompound(stack.Tag)
}
