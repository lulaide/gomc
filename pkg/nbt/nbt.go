package nbt

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

const (
	TagEndID byte = iota
	TagByteID
	TagShortID
	TagIntID
	TagLongID
	TagFloatID
	TagDoubleID
	TagByteArrayID
	TagStringID
	TagListID
	TagCompoundID
	TagIntArrayID
)

var nbtTypes = [...]string{
	"END", "BYTE", "SHORT", "INT", "LONG", "FLOAT", "DOUBLE", "BYTE[]", "STRING", "LIST", "COMPOUND", "INT[]",
}

func TagName(id byte) string {
	switch id {
	case TagEndID:
		return "TAG_End"
	case TagByteID:
		return "TAG_Byte"
	case TagShortID:
		return "TAG_Short"
	case TagIntID:
		return "TAG_Int"
	case TagLongID:
		return "TAG_Long"
	case TagFloatID:
		return "TAG_Float"
	case TagDoubleID:
		return "TAG_Double"
	case TagByteArrayID:
		return "TAG_Byte_Array"
	case TagStringID:
		return "TAG_String"
	case TagListID:
		return "TAG_List"
	case TagCompoundID:
		return "TAG_Compound"
	case TagIntArrayID:
		return "TAG_Int_Array"
	default:
		return "UNKNOWN"
	}
}

type dataInput struct {
	r io.Reader
}

func (in *dataInput) readByte() (byte, error) {
	var buf [1]byte
	_, err := io.ReadFull(in.r, buf[:])
	return buf[0], err
}

func (in *dataInput) readInt16() (int16, error) {
	var v int16
	err := binary.Read(in.r, binary.BigEndian, &v)
	return v, err
}

func (in *dataInput) readInt32() (int32, error) {
	var v int32
	err := binary.Read(in.r, binary.BigEndian, &v)
	return v, err
}

func (in *dataInput) readInt64() (int64, error) {
	var v int64
	err := binary.Read(in.r, binary.BigEndian, &v)
	return v, err
}

func (in *dataInput) readFloat32() (float32, error) {
	var bits uint32
	if err := binary.Read(in.r, binary.BigEndian, &bits); err != nil {
		return 0, err
	}
	return math.Float32frombits(bits), nil
}

func (in *dataInput) readFloat64() (float64, error) {
	var bits uint64
	if err := binary.Read(in.r, binary.BigEndian, &bits); err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
}

func (in *dataInput) readUTF() (string, error) {
	var byteLen uint16
	if err := binary.Read(in.r, binary.BigEndian, &byteLen); err != nil {
		return "", err
	}
	if byteLen == 0 {
		return "", nil
	}
	buf := make([]byte, int(byteLen))
	if _, err := io.ReadFull(in.r, buf); err != nil {
		return "", err
	}
	return decodeModifiedUTF8(buf)
}

func (in *dataInput) readFully(buf []byte) error {
	_, err := io.ReadFull(in.r, buf)
	return err
}

type dataOutput struct {
	w io.Writer
}

func (out *dataOutput) writeByte(v byte) error {
	_, err := out.w.Write([]byte{v})
	return err
}

func (out *dataOutput) writeInt16(v int16) error {
	return binary.Write(out.w, binary.BigEndian, v)
}

func (out *dataOutput) writeInt32(v int32) error {
	return binary.Write(out.w, binary.BigEndian, v)
}

func (out *dataOutput) writeInt64(v int64) error {
	return binary.Write(out.w, binary.BigEndian, v)
}

func (out *dataOutput) writeFloat32(v float32) error {
	return binary.Write(out.w, binary.BigEndian, math.Float32bits(v))
}

func (out *dataOutput) writeFloat64(v float64) error {
	return binary.Write(out.w, binary.BigEndian, math.Float64bits(v))
}

func (out *dataOutput) writeUTF(v string) error {
	encoded, err := encodeModifiedUTF8(v)
	if err != nil {
		return err
	}
	if err := binary.Write(out.w, binary.BigEndian, uint16(len(encoded))); err != nil {
		return err
	}
	if len(encoded) == 0 {
		return nil
	}
	_, err = out.w.Write(encoded)
	return err
}

func (out *dataOutput) writeRaw(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	_, err := out.w.Write(buf)
	return err
}

type Tag interface {
	ID() byte
	Name() string
	SetName(name string) Tag
	Copy() Tag
	String() string
	writePayload(out *dataOutput) error
	readPayload(in *dataInput, depth int) error
}

type baseTag struct {
	name string
}

func normalizeName(name string) string {
	return name
}

func (b *baseTag) Name() string {
	if b.name == "" {
		return ""
	}
	return b.name
}

func (b *baseTag) setName(name string) {
	b.name = normalizeName(name)
}

type EndTag struct {
	baseTag
}

func NewEndTag() *EndTag {
	return &EndTag{}
}

func (t *EndTag) ID() byte { return TagEndID }
func (t *EndTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *EndTag) Copy() Tag                        { return &EndTag{} }
func (t *EndTag) String() string                   { return "END" }
func (t *EndTag) writePayload(_ *dataOutput) error { return nil }
func (t *EndTag) readPayload(_ *dataInput, _ int) error {
	return nil
}

type ByteTag struct {
	baseTag
	Data int8
}

func NewByteTag(name string, data int8) *ByteTag {
	t := &ByteTag{Data: data}
	t.setName(name)
	return t
}

func (t *ByteTag) ID() byte { return TagByteID }
func (t *ByteTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *ByteTag) Copy() Tag {
	return NewByteTag(t.Name(), t.Data)
}
func (t *ByteTag) String() string { return fmt.Sprintf("%d", t.Data) }
func (t *ByteTag) writePayload(out *dataOutput) error {
	return out.writeByte(byte(t.Data))
}
func (t *ByteTag) readPayload(in *dataInput, _ int) error {
	b, err := in.readByte()
	t.Data = int8(b)
	return err
}

type ShortTag struct {
	baseTag
	Data int16
}

func NewShortTag(name string, data int16) *ShortTag {
	t := &ShortTag{Data: data}
	t.setName(name)
	return t
}

func (t *ShortTag) ID() byte { return TagShortID }
func (t *ShortTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *ShortTag) Copy() Tag { return NewShortTag(t.Name(), t.Data) }
func (t *ShortTag) String() string {
	return fmt.Sprintf("%d", t.Data)
}
func (t *ShortTag) writePayload(out *dataOutput) error {
	return out.writeInt16(t.Data)
}
func (t *ShortTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readInt16()
	t.Data = v
	return err
}

type IntTag struct {
	baseTag
	Data int32
}

func NewIntTag(name string, data int32) *IntTag {
	t := &IntTag{Data: data}
	t.setName(name)
	return t
}

func (t *IntTag) ID() byte { return TagIntID }
func (t *IntTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *IntTag) Copy() Tag { return NewIntTag(t.Name(), t.Data) }
func (t *IntTag) String() string {
	return fmt.Sprintf("%d", t.Data)
}
func (t *IntTag) writePayload(out *dataOutput) error {
	return out.writeInt32(t.Data)
}
func (t *IntTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readInt32()
	t.Data = v
	return err
}

type LongTag struct {
	baseTag
	Data int64
}

func NewLongTag(name string, data int64) *LongTag {
	t := &LongTag{Data: data}
	t.setName(name)
	return t
}

func (t *LongTag) ID() byte { return TagLongID }
func (t *LongTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *LongTag) Copy() Tag { return NewLongTag(t.Name(), t.Data) }
func (t *LongTag) String() string {
	return fmt.Sprintf("%d", t.Data)
}
func (t *LongTag) writePayload(out *dataOutput) error {
	return out.writeInt64(t.Data)
}
func (t *LongTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readInt64()
	t.Data = v
	return err
}

type FloatTag struct {
	baseTag
	Data float32
}

func NewFloatTag(name string, data float32) *FloatTag {
	t := &FloatTag{Data: data}
	t.setName(name)
	return t
}

func (t *FloatTag) ID() byte { return TagFloatID }
func (t *FloatTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *FloatTag) Copy() Tag { return NewFloatTag(t.Name(), t.Data) }
func (t *FloatTag) String() string {
	return fmt.Sprintf("%g", t.Data)
}
func (t *FloatTag) writePayload(out *dataOutput) error {
	return out.writeFloat32(t.Data)
}
func (t *FloatTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readFloat32()
	t.Data = v
	return err
}

type DoubleTag struct {
	baseTag
	Data float64
}

func NewDoubleTag(name string, data float64) *DoubleTag {
	t := &DoubleTag{Data: data}
	t.setName(name)
	return t
}

func (t *DoubleTag) ID() byte { return TagDoubleID }
func (t *DoubleTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *DoubleTag) Copy() Tag { return NewDoubleTag(t.Name(), t.Data) }
func (t *DoubleTag) String() string {
	return fmt.Sprintf("%g", t.Data)
}
func (t *DoubleTag) writePayload(out *dataOutput) error {
	return out.writeFloat64(t.Data)
}
func (t *DoubleTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readFloat64()
	t.Data = v
	return err
}

type ByteArrayTag struct {
	baseTag
	Bytes []byte
}

func NewByteArrayTag(name string, data []byte) *ByteArrayTag {
	cp := make([]byte, len(data))
	copy(cp, data)
	t := &ByteArrayTag{Bytes: cp}
	t.setName(name)
	return t
}

func (t *ByteArrayTag) ID() byte { return TagByteArrayID }
func (t *ByteArrayTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *ByteArrayTag) Copy() Tag { return NewByteArrayTag(t.Name(), t.Bytes) }
func (t *ByteArrayTag) String() string {
	return fmt.Sprintf("[%d bytes]", len(t.Bytes))
}
func (t *ByteArrayTag) writePayload(out *dataOutput) error {
	if err := out.writeInt32(int32(len(t.Bytes))); err != nil {
		return err
	}
	return out.writeRaw(t.Bytes)
}
func (t *ByteArrayTag) readPayload(in *dataInput, _ int) error {
	n, err := in.readInt32()
	if err != nil {
		return err
	}
	if n < 0 {
		return errors.New("negative byte array length")
	}
	t.Bytes = make([]byte, int(n))
	return in.readFully(t.Bytes)
}

type StringTag struct {
	baseTag
	Data string
}

func NewStringTag(name, data string) *StringTag {
	t := &StringTag{Data: data}
	t.setName(name)
	return t
}

func (t *StringTag) ID() byte { return TagStringID }
func (t *StringTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *StringTag) Copy() Tag { return NewStringTag(t.Name(), t.Data) }
func (t *StringTag) String() string {
	return t.Data
}
func (t *StringTag) writePayload(out *dataOutput) error {
	return out.writeUTF(t.Data)
}
func (t *StringTag) readPayload(in *dataInput, _ int) error {
	v, err := in.readUTF()
	t.Data = v
	return err
}

type ListTag struct {
	baseTag
	TagList []Tag
	TagType byte
}

func NewListTag(name string) *ListTag {
	t := &ListTag{
		TagList: make([]Tag, 0),
	}
	t.setName(name)
	return t
}

func (t *ListTag) ID() byte { return TagListID }
func (t *ListTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *ListTag) String() string {
	return fmt.Sprintf("%d entries of type %s", len(t.TagList), TagName(t.TagType))
}
func (t *ListTag) writePayload(out *dataOutput) error {
	if len(t.TagList) > 0 {
		t.TagType = t.TagList[0].ID()
	} else {
		t.TagType = TagByteID
	}
	if err := out.writeByte(t.TagType); err != nil {
		return err
	}
	if err := out.writeInt32(int32(len(t.TagList))); err != nil {
		return err
	}
	for _, tag := range t.TagList {
		if err := tag.writePayload(out); err != nil {
			return err
		}
	}
	return nil
}
func (t *ListTag) readPayload(in *dataInput, depth int) error {
	// Translation target: NBTTagList#load(DataInput,int)
	if depth > 512 {
		return errors.New("tried to read NBT tag with too high complexity, depth > 512")
	}
	tagType, err := in.readByte()
	if err != nil {
		return err
	}
	count, err := in.readInt32()
	if err != nil {
		return err
	}
	t.TagType = tagType
	t.TagList = make([]Tag, 0, maxInt32(count))
	if count <= 0 {
		return nil
	}
	for i := int32(0); i < count; i++ {
		tag, newErr := newTag(tagType, "")
		if newErr != nil {
			return newErr
		}
		if err := tag.readPayload(in, depth+1); err != nil {
			return err
		}
		t.TagList = append(t.TagList, tag)
	}
	return nil
}
func (t *ListTag) Copy() Tag {
	cp := NewListTag(t.Name())
	cp.TagType = t.TagType
	cp.TagList = make([]Tag, 0, len(t.TagList))
	for _, tag := range t.TagList {
		cp.TagList = append(cp.TagList, tag.Copy())
	}
	return cp
}
func (t *ListTag) AppendTag(tag Tag) {
	t.TagType = tag.ID()
	t.TagList = append(t.TagList, tag)
}
func (t *ListTag) RemoveTag(index int) Tag {
	tag := t.TagList[index]
	t.TagList = append(t.TagList[:index], t.TagList[index+1:]...)
	return tag
}
func (t *ListTag) TagAt(index int) Tag {
	return t.TagList[index]
}
func (t *ListTag) TagCount() int {
	return len(t.TagList)
}

type CompoundTag struct {
	baseTag
	TagMap map[string]Tag
}

func NewCompoundTag(name string) *CompoundTag {
	t := &CompoundTag{
		TagMap: make(map[string]Tag),
	}
	t.setName(name)
	return t
}

func (t *CompoundTag) ID() byte { return TagCompoundID }
func (t *CompoundTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *CompoundTag) String() string {
	return fmt.Sprintf("%s:[%d tags]", t.Name(), len(t.TagMap))
}
func (t *CompoundTag) writePayload(out *dataOutput) error {
	for _, tag := range t.TagMap {
		if err := writeNamedTag(tag, out); err != nil {
			return err
		}
	}
	return out.writeByte(TagEndID)
}
func (t *CompoundTag) readPayload(in *dataInput, depth int) error {
	// Translation target: NBTTagCompound#load(DataInput,int)
	if depth > 512 {
		return errors.New("tried to read NBT tag with too high complexity, depth > 512")
	}
	t.TagMap = make(map[string]Tag)
	for {
		tag, err := readNamedTagDepth(in, depth+1)
		if err != nil {
			return err
		}
		if tag.ID() == TagEndID {
			break
		}
		t.TagMap[tag.Name()] = tag
	}
	return nil
}
func (t *CompoundTag) Copy() Tag {
	cp := NewCompoundTag(t.Name())
	for k, v := range t.TagMap {
		cp.TagMap[k] = v.Copy()
	}
	return cp
}
func (t *CompoundTag) GetTags() []Tag {
	tags := make([]Tag, 0, len(t.TagMap))
	for _, v := range t.TagMap {
		tags = append(tags, v)
	}
	return tags
}
func (t *CompoundTag) SetTag(name string, tag Tag) {
	t.TagMap[name] = tag.SetName(name)
}
func (t *CompoundTag) SetByte(name string, value int8) {
	t.SetTag(name, NewByteTag(name, value))
}
func (t *CompoundTag) SetShort(name string, value int16) {
	t.SetTag(name, NewShortTag(name, value))
}
func (t *CompoundTag) SetInteger(name string, value int32) {
	t.SetTag(name, NewIntTag(name, value))
}
func (t *CompoundTag) SetLong(name string, value int64) {
	t.SetTag(name, NewLongTag(name, value))
}
func (t *CompoundTag) SetFloat(name string, value float32) {
	t.SetTag(name, NewFloatTag(name, value))
}
func (t *CompoundTag) SetDouble(name string, value float64) {
	t.SetTag(name, NewDoubleTag(name, value))
}
func (t *CompoundTag) SetString(name, value string) {
	t.SetTag(name, NewStringTag(name, value))
}
func (t *CompoundTag) SetByteArray(name string, value []byte) {
	t.SetTag(name, NewByteArrayTag(name, value))
}
func (t *CompoundTag) SetIntArray(name string, value []int32) {
	t.SetTag(name, NewIntArrayTag(name, value))
}
func (t *CompoundTag) SetCompoundTag(name string, value *CompoundTag) {
	t.SetTag(name, value.SetName(name))
}
func (t *CompoundTag) SetBoolean(name string, value bool) {
	if value {
		t.SetByte(name, 1)
	} else {
		t.SetByte(name, 0)
	}
}
func (t *CompoundTag) GetTag(name string) Tag {
	return t.TagMap[name]
}
func (t *CompoundTag) HasKey(name string) bool {
	_, ok := t.TagMap[name]
	return ok
}
func (t *CompoundTag) RemoveTag(name string) {
	delete(t.TagMap, name)
}
func (t *CompoundTag) HasNoTags() bool {
	return len(t.TagMap) == 0
}

type IntArrayTag struct {
	baseTag
	Ints []int32
}

func NewIntArrayTag(name string, data []int32) *IntArrayTag {
	cp := make([]int32, len(data))
	copy(cp, data)
	t := &IntArrayTag{Ints: cp}
	t.setName(name)
	return t
}

func (t *IntArrayTag) ID() byte { return TagIntArrayID }
func (t *IntArrayTag) SetName(name string) Tag {
	t.setName(name)
	return t
}
func (t *IntArrayTag) Copy() Tag { return NewIntArrayTag(t.Name(), t.Ints) }
func (t *IntArrayTag) String() string {
	return fmt.Sprintf("[%d bytes]", len(t.Ints))
}
func (t *IntArrayTag) writePayload(out *dataOutput) error {
	if err := out.writeInt32(int32(len(t.Ints))); err != nil {
		return err
	}
	for _, v := range t.Ints {
		if err := out.writeInt32(v); err != nil {
			return err
		}
	}
	return nil
}
func (t *IntArrayTag) readPayload(in *dataInput, _ int) error {
	n, err := in.readInt32()
	if err != nil {
		return err
	}
	if n < 0 {
		return errors.New("negative int array length")
	}
	t.Ints = make([]int32, int(n))
	for i := int32(0); i < n; i++ {
		v, err := in.readInt32()
		if err != nil {
			return err
		}
		t.Ints[i] = v
	}
	return nil
}

func newTag(id byte, name string) (Tag, error) {
	switch id {
	case TagEndID:
		return NewEndTag().SetName(name), nil
	case TagByteID:
		return NewByteTag(name, 0), nil
	case TagShortID:
		return NewShortTag(name, 0), nil
	case TagIntID:
		return NewIntTag(name, 0), nil
	case TagLongID:
		return NewLongTag(name, 0), nil
	case TagFloatID:
		return NewFloatTag(name, 0), nil
	case TagDoubleID:
		return NewDoubleTag(name, 0), nil
	case TagByteArrayID:
		return NewByteArrayTag(name, nil), nil
	case TagStringID:
		return NewStringTag(name, ""), nil
	case TagListID:
		return NewListTag(name), nil
	case TagCompoundID:
		return NewCompoundTag(name), nil
	case TagIntArrayID:
		return NewIntArrayTag(name, nil), nil
	default:
		return nil, fmt.Errorf("unknown nbt tag type: %d", id)
	}
}

func readNamedTagDepth(in *dataInput, depth int) (Tag, error) {
	id, err := in.readByte()
	if err != nil {
		return nil, err
	}
	if id == TagEndID {
		return NewEndTag(), nil
	}

	name, err := in.readUTF()
	if err != nil {
		return nil, err
	}

	tag, err := newTag(id, name)
	if err != nil {
		return nil, err
	}
	if err := tag.readPayload(in, depth); err != nil {
		return nil, err
	}
	return tag, nil
}

func writeNamedTag(tag Tag, out *dataOutput) error {
	if err := out.writeByte(tag.ID()); err != nil {
		return err
	}
	if tag.ID() == TagEndID {
		return nil
	}
	if err := out.writeUTF(tag.Name()); err != nil {
		return err
	}
	return tag.writePayload(out)
}

// ReadNamedTag translates NBTBase#readNamedTag.
func ReadNamedTag(r io.Reader) (Tag, error) {
	return readNamedTagDepth(&dataInput{r: r}, 0)
}

// WriteNamedTag translates NBTBase#writeNamedTag.
func WriteNamedTag(tag Tag, w io.Writer) error {
	return writeNamedTag(tag, &dataOutput{w: w})
}

// Read reads a root compound from DataInput, equivalent to CompressedStreamTools#read(DataInput).
func Read(r io.Reader) (*CompoundTag, error) {
	tag, err := ReadNamedTag(r)
	if err != nil {
		return nil, err
	}
	root, ok := tag.(*CompoundTag)
	if !ok {
		return nil, errors.New("root tag must be a named compound tag")
	}
	return root, nil
}

// Write writes a root compound to DataOutput.
func Write(root *CompoundTag, w io.Writer) error {
	if root == nil {
		return errors.New("nil root compound")
	}
	return WriteNamedTag(root, w)
}

// ReadCompressed translates CompressedStreamTools#readCompressed(InputStream).
func ReadCompressed(r io.Reader) (*CompoundTag, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	return Read(gzr)
}

// WriteCompressed translates CompressedStreamTools#writeCompressed.
func WriteCompressed(root *CompoundTag, w io.Writer) error {
	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	return Write(root, gzw)
}

// Compress translates CompressedStreamTools#compress.
func Compress(root *CompoundTag) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteCompressed(root, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress translates CompressedStreamTools#decompress.
func Decompress(data []byte) (*CompoundTag, error) {
	return ReadCompressed(bytes.NewReader(data))
}

// ReadFile translates CompressedStreamTools#read(File).
func ReadFile(path string) (*CompoundTag, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// WriteFile translates CompressedStreamTools#write(NBTTagCompound, File).
func WriteFile(root *CompoundTag, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Write(root, f)
}

// SafeWriteFile translates CompressedStreamTools#safeWrite.
func SafeWriteFile(root *CompoundTag, path string) error {
	tmpPath := path + "_tmp"
	_ = os.Remove(tmpPath)

	if err := WriteFile(root, tmpPath); err != nil {
		return err
	}

	_ = os.Remove(path)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("failed to delete %s", path)
	}

	return os.Rename(tmpPath, path)
}

func maxInt32(v int32) int {
	if v <= 0 {
		return 0
	}
	return int(v)
}
