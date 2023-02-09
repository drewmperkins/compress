package zip

import (
	"bytes"
	"compress/encoding/binary"
	"hash/crc32"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// little endian flip signatures
// 50 = PK = Phil Katz
const (
	fileHeaderSig       = 0x04034b50
	centralDirHeaderSig = 0x02014b50
	centralDirFooterSig = 0x06054b50
	directory64LocSig   = 0x07064b50
	directory64EndSig   = 0x06064b50
	dataDescriptorSig   = 0x08074b50
	fileHeaderLen       = 30 // + filename + extra
	centralDirHeaderLen = 46 // + filename + extra + comment
	centralDirFooterLen = 22 // + comment
)

// All multi-byte values in the header are stored in little-endian byte order. All length fields count the length in bytes.
// see: https://github.com/cthackers/adm-zip/blob/master/APPNOTE.md#44--explanation-of-fields
type Header struct {
	// fileHeader or centralDirectory
	signature [4]byte
	// NTFS 	= 0x0A
	//
	// Linux 	= 0x03
	versionMadeBy [2]byte
	/*
		0x3F = 6.3 = 7-zip .. wrong again
		0x14 = 20 = 2.0 = windows .. correct!
		6.3 - File is compressed using LZMA
		6.3 - File is compressed using PPMd+
		6.3 - File is encrypted using Blowfish
		6.3 - File is encrypted using Twofish

		20 / 10 = 2
		20 % 10 = 0
		2.0 - File is compressed using Deflate compression
	*/
	///
	versionNeedToExtract [2]byte
	flag                 [2]byte
	//	0x0 	= No compression
	//
	//	0x8 	= DEFLATE
	//
	//	0x14 	= LZMA
	//
	//	0x19 	= LZ77
	compression [2]byte
	// MS-DOS Time
	fileModTime []byte
	// MS-DOS Date
	fileModDate []byte
	// CRC-32 Checksum of uncompressed data
	crcUncompressed       [4]byte
	compressedSize        [4]byte
	unCompressedSize      [4]byte
	fileNameLen           [2]byte
	extraFieldLen         [2]byte
	fileCommentLen        [2]byte
	diskNumberLocation    [2]byte
	intFileAttrib         [2]byte
	extFileAttrib         [4]byte
	localFileHeaderOffset [4]byte
	fileName              []byte
	extraField            []byte
	fileComment           []byte
}

type Footer struct {
	signature          [4]byte
	diskNumberOf       [2]byte
	diskStart          [2]byte
	diskNumberRecords  [2]byte
	totalNumberRecords [2]byte
	sizeOfCentralDir   [4]byte
	offsetCentralDir   [4]byte
	commentLen         [2]byte
	comment            []byte
}

/*
	Note: We could of course use fmt.Sprintf("%b"), but I wanted to test writing my own.

	19/2 = 9.5 = 1
	9/2 = 4.5 = 1
	4/2 = 2 = 0
	2/2 = 1 = 0
	1
	19 = 10011
*/
///
func calcBaseTenToBinary(baseTen uint) string {
	var binary string = ""
	for baseTen > 1 {
		// 19 / 2 = 9.5
		var tmp float64 = float64(baseTen) / 2
		// check if whole number
		if isWholeNumber(tmp) {
			binary = "0" + binary
		} else {
			binary = "1" + binary
		}
		// 9.5 = decimal will be discarded
		baseTen = uint(tmp)
	}
	return "1" + binary
}

func isWholeNumber(n float64) bool {
	return math.Mod(math.Abs(n), 1) < 1e-10
}

func prefixConcatZeroes(s string, length int) string {
	for len(s) < length {
		s = "0" + s
	}
	return s
}

func msDosDateTimeConv(t time.Time, getDate bool) ([]byte, error) {
	// binary string
	var binStr string
	if getDate {
		if t.Year() < 1980 {
			t = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		binStr = prefixConcatZeroes(calcBaseTenToBinary(uint(t.Year()-1980)), 7) +
			prefixConcatZeroes(calcBaseTenToBinary(uint(t.Month())), 4) +
			prefixConcatZeroes(calcBaseTenToBinary(uint(t.Day())), 5)
	} else {
		// TODO: Hmm..either MS & 7-Zip are wrong or there is some issue calculating the seconds here.
		binStr = prefixConcatZeroes(calcBaseTenToBinary(uint(t.Hour())), 5) +
			prefixConcatZeroes(calcBaseTenToBinary(uint(t.Minute())), 6) +
			prefixConcatZeroes(calcBaseTenToBinary(uint(t.Second()/2)), 5)
	}
	// binary -> int
	ui, err := strconv.ParseUint(binStr, 2, 16)
	if err != nil {
		return nil, err
	}
	// flip to littleEndian
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(ui))
	return b, nil
}

func MsDosDateConv(t time.Time) ([]byte, error) {
	return msDosDateTimeConv(t, true)
}

func MsDosTimeConv(t time.Time) ([]byte, error) {
	return msDosDateTimeConv(t, false)
}

func Start() {
	// TODO: actually allow user to add files to zip.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(filepath.Dir(ex))

	// TODO: iterate through files\dirs
	fileBytes, err := os.ReadFile(exPath + "/data/lorem.txt")
	if err != nil {
		log.Printf("Error %v", err)
	}

	locHead, cdHead, err := GenHeader(exPath + "/data/lorem.txt")
	if err != nil {
		log.Printf("Error %v", err)
	}
	var fileHeadersWithData = append(locHead, fileBytes...)
	cdFoot := GenCentralDirFooter(fileHeadersWithData, cdHead)

	buf := append(fileHeadersWithData, cdHead...)
	buf = append(buf, cdFoot...)
	err = os.WriteFile(exPath+"/data/lorem.zip", buf, 0644)
	if err != nil {
		log.Printf("Error %v", err)
	}
}

func GenHeader(filePath string) ([]byte, []byte, error) {

	fileStat, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error %v", err)
		return nil, nil, err
	}
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error %v", err)
		return nil, nil, err
	}

	var h Header

	// TODO: Change signature between file and centralDir
	locSig := make([]byte, 4)
	binary.LittleEndian.PutUint32(locSig, fileHeaderSig)
	binary.LittleEndian.PutUint32(h.signature[:], centralDirHeaderSig)

	//versionMadeBy := make([]byte, 2)
	if runtime.GOOS == "windows" {
		binary.LittleEndian.PutUint16(h.versionMadeBy[:], 0x0A) // NTFS
	} else {
		binary.LittleEndian.PutUint16(h.versionMadeBy[:], 0x03) // UNIX
	}

	binary.LittleEndian.PutUint16(h.versionNeedToExtract[:], 0x14)

	h.flag = [2]byte{0x00, 0x00}
	h.compression = [2]byte{0x00, 0x00}

	// File last modification time = MS-DOS format
	h.fileModTime, _ = MsDosTimeConv(fileStat.ModTime())
	// File last modification date
	h.fileModDate, _ = MsDosDateConv(fileStat.ModTime())

	// checksum calculating
	table := crc32.MakeTable(crc32.IEEE)
	cs := crc32.Checksum(fileBytes, table)
	binary.LittleEndian.PutUint32(h.crcUncompressed[:], cs)

	// TODO: unsigned 32bit max here unless supporting Zip64.
	binary.LittleEndian.PutUint32(h.compressedSize[:], uint32(len(fileBytes)))
	binary.LittleEndian.PutUint32(h.unCompressedSize[:], uint32(len(fileBytes)))
	binary.LittleEndian.PutUint16(h.fileNameLen[:], uint16(len(fileStat.Name())))

	//h.extraField
	binary.LittleEndian.PutUint16(h.extraFieldLen[:], uint16(len(h.extraField)))
	//h.fileComment
	binary.LittleEndian.PutUint16(h.fileCommentLen[:], uint16(len(h.fileComment)))
	// TODO: update disk number as added
	h.diskNumberLocation = [2]byte{0x00, 0x00}
	h.intFileAttrib = [2]byte{0x00, 0x00}
	h.extFileAttrib = [4]byte{0x20, 0x00, 0x00, 0x00}
	// TODO: size of previous local file headers to find this offset location
	h.localFileHeaderOffset = [4]byte{0x00, 0x00, 0x00, 0x00}
	h.fileName = []byte(fileStat.Name())

	//buf := make([]byte, centralDirectoryHeaderLen)
	//buf = append(h.signature[:], versionMadeBy...)
	//buf = append(buf, versionNeedToExtract...)

	var locHead bytes.Buffer
	locHead.Write(locSig)
	locHead.Write(h.versionNeedToExtract[:])
	locHead.Write(h.flag[:])
	locHead.Write(h.compression[:])
	locHead.Write(h.fileModTime[:])
	locHead.Write(h.fileModDate[:])
	locHead.Write(h.crcUncompressed[:])
	locHead.Write(h.compressedSize[:])
	locHead.Write(h.unCompressedSize[:])
	locHead.Write(h.fileNameLen[:])
	locHead.Write(h.extraFieldLen[:])
	locHead.Write(h.fileName[:])
	locHead.Write(h.extraField[:])

	var cdHead bytes.Buffer
	cdHead.Write(h.signature[:])
	cdHead.Write(h.versionMadeBy[:])
	cdHead.Write(h.versionNeedToExtract[:])
	cdHead.Write(h.flag[:])
	cdHead.Write(h.compression[:])
	cdHead.Write(h.fileModTime[:])
	cdHead.Write(h.fileModDate[:])
	cdHead.Write(h.crcUncompressed[:])
	cdHead.Write(h.compressedSize[:])
	cdHead.Write(h.unCompressedSize[:])
	cdHead.Write(h.fileNameLen[:])
	cdHead.Write(h.extraFieldLen[:])
	cdHead.Write(h.fileCommentLen[:])
	cdHead.Write(h.diskNumberLocation[:])
	cdHead.Write(h.intFileAttrib[:])
	cdHead.Write(h.extFileAttrib[:])
	cdHead.Write(h.localFileHeaderOffset[:])
	cdHead.Write(h.fileName[:])
	cdHead.Write(h.extraField[:])
	cdHead.Write(h.fileComment[:])
	return locHead.Bytes(), cdHead.Bytes(), nil
}

func GenCentralDirFooter(fileHeadersWithData []byte, centralDirHeader []byte) []byte {

	var f Footer

	binary.LittleEndian.PutUint32(f.signature[:], centralDirFooterSig)
	f.diskNumberOf = [2]byte{0x00, 0x00}
	f.diskStart = [2]byte{0x00, 0x00}
	// TODO: Search through byte array for centralHeaderSig to get a count
	f.diskNumberRecords = [2]byte{0x01, 0x00}
	f.totalNumberRecords = [2]byte{0x01, 0x00}
	binary.LittleEndian.PutUint32(f.sizeOfCentralDir[:], uint32(len(centralDirHeader)))
	binary.LittleEndian.PutUint32(f.offsetCentralDir[:], uint32(len(fileHeadersWithData)))
	// comment
	f.commentLen = [2]byte{0x00, 0x00}

	var b bytes.Buffer
	b.Write(f.signature[:])
	b.Write(f.diskNumberOf[:])
	b.Write(f.diskStart[:])
	b.Write(f.diskNumberRecords[:])
	b.Write(f.totalNumberRecords[:])
	b.Write(f.sizeOfCentralDir[:])
	b.Write(f.offsetCentralDir[:])
	b.Write(f.commentLen[:])
	b.Write(f.comment[:])

	return b.Bytes()
}
