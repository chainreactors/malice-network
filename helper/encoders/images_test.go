package encoders

import (
	"bytes"
	"testing"
)

var (
	imageTests = []struct {
		Input []byte
	}{
		{[]byte("abc")},   // byte count on image pixel alignment
		{[]byte("abcde")}, // byte count offset of image pixel alignment
		{[]byte{0x0, 0x01, 0x02, 0x03, 0x04}},
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x0}},
	}
)

func TestPNG(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for _, test := range imageTests {
		buf, _ := pngEncoder.Encode(test.Input)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(test.Input, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, test.Input)
		}
	}
}

func TestPNGRandomDataRandomSize(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(1024 * 1024)
		buf, _ := pngEncoder.Encode(sample)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(sample, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, sample)
		}
	}
}
