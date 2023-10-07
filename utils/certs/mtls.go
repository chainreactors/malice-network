package certs

const (
	// MtlsImplantCA - Directory containing HTTPS server certificates
	MtlsImplantCA = "mtls-implant"
	MtlsServerCA  = "mtls-server"
)

// MtlsC2ServerGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ServerGenerateECCCertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsServerCA, host, false, false)
	err := saveCertificate(MtlsServerCA, ECCKey, host, cert, key)
	return cert, key, err
}

// MtlsC2ImplantGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ImplantGenerateECCCertificate(name string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsImplantCA, name, false, true)
	err := saveCertificate(MtlsImplantCA, ECCKey, name, cert, key)
	return cert, key, err
}
