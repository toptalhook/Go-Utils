package encrypt

import (
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testCertChain = `-----BEGIN CERTIFICATE-----
MIIE3DCCA0SgAwIBAgIUKvzFXZamgum1ss+T490hiYDoszAwDQYJKoZIhvcNAQEM
BQAwTDELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJh
bmNpc2NvMRgwFgYDVQQDEw9zZ3gtY29vcmRpbmF0b3IwHhcNMjIwOTI4MDcxMzAw
WhcNMzIwOTI1MDcxMzAwWjBSMQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAU
BgNVBAcTDVNhbiBGcmFuY2lzY28xHjAcBgNVBAMTFXNneC1jb29yZGluYXRvci1p
bnRlcjCCAaIwDQYJKoZIhvcNAQEBBQADggGPADCCAYoCggGBAMjh9A4Wmsy5LHQp
DjikniH/jqIsJJRg7TBUqdiNgCoQbWAPWj+a3huQ7AEKgQH+MdKvFwRIoOftAV7r
uNrX+a4Q/b1Kx1EvjNgCs8zSQYw3s/UBfw9BnXcrwGplj7wsanHFreS8Ul7VQ5NV
Fb5G20yw31tbXpb0LGj3t5hFU+v578soorJGB0OXFZm6HYs77FxdvHZFfluTA6aK
4ThutDqgwmhZydMVuuO95fe01DUFvwR7gXxkRJwIumJaoYYBGI2WBrD1BmRzrWBx
LoQU0AWUl/joV2qPLechpnVZuMb8nAM5/epPEkf6CF0Caj2+PY6VoZnM4iSafgzC
eu8oKKbyEWEaRz0f8TezUwpFl/ROa3JS9v0b3yILV1Jp1wFcPsIdF0925hOTM6/m
H0iCuFChzfKakEsE0I5DoVlgxHXq0ruOsmuY32Lp5vBJk7N5JNpnfUrELGGToDFm
ZqgbCRFdBXv5xVRFT2fdyrSpI6KvrUekVpsfe4FByEUfBPSqUQIDAQABo4GvMIGs
MA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBRBJsMa
3GclgkZnPM0s2AHelW47uDAfBgNVHSMEGDAWgBSPUheLFd1VIGK858UuSYrMe42+
VDAPBgNVHREECDAGhwR/AAABMDgGA1UdHwQxMC8wLaAroCmGJ2h0dHBzOi8vczMu
bGFpc2t5LmNvbS9wdWJsaWMvbGFpc2t5LmNybDANBgkqhkiG9w0BAQwFAAOCAYEA
KKbLRHfaG/mEB3az4qoKBAQYy3SIDBSvBT5jT+AqLMzivLHAw5oHoF1AkfsGxcea
XQcFcqIVm49cS8x6hhY7RSCAnCzOcSOu5oGEuDvzqbc5O9DUtDEkh46kiVSnJzny
k2DJFpP0aXfRszSehEa58nQmWQMf9YmIGo/ZTKrO7Er0jXnXdWKTx4bZHbRYKnXG
MPC7YwtLB65kTab13Ln0/c9gsb0yFjfg6Niz6uEGDCFnriB5L1mGuPzB7pUVXQmn
YWpmmLsprvVNNySy3BDtGqyxKDxqTTaMX0iOKQ1AEt+bE+mqE/+GajPMp89NEqnL
UVGpNBYHMtuO30mf1W/BXXkHa+n9MMrbx0Kx+sZMMNJEjRddFJvVzZExFcIzw6un
2PBvgd0kWUOgTspjIPHpBVnuOYmp6I2+g7G2NfPf5NXg5e8ilp5OIqvwbvlrtsLa
PTuL0dTl0RFO5wsyAokn8EUjfzJhfz+8xEUo28CjO9Ku8JsOfCJN2stzSmK6stJl
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIEeTCCAuGgAwIBAgIUEV77hRsKEOh5u65RVaQNvW+gXW0wDQYJKoZIhvcNAQEM
BQAwTDELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJh
bmNpc2NvMRgwFgYDVQQDEw9zZ3gtY29vcmRpbmF0b3IwHhcNMjIwOTI4MDcxMDAw
WhcNMzIwOTI1MDcxMDAwWjBMMQswCQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAU
BgNVBAcTDVNhbiBGcmFuY2lzY28xGDAWBgNVBAMTD3NneC1jb29yZGluYXRvcjCC
AaIwDQYJKoZIhvcNAQEBBQADggGPADCCAYoCggGBANwjjvzxyUBNuQYDuboFDgFu
qtOCkuCK+JZd6+ITzaI473YCNP8SLjL0nJFV//ofzUl+IvErSZT55E97DKi1I4gu
tJK72eQfEbgd6BFJ+kHqu3uAKbjNGyrAOs6MgcKZNzINYSlA5fk9c4oX1nV/4sOc
8fx4232pjeRnUwiDc0ZSF/RBNOErnUHbYdHBoVhDXjMLb2JZGsmPFD6FapFqOJCF
3rfXEUOlkOzsdjbUXnTjXVLKv3u6yqOvetJhGVdq9/iLLnz6U4gTtcuUOimWS9eP
ArWYR883vHPsctBfaqsBkv4HcAQTvhQrS4FdhF/DKjw61kFfIVjZlsZbLZvIAqbT
HhqFxebUPMMXIRSxuaxXiQbxesZXsHjkoaOW8Xly2dlOdW57FPCzxHhivggSYMzf
7daIqJ0E9Jl2OIHCZieVo5KGsjDmR6gSp4MVqf7wYhvucPzcZqNHaVuOH23BrlQ4
k/UovQ9IRobES4i5pCJifS65DBcib4ryPX+KNOZJcQIDAQABo1MwUTAOBgNVHQ8B
Af8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUj1IXixXdVSBivOfF
LkmKzHuNvlQwDwYDVR0RBAgwBocEfwAAATANBgkqhkiG9w0BAQwFAAOCAYEAIUrE
O/Q13nDHE12zl1pnY1smqBRRAIpHpIJPRNJvAnbi5REMk1JisJepTZRq5dbuZK0m
PNEjCIagl9mmnO73dEyCaEOz7OQOaQ9yPTpwAk9DkXuNGX2BzhLYqzH7apeLyEyD
SEaIEHyhcPUAkmjWqxWLrgM0dL5LmXR0yKLuzbw6sDKfWWQFQRg1wOqvJs1B/oE0
xXc/NNXJu2BhU+VTPhGqa/Vvd7nCkr4aVSiVr8q7dWM3GKAA4ZvxLoRv0NJyETmn
WQjpFVscMRBKZp/QbpaGPv71K8ZyqxvO8GTMS6g5t5s7O5ZgJeafxftgVeFZC+6o
4cOdHScy5GiqDvuHfybhQ7B/9U7XNvrPXuA9zhghO7FB5axp8KdXslhFc2rMUHC6
689h6LJZOpVsoUN+8qpzvcGOjlM/m4IIppnq2jKAx8aSCf05B/1yLn+KIa81wYap
emCoppSZz2o5Go8jmqJYBJJEv0lst+cGTuUErhx08DoADfUveAQkgzVdE9/z
-----END CERTIFICATE-----`
)

func TestTLSPrivatekey(t *testing.T) {
	t.Run("err", func(t *testing.T) {
		_, err := NewRSAPrikey(RSAPrikeyBits(123))
		require.Error(t, err)

		_, err = NewECDSAPrikey(ECDSACurve("123"))
		require.Error(t, err)
	})

	rsa2048, err := NewRSAPrikey(RSAPrikeyBits2048)
	require.NoError(t, err)
	rsa3072, err := NewRSAPrikey(RSAPrikeyBits3072)
	require.NoError(t, err)
	es224, err := NewECDSAPrikey(ECDSACurveP224)
	require.NoError(t, err)
	es256, err := NewECDSAPrikey(ECDSACurveP256)
	require.NoError(t, err)
	es384, err := NewECDSAPrikey(ECDSACurveP384)
	require.NoError(t, err)
	es521, err := NewECDSAPrikey(ECDSACurveP521)
	require.NoError(t, err)
	edkey, err := NewEd25519Prikey()
	require.NoError(t, err)

	for _, key := range []crypto.PrivateKey{
		rsa2048,
		rsa3072,
		es224,
		es256,
		es384,
		es521,
		edkey,
	} {
		if rsaPrikey, ok := key.(*rsa.PrivateKey); ok {
			prider := x509.MarshalPKCS1PrivateKey(rsaPrikey)
			pripem := PrikeyDer2Pem(prider)
			prider2, err := Pem2Der(pripem)
			require.NoError(t, err)
			require.Equal(t, prider, prider2)
			key2, err := RSADer2Prikey(prider)
			require.NoError(t, err)
			require.True(t, rsaPrikey.Equal(key2))
			key2, err = RSAPem2Prikey(pripem)
			require.NoError(t, err)
			require.True(t, rsaPrikey.Equal(key2))
		}

		der, err := Prikey2Der(key)
		require.NoError(t, err)

		pem, err := Prikey2Pem(key)
		require.NoError(t, err)

		der2, err := Pem2Der(pem)
		require.NoError(t, err)
		require.Equal(t, pem, PrikeyDer2Pem(der2))
		require.Equal(t, der, der2)
		der22, err := Pem2Der(pem)
		require.NoError(t, err)
		require.Equal(t, der, der22)

		ders, err := Pem2Ders(pem)
		require.NoError(t, err)
		require.Equal(t, pem, PrikeyDer2Pem(ders[0]))
		require.Equal(t, der, der2)

		key, err = Pem2Prikey(pem)
		require.NoError(t, err)
		der2, err = Prikey2Der(key)
		require.NoError(t, err)
		require.Equal(t, der, der2)

		key, err = Der2Prikey(der)
		require.NoError(t, err)
		der2, err = Prikey2Der(key)
		require.NoError(t, err)
		require.Equal(t, der, der2)

		require.NotNil(t, GetPubkeyFromPrikey(key))

		t.Run("cert", func(t *testing.T) {
			der, err := NewX509Cert(key,
				WithX509CertCommonName("laisky"),
				WithX509CertDNS([]string{"laisky"}),
				WithX509CertIsCA(),
				WithX509CertOrganization([]string{"laisky"}),
				WithX509CertValidFrom(time.Now()),
				WithX509CertValidFor(time.Second),
			)
			require.NoError(t, err)

			cert, err := Der2Cert(der)
			require.NoError(t, err)

			pem := Cert2Pem(cert)
			cert, err = Pem2Cert(pem)
			require.NoError(t, err)
			require.Equal(t, der, Cert2Der(cert))
		})
	}
}

func TestTLSPublickey(t *testing.T) {
	rsa2048, err := NewRSAPrikey(RSAPrikeyBits2048)
	require.NoError(t, err)
	rsa3072, err := NewRSAPrikey(RSAPrikeyBits3072)
	require.NoError(t, err)
	es224, err := NewECDSAPrikey(ECDSACurveP224)
	require.NoError(t, err)
	es256, err := NewECDSAPrikey(ECDSACurveP256)
	require.NoError(t, err)
	es384, err := NewECDSAPrikey(ECDSACurveP384)
	require.NoError(t, err)
	es521, err := NewECDSAPrikey(ECDSACurveP521)
	require.NoError(t, err)
	edkey, err := NewEd25519Prikey()
	require.NoError(t, err)

	_, err = Pubkey2Der(nil)
	require.Error(t, err)

	for _, key := range []crypto.PublicKey{
		GetPubkeyFromPrikey(rsa2048),
		GetPubkeyFromPrikey(rsa3072),
		GetPubkeyFromPrikey(es224),
		GetPubkeyFromPrikey(es256),
		GetPubkeyFromPrikey(es384),
		GetPubkeyFromPrikey(es521),
		GetPubkeyFromPrikey(edkey),
	} {
		require.NotNil(t, key)
		der, err := Pubkey2Der(key)
		require.NoError(t, err)

		pem, err := Pubkey2Pem(key)
		require.NoError(t, err)

		der2, err := Pem2Der(pem)
		require.NoError(t, err)
		require.Equal(t, pem, PubkeyDer2Pem(der2))
		require.Equal(t, der, der2)
		der22, err := Pem2Der(pem)
		require.NoError(t, err)
		require.Equal(t, der, der22)

		key, err = Pem2Pubkey(pem)
		require.NoError(t, err)
		der2, err = Pubkey2Der(key)
		require.NoError(t, err)
		require.Equal(t, der, der2)

		key, err = Der2Pubkey(der)
		require.NoError(t, err)
		der2, err = Pubkey2Der(key)
		require.NoError(t, err)
		require.Equal(t, der, der2)
	}
}

func TestPem2Der_multi_certs(t *testing.T) {
	der, err := Pem2Der([]byte(testCertChain))
	require.NoError(t, err)
	cs, err := Der2Certs(der)
	require.NoError(t, err)

	require.Equal(t, "sgx-coordinator-inter", cs[0].Subject.CommonName)
	require.Equal(t, "sgx-coordinator", cs[1].Subject.CommonName)
}

func TestSecureCipherSuites(t *testing.T) {
	raw := SecureCipherSuites(nil)
	filtered := SecureCipherSuites(func(cs *tls.CipherSuite) bool {
		return true
	})
	require.Equal(t, len(raw), len(filtered))

	filtered = SecureCipherSuites(func(cs *tls.CipherSuite) bool {
		return false
	})
	require.Zero(t, len(filtered))
}

func TestVerifyCertByPrikey(t *testing.T) {
	prikey, certDer, err := NewRSAPrikeyAndCert(RSAPrikeyBits3072)
	require.NoError(t, err)

	certPem := CertDer2Pem(certDer)

	err = VerifyCertByPrikey(certPem, prikey)
	require.NoError(t, err)

	t.Run("different cert", func(t *testing.T) {
		_, certDer2, err := NewRSAPrikeyAndCert(RSAPrikeyBits3072)
		require.NoError(t, err)
		certPem2 := CertDer2Pem(certDer2)
		err = VerifyCertByPrikey(certPem2, prikey)
		require.Error(t, err)
	})
}
