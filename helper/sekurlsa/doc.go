// Package sekurlsa extracts credential material from a Windows LSASS
// minidump — the consumer counterpart to credentials/lsassdump.
//
// Parse a MINIDUMP blob, locate the auth-package DLLs (lsasrv.dll,
// msv1_0.dll, wdigest.dll, kerberos.dll, tspkg.dll, cloudap.dll,
// livessp.dll, dpapisrv.dll) in the captured module list, scan for
// per-provider globals via byte-pattern templates, decrypt each
// provider's session list with the LSA crypto chain (3DES / AES via
// BCRYPT_KEY_HANDLE → KIWI_BCRYPT_KEY), and surface the typed
// credentials per LUID.
//
// Public surface:
//
//   - [Parse] — main entry point; takes any io.ReaderAt.
//   - [ParseFile] — convenience wrapper around os.Open + Parse.
//   - Result.Wipe() — zeroizes every credential's sensitive byte
//     buffer in place. wipe() is a mandatory part of the Credential
//     interface (compile-time guarantee that no provider's
//     plaintext / hash / PRT can leak through Result discard).
//   - [RegisterTemplate] — adds a per-build template at runtime for
//     builds the inline default set doesn't cover.
//
// Sentinel errors (errors.Is-friendly):
//
//   - [ErrNotMinidump] — not an MDMP blob.
//   - [ErrUnsupportedBuild] — no template for the dump's
//     BuildNumber. Result still populates Modules + BuildNumber +
//     Architecture so the caller can RegisterTemplate and retry.
//   - [ErrUnsupportedArchitecture] — dump is not x64 (WoW64 /
//     legacy x86 dumps rejected; v1 ships x64-only walkers).
//   - [ErrLSASRVNotFound] / [ErrMSVNotFound] — the named module is
//     missing from MODULE_LIST.
//   - [ErrKeyExtractFailed] — pattern matched but the BCRYPT key
//     blob header was malformed (typically a wrong-template false
//     positive on signatures).
//
// Scope (8 providers; each auto-disables when its Layout.NodeSize
// is zero, so a partial template stays inert at no runtime cost):
//
//   - MSV1_0 NTLM hashes (NT / LM / SHA1 / DPAPI key) — MSVCredential.
//   - Wdigest plaintext passwords — WdigestCredential.
//   - DPAPI master-key cache — DPAPIMasterKey. Parse() looks in
//     lsasrv.dll first, falls back to dpapisrv.dll where modern
//     builds keep the cache.
//   - TSPkg (Terminal Services / RDP) plaintext — TSPkgCredential.
//   - Kerberos password + ticket cache — KerberosCredential.
//     Walker uses RTL_AVL_TABLE on Vista+.
//   - CredMan / Vault per-session list — CredManCredential.
//   - CloudAP (Azure AD PRT, Microsoft-Account SSO) —
//     CloudAPCredential. Carries the raw Primary Refresh Token.
//   - LiveSSP (legacy Microsoft Account) — LiveSSPCredential.
//
// All Credential implementations are stored as pointer types
// (*XxxCredential) inside Session.Credentials so Result.Wipe can
// zeroize the underlying byte buffers in place. Constructing a
// Credential value (e.g., MSVCredential{...}) and putting it into
// []Credential will not compile — the interface requires the
// pointer-receiver wipe().
//
// Templates ship inline for Win10 19H1 → 22H2 (builds 18362–19045)
// and Win11 21H2 → 22H2 pre-22622 (builds 22000–22621). A dump from
// any of those builds parses out of the box — no operator setup. See
// default_templates.go for the canonical set; pypykatz (GPL-3) and
// mimikatz (CC-BY-NC-SA) are the published research sources for the
// byte patterns + offsets, which are facts about Microsoft binaries
// rather than copyrightable expression.
//
// Out of scope: WoW64 (x86) extraction, live-process attach. WoW64
// would need a parallel set of layouts with 4-byte pointers and
// 8-byte UNICODE_STRINGs; modern Win 10/11 lsass is x64 by default
// so the operational value is limited.
//
// Layering: Layer 2 alongside credentials/lsassdump. Pure Go; no
// dependency on win/api or kernel/driver. Cross-platform — analysts
// can parse a dump from any host, not just Windows. The package
// never opens lsass.exe itself; that's credentials/lsassdump's job.
//
// # MITRE ATT&CK
//
//   - T1003.001 (OS Credential Dumping: LSASS Memory)
//   - T1550.002 (Use Alternate Authentication Material: Pass the
//     Hash) — downstream consumer of MSVCredential
//   - T1558.003 (Steal or Forge Kerberos Tickets: Kerberoasting) —
//     downstream consumer of KerberosCredential
//
// # Detection level
//
// quiet
//
// The loud operations all live upstream — the dump itself
// (PROCESS_VM_READ + NtReadVirtualMemory, see
// docs/techniques/collection/lsass-dump.md) and any driver-assisted
// PPL bypass. Parsing happens in the implant's own address space
// with pure-Go primitives — no syscalls, no file opens beyond the
// caller-supplied dump bytes / path.
//
// # Required privileges
//
// unprivileged. The package never opens lsass.exe — it parses
// dump bytes the caller already has. All loud privileged work
// (PROCESS_VM_READ + NtReadVirtualMemory + optional PPL bypass)
// lives upstream of this package. [ParseFile] only needs
// standard read access to the dump file.
//
// # Platform
//
// Cross-platform. Pure-Go MINIDUMP walker + AES/3DES via
// `crypto/aes` + `crypto/des`; no syscalls, no win/api
// dependency. Analysts can parse a Windows dump on Linux,
// macOS, or in CI.
package sekurlsa
