package webhook

import (
	"net/http"
	"os"

	"github.com/acorn-io/baaah/pkg/webhook"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	cert = `
-----BEGIN CERTIFICATE-----
MIIDMjCCAhqgAwIBAgIRANEQVuBM4kHlB6mff7niotwwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBALOlrC5cw6EoWBEUr68Ebwy9DJllj07Mog6xglO54GaySNEbK6bD
Mj96QAWP3Fg5+2N/O3dnRDIl8+5RCuAx+Xk2dA5HWRyLmVrX8z43YwesB9T7Fhpm
VLuU/GWsVj+EIwND2Z9qSVF+lPzlhtg3k+GPhpMqi3ovU6E+n4ONd1Nr+hT79LlI
WOROGGqJ17ZGGLb04GaWiZjIquwAEVNVLAgY2n7LcQIsL3B8fYmUDZQdnBVGFp9D
3CThuaBUJOFxp1wO03KWRsSwUPvRqytfJ7eYP6aQ/174Ebup7v89H620XhDIA3wk
RhRzHDne8DXhOx02461CtpEdcTLN6oPHvpcCAwEAAaOBgDB+MA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBTa8m928U8PnbszvBz15djjIRaX0jAnBgNVHREEIDAeghxhY29ybi1sb2Nh
bC5hY29ybi1zeXN0ZW0uc3ZjMA0GCSqGSIb3DQEBCwUAA4IBAQA7kSNcQjCqjQPU
U73uuaTNLhIAhOhEKKVR7r8enO1YOvy0KUqDsOzXQ6gxiynC7dBOgYavSZuVnuk4
jXvw7VLbW+B/gxx+4zXxQL56ZvkvWaZ8nRztw0yqxOsFnxKyck++UvTII+CBzoCL
7Vg9keAxDZEIzghRJZoBqkP2Ys8giMU1j5AgbcgrRmTLPV8A7tOpxQskZVkM9GJ8
SEKz9Wr1jIoLt+CV1taK19LSFdoHgpZ5OvEuJ7GhuBDJQ9fI2bggT75vIGVqRC9K
WX5kRaRfOaKMejFQzCKPgZYbBwHtKH19cUUbPMnr0L5k6+ZOLyguVlTUvtZPqaOx
BR+BRDs5
-----END CERTIFICATE-----
`
	key = `
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCzpawuXMOhKFgR
FK+vBG8MvQyZZY9OzKIOsYJTueBmskjRGyumwzI/ekAFj9xYOftjfzt3Z0QyJfPu
UQrgMfl5NnQOR1kci5la1/M+N2MHrAfU+xYaZlS7lPxlrFY/hCMDQ9mfaklRfpT8
5YbYN5Phj4aTKot6L1OhPp+DjXdTa/oU+/S5SFjkThhqide2Rhi29OBmlomYyKrs
ABFTVSwIGNp+y3ECLC9wfH2JlA2UHZwVRhafQ9wk4bmgVCThcadcDtNylkbEsFD7
0asrXye3mD+mkP9e+BG7qe7/PR+ttF4QyAN8JEYUcxw53vA14TsdNuOtQraRHXEy
zeqDx76XAgMBAAECggEAQ3GDPrSczTf0OBVoD3C+sC2ZOU2ji5XPkWya3Qv/bw6n
v7lPnf/SMXqX5n6n3oeGFUiC7sTaGWmeNm4+gwS///2tfF7U2Z6fKDfCniz1BMBV
AGRzW93nGbVJPHKCvr3A10z1up+QfwPisz8IbMwQvLHBeCaUzn5OC08TW7QUhEB3
of2wpJHFuewE065ipIqBNpRO1UwK+JAmq5BkI2tRqy3/z3p3lthA69niV/7Qhe8t
IU6wuR3Hsas9d7QxBjJ14Ata0223lSoMkWoByOHno+7Tf2SG+naciqG6h5TI66WG
tlyTou8tUdW9sZE1cF1C2iy/BjPM4cPsC5lQ+q8WQQKBgQDrJd75RnxUnCqBnV71
YC467gD1epSnbediTrHPa36SG2TQ58yfdYY4uD7JWpQHmfxePHmIXUDGatkCCdU5
p/Wos1QKsTrhlTnZkswqQ/YLqTKff+bsc2QXbhTPalO6+h2K+6EHMQIWfznAGvyo
NkC9XLrfqTrxdMNPjrtt0aaHiQKBgQDDk9676fgmnvBJGfaWDzQmMPGH2fTP7pU2
LhvcELbK97xfL2xJcNkoaXfdXfnmYVlFVVNhhOjl3suBE+CXdjZSyYNGn+O1Zbef
yVfgWTNqUqsiuqmCGIKiBASGqLAevC3LbQSWVNfZ64Kyp31daoTH7Fr1IIKZD7ED
eNDOT6htHwKBgAZ4HUFIYiVFwpmcRb+EbOEsKRSX0b0ldecreRLWxz2nyUdCCUwd
xJqM3xVVC5uF7f59tW49+ok66Ut4D8itSUHh5R8CLzeDjnrg4gMLqZo6hm0C7Mx2
hDtsyN/H8hPDy8pGD/ENtRv/Vgxl8auDCpbrFS0QD9ISv0jSCXAFA4rZAoGATKZm
c/VVqSU/fRbs2pDo2lLyRlD4romN9ycJCi2Oxmtja1a1tO7CXSFAtgR/zXe3ugGf
5Sdm87hmv5bfvdy5m6aYiZRedRiBZ+FMTIQJL5Fouvq3NmKOyBqU/4WbSOBtfj2i
v5xO4Hx5w7T64CLAGW6bk1iDdqN8t6ShCUqU8vcCgYEApdVycTq9zJtdF1b2qafr
8QbyqBkPMoiFsTg//s9Rtrr51HoGv6awgeYn6IUM7D7BtDPprqnx4TSj51gqUPno
mznymGW8AcUnHbUyDbO3i4nfu60HMkBQSoDQD4A01qYskW7G3Hc6GZRLu5hriC9l
Ra530GC418C8T6SwhqxVGeQ=
-----END PRIVATE KEY-----
`
)

func keyFile() string {
	if err := os.WriteFile("key.pem", []byte(key), 0400); err != nil {
		panic(err)
	}
	return "key.pem"
}

func certFile() string {
	if err := os.WriteFile("cert.pem", []byte(cert), 0400); err != nil {
		panic(err)
	}
	return "cert.pem"
}

func Server(httpsAddr string) error {
	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	handler := &Handler{
		c: c,
	}

	router := webhook.NewRouter()
	router.Handle(handler)

	check := &healthz.Handler{}

	mux := http.NewServeMux()
	mux.Handle("/ping", check)
	mux.Handle("/healthz", check)
	mux.Handle("/", router)

	logrus.Infof("Starting Webhook HTTPS server on %s", httpsAddr)
	go func() {
		err := http.ListenAndServeTLS(httpsAddr, certFile(), keyFile(), mux)
		logrus.Fatal(err)
	}()
	return nil
}
