package swfinfo

import (
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/smira/lzma"
)

type SWF struct {
	Compression Compression
	Version     uint8
	FrameSize   Rect
	FrameRate   float32
	FrameCount  uint16
}

func Open(filename string) (*SWF, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &SWF{}
	err = s.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SWF) ReadFrom(f io.Reader) error {
	var (
		err        error
		signature  [3]byte
		fileLength int32
	)

	if err = binary.Read(f, binary.LittleEndian, &signature); err != nil {
		return err
	}

	switch signature[0] {
	case 'F':
		s.Compression = COMPRESS_NONE
	case 'C':
		s.Compression = COMPRESS_ZLIB
	case 'Z':
		s.Compression = COMPRESS_LZMA
	default:
		return &BadHeader{0, errors.New(string(signature[:]))}
	}

	if signature[1] != 'W' || signature[2] != 'S' {
		return &BadHeader{1, errors.New(string(signature[:]))}
	}
	// read version
	if err = binary.Read(f, binary.LittleEndian, &s.Version); err != nil {
		return &BadHeader{0, err}
	}
	// read length
	if err = binary.Read(f, binary.LittleEndian, &fileLength); err != nil {
		return &BadHeader{0, err}
	}

	// decompress
	if s.Compression == COMPRESS_ZLIB {
		var d io.ReadCloser
		d, err = zlib.NewReader(f)
		if err != nil {
			return &BadHeader{0, err}
		}
		defer d.Close()
		f = d
	} else if s.Compression == COMPRESS_LZMA {
		d := lzma.NewReader(f)
		defer d.Close()
		f = d
	}

	// read frame size
	if err = s.FrameSize.ReadFrom(f); err != nil {
		return &BadHeader{0, err}
	}
	// read frame rate
	/*
		if err = binary.Read(f, binary.LittleEndian, &s.FrameRate); err != nil {
			return &BadHeader{0, err}
		}
	*/
	var fr [2]byte
	if err = binary.Read(f, binary.LittleEndian, &fr); err != nil {
		return &BadHeader{0, err}
	}
	fmt.Printf("%d %d\n", fr[1], fr[0])
	frp, err := strconv.ParseFloat(fmt.Sprintf("%d.%d", fr[1], fr[0]), 32)
	if err != nil {
		return err
	}
	s.FrameRate = float32(frp)
	// read frame count
	if err = binary.Read(f, binary.LittleEndian, &s.FrameCount); err != nil {
		return &BadHeader{0, err}
	}

	return nil
}