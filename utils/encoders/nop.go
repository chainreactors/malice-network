package encoders

// NoEncoder - A NOP encoder
type NoEncoder struct{}

// Encode - Don't do anything
func (n NoEncoder) Encode(data []byte) ([]byte, error) {
	return data, nil
}

// Decode - Don't do anything
func (n NoEncoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}
