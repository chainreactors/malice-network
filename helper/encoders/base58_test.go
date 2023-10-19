package encoders

import (
	"bytes"
	"testing"
)

func TestBase58(t *testing.T) {
	sample := randomData()

	b58 := new(Base58)
	output, _ := b58.Encode(sample)
	data, err := b58.Decode(output)
	if err != nil {
		t.Errorf("b58 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

}
