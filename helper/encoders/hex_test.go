package encoders

import (
	"bytes"
	"testing"
)

func TestHex(t *testing.T) {
	sample := randomData()

	// Server-side
	x := new(Hex)
	output, _ := x.Encode(sample)
	data, err := x.Decode(output)
	if err != nil {
		t.Errorf("hex decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

}
