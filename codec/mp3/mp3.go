package mp3

import (
	"bufio"
	"fmt"
	"io"
)

type MP3 struct {
	r     *bufio.Reader
	frame *Frame
	err   error
}

func New(r io.Reader) (*MP3, error) {
	m := &MP3{
		r: bufio.NewReader(r),
	}
	b, err := m.r.Peek(10)
	if err != nil {
		return nil, err
	}
	if b[0] == 'I' && b[1] == 'D' && b[2] == '3' && b[3] < 0xff && b[4] < 0xff && b[6] < 0x80 && b[7] < 0x80 && b[8] < 0x80 && b[9] < 0x80 {
		var sz uint32
		sz = uint32(b[9] & 0x7f)
		sz += uint32(b[8]&0x7f) << 7
		sz += uint32(b[7]&0x7f) << 14
		sz += uint32(b[6]&0x7f) << 21
		sz += uint32(len(b))
		for ; sz > 0; sz-- {
			if _, err := m.r.ReadByte(); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

func (m *MP3) Scan() bool {
	var f Frame
	for {
		b, err := m.r.Peek(4)
		if err != nil {
			m.err = err
			return false
		}
		switch b[0] {
		case 0xff:
			if b[1]&0xe0 != 0xe0 {
				break
			}
			f = Frame{
				Version:   Version(b[1] & 0x18 >> 3),
				Layer:     Layer(b[1] & 0x6 >> 1),
				Protected: b[1]&0x1 == 0,
				Bitrate:   Bitrate(b[2] & 0xf0 >> 4),
				Sampling:  Sampling(b[2] & 0xc >> 2),
				Padding:   b[2]&0x2 != 0,
				Mode:      Mode(b[3] & 0xc >> 4),
				Emphasis:  Emphasis(b[3] & 0x3),
			}
			if !f.Valid() {
				break
			}
			f.Data = make([]byte, f.Length())
			m.frame = &f
			if n, err := m.r.Read(f.Data); err != nil {
				m.err = err
				return false
			} else if n < len(f.Data) {
				m.err = fmt.Errorf("mp3: short read")
				return false
			}
			return true
		}
		m.r.ReadByte()
	}
}

func (m *MP3) Err() error {
	if m.err == io.EOF {
		return nil
	}
	return m.err
}

func (m *MP3) Frame() *Frame {
	return m.frame
}

type Frame struct {
	Version
	Layer
	Protected bool
	Bitrate
	Sampling
	Padding bool
	Mode
	Emphasis
	Data []byte
}

// Length returns the frame length in bytes.
func (f *Frame) Length() int {
	padding := 0
	if f.Padding {
		padding = 1
	}
	switch f.Layer {
	case LayerI:
		return (12*f.BitrateIndex()*1000/f.SamplingIndex() + padding) * 4
	case LayerII, LayerIII:
		return 144*f.BitrateIndex()*1000/f.SamplingIndex() + padding
	default:
		return 0
	}
}

func (f *Frame) BitrateIndex() int {
	switch {
	case f.Version == MPEG1 && f.Layer == LayerI:
		return int(f.Bitrate) * 32
	case f.Version == MPEG1 && f.Layer == LayerIII:
		switch f.Bitrate {
		case 1:
			return 32
		case 2:
			return 40
		case 3:
			return 48
		case 4:
			return 56
		case 5:
			return 64
		case 6:
			return 80
		case 7:
			return 96
		case 8:
			return 112
		case 9:
			return 128
		case 10:
			return 160
		case 11:
			return 192
		case 12:
			return 224
		case 13:
			return 256
		case 14:
			return 320
		}
	}
	return 0
}

func (f *Frame) SamplingIndex() int {
	switch f.Version {
	case MPEG1:
		switch f.Sampling {
		case 0:
			return 44100
		case 1:
			return 48000
		case 2:
			return 32000
		}
	}
	return 0
}

func (f *Frame) Valid() bool {
	if f.Version < MPEG2 || f.Version > MPEG1 {
		return false
	}
	if f.Layer < LayerIII || f.Layer > LayerI {
		return false
	}
	if f.Bitrate == 0xf || f.Bitrate == 0 {
		return false
	}
	if f.Sampling >= 3 {
		return false
	}
	return true
}

type Version byte

const (
	MPEG1 Version = 3
	MPEG2         = 2
)

func (v Version) String() string {
	switch v {
	case MPEG1:
		return "MPEG1"
	case MPEG2:
		return "MPEG2"
	default:
		return "unknown"
	}
}

type Layer byte

const (
	LayerI   Layer = 3
	LayerII        = 2
	LayerIII       = 1
)

func (l Layer) String() string {
	switch l {
	case LayerI:
		return "layer I"
	case LayerII:
		return "layer II"
	case LayerIII:
		return "layer III"
	default:
		return "unknown"
	}
}

type Bitrate byte

type Sampling byte

type Mode byte

const (
	ModeStereo Mode = 0
	ModeJoint       = 1
	ModeDual        = 2
	ModeSingle      = 3
)

type Emphasis byte

const (
	EmphasisNone  Emphasis = 0
	Emphasis50_15          = 1
	EmphasisCCIT           = 3
)
