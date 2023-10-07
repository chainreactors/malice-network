package encoders

import (
	"bytes"
	"testing"
)

func TestBase64(t *testing.T) {
	sample := randomData()

	b64 := new(Base64)
	output, _ := b64.Encode(sample)
	data, err := b64.Decode(output)
	if err != nil {
		t.Errorf("b64 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

}
