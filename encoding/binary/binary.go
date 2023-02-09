package binary

type Buf []byte

// byte ordering used by processors
var LittleEndian littleEndian

type littleEndian struct{}

// byte ordering used for networking
var BigEndian bigEndian

type bigEndian struct{}

// left shift bytes
func (littleEndian) Uint16(b []byte) uint16 {
	// 0x04034b50
	// b[0] 	= 0x4
	// b[1]<<8 	= 0x300
	// 0x0304
	_ = b[1]
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) PutUint16(b []byte, v uint16) {
	_ = b[1]
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func (littleEndian) Uint32(b []byte) uint32 {
	_ = b[3]
	return uint32(LittleEndian.Uint16(b)) | uint32(b[2])<<16 | uint32(b[3])<<24
}

func (littleEndian) PutUint32(b []byte, v uint32) {
	_ = b[3]
	LittleEndian.PutUint16(b, uint16(v))
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func (littleEndian) Uint64(b []byte) uint64 {
	_ = b[7]
	return uint64(LittleEndian.Uint32(b)) | uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// drops bytes as read
//
// Methods can only be added to types you own. You don't own the []byte
// type so you can't add methods to it.  Thus creating the buf type.
func (b *Buf) Uint16() uint16 {
	v := LittleEndian.Uint16(*b)
	// slice.. first 2 bytes dropped leaving rest
	*b = (*b)[2:]
	return v
}

func (b *Buf) Uint32() uint32 {
	v := LittleEndian.Uint32(*b)
	*b = (*b)[4:]
	return v
}

func (b *Buf) Uint64() uint64 {
	v := LittleEndian.Uint64(*b)
	*b = (*b)[8:]
	return v
}
