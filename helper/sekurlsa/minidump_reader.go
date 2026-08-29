package sekurlsa

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// MINIDUMP format reference: see credentials/lsassdump/minidump.go for
// the writer side. Layout fields are little-endian uint32 / uint64.
//
// We only parse the four streams relevant to credential extraction:
// SystemInfo, ThreadList, ModuleList, Memory64List. Other streams
// (HandleData, MemoryInfoList, …) are ignored — pypykatz behaves the
// same way.

// Stream type codes (MINIDUMP_STREAM_TYPE).
const (
	streamThreadList   uint32 = 3
	streamModuleList   uint32 = 4
	streamSystemInfo   uint32 = 7
	streamMemory64List uint32 = 9
)

const (
	miniDumpSignature = 0x504D444D // "MDMP"
	miniDumpHeaderLen = 32
	miniDumpDirEntry  = 12
)

// systemInfo is the subset of MINIDUMP_SYSTEM_INFO we keep. The CPU
// vendor / feature union is dropped — credential parsers don't care.
type systemInfo struct {
	ProcessorArchitecture uint16
	ProcessorLevel        uint16
	ProcessorRevision     uint16
	NumberOfProcessors    uint8
	ProductType           uint8
	MajorVersion          uint32
	MinorVersion          uint32
	BuildNumber           uint32
	PlatformID            uint32
	CSDVersion            string
	SuiteMask             uint16
}

// rawModule is the parsed MINIDUMP_MODULE entry. Name is resolved from
// the trailing MINIDUMP_STRINGs section during parse.
type rawModule struct {
	BaseOfImage   uint64
	SizeOfImage   uint32
	TimeDateStamp uint32
	CheckSum      uint32
	Name          string
}

// memoryRange describes one Memory64List entry — the address it was
// captured from in lsass.exe and where its bytes sit in the dump.
type memoryRange struct {
	StartOfMemoryRange uint64
	DataSize           uint64
	Rva                uint64 // file offset where the bytes live
}

// reader is the parsed-but-lazy MINIDUMP container. Callers obtain
// one via openReader; subsequent random-access reads use ReadAt
// (file offset) or ReadVA (process VA — translates through Memory64).
type reader struct {
	src        io.ReaderAt
	size       int64
	systemInfo systemInfo
	modules    []rawModule
	memory     []memoryRange
}

// openReader parses header + directory + the four streams. The
// constructor is intentionally eager — credential extraction touches
// every stream we expose, so the upfront cost amortises immediately
// and keeps the API simple (no "did I parse this stream yet?"
// state to track).
func openReader(src io.ReaderAt, size int64) (*reader, error) {
	if size < miniDumpHeaderLen {
		return nil, fmt.Errorf("%w: file shorter than MINIDUMP header (%d < %d)",
			ErrNotMinidump, size, miniDumpHeaderLen)
	}
	header := make([]byte, miniDumpHeaderLen)
	if _, err := src.ReadAt(header, 0); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	sig := binary.LittleEndian.Uint32(header[0:4])
	if sig != miniDumpSignature {
		return nil, fmt.Errorf("%w: signature 0x%08X (want 0x%08X)",
			ErrNotMinidump, sig, miniDumpSignature)
	}
	numStreams := binary.LittleEndian.Uint32(header[8:12])
	dirRva := binary.LittleEndian.Uint32(header[12:16])

	r := &reader{src: src, size: size}
	if err := r.parseDirectory(dirRva, numStreams); err != nil {
		return nil, err
	}
	return r, nil
}

// parseDirectory reads numStreams MINIDUMP_DIRECTORY entries starting
// at dirRva and dispatches each to the appropriate per-stream parser.
// Unknown stream types are silently skipped so a dump with extra
// streams (HandleData, MiscInfo, …) doesn't fail.
func (r *reader) parseDirectory(dirRva, numStreams uint32) error {
	dirSize := int64(numStreams) * miniDumpDirEntry
	if int64(dirRva)+dirSize > r.size {
		return fmt.Errorf("%w: directory range overflows dump (%d+%d > %d)",
			ErrNotMinidump, dirRva, dirSize, r.size)
	}
	dir := make([]byte, dirSize)
	if _, err := r.src.ReadAt(dir, int64(dirRva)); err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	for i := uint32(0); i < numStreams; i++ {
		off := i * miniDumpDirEntry
		streamType := binary.LittleEndian.Uint32(dir[off : off+4])
		streamSize := binary.LittleEndian.Uint32(dir[off+4 : off+8])
		streamRva := binary.LittleEndian.Uint32(dir[off+8 : off+12])
		switch streamType {
		case streamSystemInfo:
			if err := r.parseSystemInfo(streamRva, streamSize); err != nil {
				return fmt.Errorf("system_info stream: %w", err)
			}
		case streamModuleList:
			if err := r.parseModuleList(streamRva, streamSize); err != nil {
				return fmt.Errorf("module_list stream: %w", err)
			}
		case streamMemory64List:
			if err := r.parseMemory64List(streamRva, streamSize); err != nil {
				return fmt.Errorf("memory64_list stream: %w", err)
			}
		default:
			// ThreadList + every other stream we don't currently use.
			// Silent skip — pypykatz does the same.
		}
	}
	return nil
}

// parseSystemInfo reads the 56-byte SystemInfo header + the trailing
// CSDVersion MINIDUMP_STRING.
func (r *reader) parseSystemInfo(rva, sz uint32) error {
	if sz < 56 {
		return fmt.Errorf("system_info too small: %d", sz)
	}
	if int64(rva)+int64(sz) > r.size {
		return fmt.Errorf("system_info overflows dump: %d+%d > %d", rva, sz, r.size)
	}
	buf := make([]byte, 56)
	if _, err := r.src.ReadAt(buf, int64(rva)); err != nil {
		return fmt.Errorf("read system_info: %w", err)
	}
	si := systemInfo{
		ProcessorArchitecture: binary.LittleEndian.Uint16(buf[0:2]),
		ProcessorLevel:        binary.LittleEndian.Uint16(buf[2:4]),
		ProcessorRevision:     binary.LittleEndian.Uint16(buf[4:6]),
		NumberOfProcessors:    buf[6],
		ProductType:           buf[7],
		MajorVersion:          binary.LittleEndian.Uint32(buf[8:12]),
		MinorVersion:          binary.LittleEndian.Uint32(buf[12:16]),
		BuildNumber:           binary.LittleEndian.Uint32(buf[16:20]),
		PlatformID:            binary.LittleEndian.Uint32(buf[20:24]),
		SuiteMask:             binary.LittleEndian.Uint16(buf[28:30]),
	}
	csdRva := binary.LittleEndian.Uint32(buf[24:28])
	if csdRva != 0 {
		s, err := r.readMinidumpString(csdRva)
		if err != nil {
			return fmt.Errorf("CSDVersion string: %w", err)
		}
		si.CSDVersion = s
	}
	r.systemInfo = si
	return nil
}

// parseModuleList reads the count + N × MINIDUMP_MODULE entries +
// resolves each ModuleNameRva into a Go string.
func (r *reader) parseModuleList(rva, sz uint32) error {
	if sz < 4 {
		return fmt.Errorf("module_list too small: %d", sz)
	}
	if int64(rva)+int64(sz) > r.size {
		return fmt.Errorf("module_list overflows dump")
	}
	countBuf := make([]byte, 4)
	if _, err := r.src.ReadAt(countBuf, int64(rva)); err != nil {
		return fmt.Errorf("read module_list count: %w", err)
	}
	count := binary.LittleEndian.Uint32(countBuf)
	const moduleEntrySize = 108
	expectedSize := uint32(4) + count*moduleEntrySize
	if sz < expectedSize {
		return fmt.Errorf("module_list size %d < expected %d for %d modules", sz, expectedSize, count)
	}
	entries := make([]byte, count*moduleEntrySize)
	if _, err := r.src.ReadAt(entries, int64(rva)+4); err != nil {
		return fmt.Errorf("read module_list entries: %w", err)
	}
	r.modules = make([]rawModule, count)
	for i := uint32(0); i < count; i++ {
		off := i * moduleEntrySize
		m := rawModule{
			BaseOfImage:   binary.LittleEndian.Uint64(entries[off : off+8]),
			SizeOfImage:   binary.LittleEndian.Uint32(entries[off+8 : off+12]),
			CheckSum:      binary.LittleEndian.Uint32(entries[off+12 : off+16]),
			TimeDateStamp: binary.LittleEndian.Uint32(entries[off+16 : off+20]),
		}
		nameRva := binary.LittleEndian.Uint32(entries[off+20 : off+24])
		name, err := r.readMinidumpString(nameRva)
		if err != nil {
			return fmt.Errorf("module[%d] name string: %w", i, err)
		}
		m.Name = name
		r.modules[i] = m
	}
	return nil
}

// parseMemory64List reads the 16-byte stream header (count + BaseRva)
// + count × 16-byte descriptors. Each descriptor's bytes live
// contiguously starting at BaseRva — we synthesise per-region Rva
// values so ReadVA can dereference without rewinding the dump.
func (r *reader) parseMemory64List(rva, sz uint32) error {
	if sz < 16 {
		return fmt.Errorf("memory64_list too small: %d", sz)
	}
	if int64(rva)+int64(sz) > r.size {
		return fmt.Errorf("memory64_list overflows dump")
	}
	hdr := make([]byte, 16)
	if _, err := r.src.ReadAt(hdr, int64(rva)); err != nil {
		return fmt.Errorf("read memory64_list header: %w", err)
	}
	count := binary.LittleEndian.Uint64(hdr[0:8])
	baseRva := binary.LittleEndian.Uint64(hdr[8:16])
	if count == 0 {
		return nil
	}
	const descSize = 16
	if sz < uint32(16+count*descSize) {
		return fmt.Errorf("memory64_list size %d < expected for %d entries", sz, count)
	}
	descs := make([]byte, count*descSize)
	if _, err := r.src.ReadAt(descs, int64(rva)+16); err != nil {
		return fmt.Errorf("read memory64_list descriptors: %w", err)
	}
	r.memory = make([]memoryRange, count)
	cursor := baseRva
	for i := uint64(0); i < count; i++ {
		off := i * descSize
		mr := memoryRange{
			StartOfMemoryRange: binary.LittleEndian.Uint64(descs[off : off+8]),
			DataSize:           binary.LittleEndian.Uint64(descs[off+8 : off+16]),
			Rva:                cursor,
		}
		cursor += mr.DataSize
		r.memory[i] = mr
	}
	return nil
}

// readMinidumpString reads a MINIDUMP_STRING (4-byte length + UTF-16LE
// payload + 2-byte NUL) at rva and decodes it to Go UTF-8.
func (r *reader) readMinidumpString(rva uint32) (string, error) {
	if int64(rva)+4 > r.size {
		return "", fmt.Errorf("string at 0x%X overflows dump", rva)
	}
	lenBuf := make([]byte, 4)
	if _, err := r.src.ReadAt(lenBuf, int64(rva)); err != nil {
		return "", err
	}
	byteLen := binary.LittleEndian.Uint32(lenBuf)
	if byteLen == 0 {
		return "", nil
	}
	if int64(rva)+4+int64(byteLen) > r.size {
		return "", fmt.Errorf("string at 0x%X (len %d) overflows dump", rva, byteLen)
	}
	body := make([]byte, byteLen)
	if _, err := r.src.ReadAt(body, int64(rva)+4); err != nil {
		return "", err
	}
	if byteLen%2 != 0 {
		return "", fmt.Errorf("string at 0x%X has odd byte length %d", rva, byteLen)
	}
	u16 := make([]uint16, byteLen/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(body[i*2 : i*2+2])
	}
	return decodeUTF16(u16), nil
}

// decodeUTF16 converts a UTF-16LE rune slice to Go string. Trailing
// NUL is stripped if present (MINIDUMP_STRING lengths sometimes
// include the terminator and sometimes don't depending on the writer).
func decodeUTF16(u16 []uint16) string {
	for i, c := range u16 {
		if c == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}

// ReadVA reads n bytes starting at the given lsass.exe virtual
// address, looking up the bytes through the Memory64List descriptors.
// Returns ErrAddressNotInDump when no descriptor covers va.
//
// Crosses descriptor boundaries safely — a single ReadVA call may
// span two adjacent regions if their captured ranges happen to
// abut (rare but legal).
func (r *reader) ReadVA(va uint64, n int) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	out := make([]byte, n)
	written := 0
	cursor := va
	for written < n {
		mr, ok := r.findMemoryRange(cursor)
		if !ok {
			return nil, fmt.Errorf("%w: VA 0x%X not covered", errAddressNotInDump, cursor)
		}
		offsetInRegion := cursor - mr.StartOfMemoryRange
		want := n - written
		avail := int(mr.DataSize - offsetInRegion)
		if avail <= 0 {
			return nil, fmt.Errorf("%w: VA 0x%X past region end", errAddressNotInDump, cursor)
		}
		read := want
		if read > avail {
			read = avail
		}
		if _, err := r.src.ReadAt(out[written:written+read], int64(mr.Rva+offsetInRegion)); err != nil {
			return nil, fmt.Errorf("read region @0x%X off=%d: %w", mr.StartOfMemoryRange, offsetInRegion, err)
		}
		written += read
		cursor += uint64(read)
	}
	return out, nil
}

// findMemoryRange returns the descriptor whose range contains va.
// Linear scan is fine — typical lsass dumps have <100 regions and
// the parser does <1000 ReadVA calls per dump.
func (r *reader) findMemoryRange(va uint64) (memoryRange, bool) {
	for _, mr := range r.memory {
		if va >= mr.StartOfMemoryRange && va < mr.StartOfMemoryRange+mr.DataSize {
			return mr, true
		}
	}
	return memoryRange{}, false
}

// errAddressNotInDump is the unexported sentinel ReadVA wraps. We
// don't surface it as a public Err* because callers shouldn't be
// switching on this — a missing VA is the parser's problem to
// report via Result.Warnings, not the consumer's to handle.
var errAddressNotInDump = fmt.Errorf("address not in dump")
