package chunk

// NibbleArray translates net.minecraft.src.NibbleArray.
type NibbleArray struct {
	Data []byte

	depthBits         int
	depthBitsPlusFour int
}

func NewNibbleArray(size int, depthBits int) *NibbleArray {
	return &NibbleArray{
		Data:              make([]byte, size>>1),
		depthBits:         depthBits,
		depthBitsPlusFour: depthBits + 4,
	}
}

func NewNibbleArrayFromData(data []byte, depthBits int) *NibbleArray {
	return &NibbleArray{
		Data:              data,
		depthBits:         depthBits,
		depthBitsPlusFour: depthBits + 4,
	}
}

// Get returns nibble value for x,y,z.
//
// Translation target: NibbleArray#get(int,int,int)
func (n *NibbleArray) Get(x, y, z int) int {
	index := y<<n.depthBitsPlusFour | z<<n.depthBits | x
	byteIndex := index >> 1
	half := index & 1
	if half == 0 {
		return int(n.Data[byteIndex] & 0x0F)
	}
	return int((n.Data[byteIndex] >> 4) & 0x0F)
}

// Set sets nibble value for x,y,z.
//
// Translation target: NibbleArray#set(int,int,int,int)
func (n *NibbleArray) Set(x, y, z, val int) {
	index := y<<n.depthBitsPlusFour | z<<n.depthBits | x
	byteIndex := index >> 1
	half := index & 1
	if half == 0 {
		n.Data[byteIndex] = byte((n.Data[byteIndex] & 0xF0) | (byte(val) & 0x0F))
		return
	}
	n.Data[byteIndex] = byte((n.Data[byteIndex] & 0x0F) | ((byte(val) & 0x0F) << 4))
}
