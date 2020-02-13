package tresor

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	expectedPrivateKey = "-----BEGIN PRIVATE KEY-----\n" +
		"MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDD3+gqR5tLq3w2\n" +
		"KZOVCJRaQ2+0bdDmqvWf4YZjsYlIWUMSxQNhX9fm6u/X/fUbwVMpDP3t2A7ArgJP\n" +
		"iakti8676Ws7utVbYi2PvjLfcVtsM0UBtAqXfHN2Rg+Ne7B9AanepUeJIfzs+/jr\n" +
		"6MAhuhTZA/RquhLbRGJKrmHsgnGuAyGn581TXiL52HUvbJ89BbexpcQtUnqFUj8J\n" +
		"hnHWKTuoNPcLlDMRL5fRX08Zyhzxiyg66ALoZduHNu6HV/Z0YXHlxePKZCIRrbx5\n" +
		"8a74q6zYBTWdWqkKhKF1wFYWBwi2ppIPW2U47TOV0IsnWs9o7DsWkFMpf97SpE7v\n" +
		"SyxpPefNAgMBAAECggEATsKJp/aDCzo5B85P+W0pueHD2NkPVrEHcvJMB2oruVur\n" +
		"DLELWuwe9EsjhcYn+LETrz36HNjzlaZiZ3kC/b1ps0V4SNwnTkd76oCgFBiQmkFD\n" +
		"ThwG5kK0aqphNpK1tI4mr8/lo8521RO8U5+TIfygxWJBtWh8jI5Ct6TG20LYUw9a\n" +
		"QMhgmEFVXaBRyoIhccuWahJHSwZzlxlmLTj06Gf+Uv9Snhwy7LJe81i9CNWVn8E0\n" +
		"zW+77vUWQ1/AXIyh0fLmQhisHs6d/wbVr9E8GBAyyzN21uzoXNSyWxnwlGk/K1IQ\n" +
		"76KrRVw7zIQ7iqrEsycMtY8uoW8CkRHZOYvtAS5OQQKBgQD4IllwZRbiWFaRXN04\n" +
		"bUgiFjBQjkCMKyPk1b9MryaG4kIgxN9YQRiwwFWueaW4p+HyujT8pAl4xo5RbH37\n" +
		"xKPqgPCQ1XzH9mPo7Mx0OCyv9GaAXlq4FqiJU5T5xF6SoWSgJTKgVPfNtGLAzWaX\n" +
		"l/BRY+19ATAL1kSRXKq7cHpJjwKBgQDKFXZpq5QPXk37CE1hpN6cs8cKkvfU4oaq\n" +
		"V4lC+4TlAah8JjtzXNyAbKtGdV9Q9kgsgDBeaTBY4MZrtnhh6JVY3twGaRBq6pcv\n" +
		"0IleaVVhp7eOwMA4W5AYSnZ6LahFY0YFyzFeEgyzqwbQlFX+A9ovXX+DJlBoM6pn\n" +
		"gcowfqNy4wKBgAVs8tmzTCnM1q+9ARVPxmkAZTQNuDmYY+OIDPPHTKdcYSfIRj3u\n" +
		"xnRu8DCtdkMwYI9nJOt1RsO+S7RaE/MiXJcvFJOGJ4FT0OFx9BKCe++o/2jFJ2Sp\n" +
		"EixWiIZhldPM9Z9O0OmSkgyMajBfDWQ5LUcKUVIPaZaIq90l0pHgprvfAoGBALBc\n" +
		"eMIR3p5m8/FQNpAv3aOuddfxmV5t74675GvTrBBcGRl4GEw+z6U4sWVFS9ERjr1f\n" +
		"hlbuwCXgzOn2DiuMWsJ7hFQH3y8f2p/9A9WkYcJfJ5/q8hZ9Ok0otys7q24bDGJE\n" +
		"CaqKYBFxAfqIal/MJt9NXtorVuMJq/63U6hs7OJ3AoGAAz5s2BEJQ4V5eD3U2ybn\n" +
		"pxtNBGA9nxmM8LZlg80XdhBfrWp44rCPOWsZEUlI800gy3qerF1bZywpWkDydJrX\n" +
		"TDO2ZGgoxQvaQfdAhjYKeD+7/Y9M/AacQSDaYOeXAdR9f6hJrf+1SHAGjqbaUXuR\n" +
		"sIpZJboKv7uhHDhGJsdP/8Y=\n" +
		"-----END PRIVATE KEY-----\n"
)

var _ = Describe("Test Tresor Tools", func() {
	Context("Test encodeCert function", func() {
		cert, err := encodeCert([]byte{1, 2, 3})
		It("AAA", func() {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cert).NotTo(Equal(nil))
		})
	})

	Context("Test encodeKey function", func() {
		privKey, err1 := privKeyFromFile("sample_private_key.pem")
		actual, err2 := encodeKey(privKey)
		It("AAA", func() {
			Expect(err1).ShouldNot(HaveOccurred())
			Expect(err2).ShouldNot(HaveOccurred())
			Expect(string(actual)).To(Equal(expectedPrivateKey))
		})
	})

	Context("Test makeTemplate function", func() {
		expectedTemplate := &x509.Certificate{
			Subject: pkix.Name{
				Organization: []string{"contoso"},
				CommonName:   "www.contoso.com",
			},
		}
		host := "www.contoso.com"
		org := "contoso"
		validity := 3 * time.Second
		actual, err := makeTemplate(host, org, validity)
		It("should have created a template", func() {
			Expect(err).ShouldNot(HaveOccurred())
			Expect(actual.NotAfter.Sub(actual.NotBefore)).To(Equal(validity))
			Expect(actual.Subject).To(Equal(expectedTemplate.Subject))
		})
	})
})
