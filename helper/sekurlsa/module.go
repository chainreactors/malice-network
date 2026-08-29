package sekurlsa

import "strings"

// Module is a loaded module captured in the minidump's MODULE_LIST
// stream — typically `lsasrv.dll`, `msv1_0.dll`, `kerberos.dll` for
// our use case. Mirrors the credentials/lsassdump.Module shape so
// callers that already consume the writer side recognise the fields,
// but the type is independent (no dependency from the parser back
// onto the dumper).
type Module struct {
	// Name is the basename — pypykatz emits "lsasrv.dll" with the
	// extension and lowercase casing; we preserve whatever the
	// MINIDUMP_STRING contained. ModuleByName matches case-insensitively.
	Name string

	// BaseOfImage is the lsass.exe virtual address where the module is
	// mapped. Pattern scans run inside [BaseOfImage, BaseOfImage+SizeOfImage).
	BaseOfImage uint64

	// SizeOfImage is the in-memory image size from the PE optional
	// header. Used to bound pattern scans + reject scans that would
	// run past the end of the captured region.
	SizeOfImage uint32

	// TimeDateStamp + CheckSum from the PE header. Useful for cross-
	// referencing a specific module build against a templates
	// catalogue when BuildNumber alone is too coarse.
	TimeDateStamp uint32
	CheckSum      uint32
}

// ModuleByName returns the first module whose basename matches name
// case-insensitively. Returns (zero, false) when no match — callers
// surface that as ErrLSASRVNotFound or ErrMSVNotFound.
//
// Real Win 10/11 dumps store the full module path
// (`C:\Windows\system32\lsasrv.dll`) in MODULE_LIST.MODULE_STRING;
// the matcher reduces both sides to the basename (last `\` or `/`
// segment) so callers can pass either `lsasrv.dll` or the full
// path. Linear scan — typical lsass dumps have <100 modules and we
// look up each provider exactly once per parse.
func (r *Result) ModuleByName(name string) (Module, bool) {
	target := basename(name)
	for _, m := range r.Modules {
		if strings.EqualFold(basename(m.Name), target) {
			return m, true
		}
	}
	return Module{}, false
}

// basename returns the trailing `\` or `/`-delimited segment of p.
// Used by ModuleByName to reduce a full Windows path
// (`C:\Windows\system32\lsasrv.dll`) to its basename for case-
// insensitive matching against well-known DLL names.
func basename(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// modulesFromReader projects the parser's internal rawModule slice
// onto the public Module type. Called once during Parse so the
// caller's Result snapshot is decoupled from the live reader's
// internals.
func modulesFromReader(r *reader) []Module {
	out := make([]Module, len(r.modules))
	for i, m := range r.modules {
		out[i] = Module{
			Name:          m.Name,
			BaseOfImage:   m.BaseOfImage,
			SizeOfImage:   m.SizeOfImage,
			TimeDateStamp: m.TimeDateStamp,
			CheckSum:      m.CheckSum,
		}
	}
	return out
}
