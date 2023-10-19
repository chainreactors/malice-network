package encoders

import (
	"bytes"
	"testing"
)

func TestBase32(t *testing.T) {
	sample := randomData()

	b32 := new(Base32)
	output, _ := b32.Encode(sample)
	data, err := b32.Decode(output)
	if err != nil {
		t.Errorf("b32 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}
