package v1

type CertUsage string

var (
	CertUsageServer = CertUsage("server")
	CertUsageClient = CertUsage("client")
)

type TLSAlgorithm string

var (
	TLSAlgorithmRSA   = TLSAlgorithm("rsa")
	TLSAlgorithmECDSA = TLSAlgorithm("ecdsa")
)

type TLSParams struct {
	Algorithm    TLSAlgorithm `json:"algorithm,omitempty"`
	CASecret     string       `json:"caSecret,omitempty"`
	Usage        CertUsage
	CommonName   string   `json:"commonName,omitempty"`
	Organization []string `json:"organization,omitempty"`
	SANs         []string `json:"sans,omitempty"`
	DurationDays int      `json:"durationDays,omitempty"`
}

func (t *TLSParams) Complete() {
	if t.Usage == "" {
		t.Usage = CertUsageServer
	}
	if t.DurationDays == 0 {
		t.DurationDays = 365
	}
}
