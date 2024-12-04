package gonut

import (
	"encoding/hex"
	"testing"
	"unsafe"
)

func TestInstance_ToBytes(t *testing.T) {
	inst := Instance{}

	t.Logf("%x", unsafe.Sizeof(inst.DonutInstance))
	t.Logf("Sig: 0x%08x", unsafe.Offsetof(inst.Sig))
	t.Logf("Mac: 0x%08x", unsafe.Offsetof(inst.Mac))
	t.Logf("ModuleKey: 0x%08x", unsafe.Offsetof(inst.ModuleKey))
	t.Logf("Module: 0x%08x", unsafe.Offsetof(inst.Module))
	t.Logf("Module: 0x%08x", unsafe.Offsetof(inst.Module.Args))

	hex.Decode(inst.Sig[:DONUT_SIG_LEN], []byte("5033524e43365836"))
	//copy(inst.Sig[:DONUT_SIG_LEN], []byte{0x36, 0x58, 0x36, 0x43, 0x4e, 0x52, 0x33, 0x50})
	//copy(inst.Sig[:DONUT_SIG_LEN], []byte{0x50, 0x33, 0x52, 0x4e, 0x43, 0x36, 0x58, 0x36})

	inst.Iv = 0x5CFCA0F2F461E50D

	t.Logf("inst.SigX: %X", inst.Sig[:DONUT_SIG_LEN])
	t.Logf("inst.MacX: %X", Maru(inst.Sig[:DONUT_SIG_LEN], inst.Iv))

	t.Logf("inst.SigX: %X", inst.Sig[:])
	t.Logf("inst.MacX: %X", Maru(inst.Sig[:], inst.Iv))

	t.Logf("inst.Sig: %X", []byte{0x50, 0x33, 0x52, 0x4e, 0x43, 0x36, 0x58, 0x36})
	t.Logf("inst.Mac: %X", Maru([]byte{0x50, 0x33, 0x52, 0x4e, 0x43, 0x36, 0x58, 0x36}, inst.Iv))
}
