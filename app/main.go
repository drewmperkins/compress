package main

import (
	"compress/encoding/binary"
	"compress/zip"
	"log"
)

func main() {
	zip.Start()
}

func testByteRearranging() {
	b := []byte{0x04, 0x03, 0x4b, 0x50}
	//b := []byte{0x04}
	log.Println(string(b), len(b))
	aa := binary.LittleEndian.Uint16(b)
	var bbb binary.Buf = b
	log.Println(aa, bbb.Uint16())

	// 0x04034b50
	// b[0] = 0x4
	bb := uint32(b[0])
	// b[1]<<8 = 0x300
	c := uint32(b[1]) << 8
	// b[2]<<16 = 0x4b0000
	d := uint32(b[2]) << 16
	// b[3]<<24 = 0x50000000
	e := uint32(b[3]) << 24
	// 0x504b0304
	// 01010000 00000000 00000000 00000000
	f := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	g := uint16(b[0]) | uint16(b[1])<<8
	h := uint32(b[2])<<16 | uint32(b[3])<<24
	i := uint32(g) | h

	// create a [4]byte array
	var a []byte = make([]byte, 4)
	a[0] = byte(i >> 24)
	a[1] = byte(i >> 16)
	a[2] = byte(i >> 8)
	a[3] = byte(i)
	log.Printf("%v%v%v%v%v%v%v%v%v", bb, c, d, e, f, g, i, a, aa)
	log.Println(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}
