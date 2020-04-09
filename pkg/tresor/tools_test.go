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
		privKey, pemKey, err1 := privKeyFromFile("sample_private_key.pem")
		actual, err2 := encodeKey(privKey)
		It("AAA", func() {
			Expect(err1).ShouldNot(HaveOccurred())
			Expect(err2).ShouldNot(HaveOccurred())
			Expect(string(actual)).To(Equal(expectedPrivateKey))
			Expect([]byte(pemKey)).To(Equal([]byte{48, 130, 4, 189, 2, 1, 0, 48, 13, 6, 9, 42, 134, 72, 134, 247, 13, 1, 1, 1, 5, 0, 4, 130, 4, 167, 48, 130, 4, 163, 2, 1, 0, 2, 130, 1, 1, 0, 195, 223, 232, 42, 71, 155, 75, 171, 124, 54, 41, 147, 149, 8, 148, 90, 67, 111, 180, 109, 208, 230, 170, 245, 159, 225, 134, 99, 177, 137, 72, 89, 67, 18, 197, 3, 97, 95, 215, 230, 234, 239, 215, 253, 245, 27, 193, 83, 41, 12, 253, 237, 216, 14, 192, 174, 2, 79, 137, 169, 45, 139, 206, 187, 233, 107, 59, 186, 213, 91, 98, 45, 143, 190, 50, 223, 113, 91, 108, 51, 69, 1, 180, 10, 151, 124, 115, 118, 70, 15, 141, 123, 176, 125, 1, 169, 222, 165, 71, 137, 33, 252, 236, 251, 248, 235, 232, 192, 33, 186, 20, 217, 3, 244, 106, 186, 18, 219, 68, 98, 74, 174, 97, 236, 130, 113, 174, 3, 33, 167, 231, 205, 83, 94, 34, 249, 216, 117, 47, 108, 159, 61, 5, 183, 177, 165, 196, 45, 82, 122, 133, 82, 63, 9, 134, 113, 214, 41, 59, 168, 52, 247, 11, 148, 51, 17, 47, 151, 209, 95, 79, 25, 202, 28, 241, 139, 40, 58, 232, 2, 232, 101, 219, 135, 54, 238, 135, 87, 246, 116, 97, 113, 229, 197, 227, 202, 100, 34, 17, 173, 188, 121, 241, 174, 248, 171, 172, 216, 5, 53, 157, 90, 169, 10, 132, 161, 117, 192, 86, 22, 7, 8, 182, 166, 146, 15, 91, 101, 56, 237, 51, 149, 208, 139, 39, 90, 207, 104, 236, 59, 22, 144, 83, 41, 127, 222, 210, 164, 78, 239, 75, 44, 105, 61, 231, 205, 2, 3, 1, 0, 1, 2, 130, 1, 0, 78, 194, 137, 167, 246, 131, 11, 58, 57, 7, 206, 79, 249, 109, 41, 185, 225, 195, 216, 217, 15, 86, 177, 7, 114, 242, 76, 7, 106, 43, 185, 91, 171, 12, 177, 11, 90, 236, 30, 244, 75, 35, 133, 198, 39, 248, 177, 19, 175, 61, 250, 28, 216, 243, 149, 166, 98, 103, 121, 2, 253, 189, 105, 179, 69, 120, 72, 220, 39, 78, 71, 123, 234, 128, 160, 20, 24, 144, 154, 65, 67, 78, 28, 6, 230, 66, 180, 106, 170, 97, 54, 146, 181, 180, 142, 38, 175, 207, 229, 163, 206, 118, 213, 19, 188, 83, 159, 147, 33, 252, 160, 197, 98, 65, 181, 104, 124, 140, 142, 66, 183, 164, 198, 219, 66, 216, 83, 15, 90, 64, 200, 96, 152, 65, 85, 93, 160, 81, 202, 130, 33, 113, 203, 150, 106, 18, 71, 75, 6, 115, 151, 25, 102, 45, 56, 244, 232, 103, 254, 82, 255, 82, 158, 28, 50, 236, 178, 94, 243, 88, 189, 8, 213, 149, 159, 193, 52, 205, 111, 187, 238, 245, 22, 67, 95, 192, 92, 140, 161, 209, 242, 230, 66, 24, 172, 30, 206, 157, 255, 6, 213, 175, 209, 60, 24, 16, 50, 203, 51, 118, 214, 236, 232, 92, 212, 178, 91, 25, 240, 148, 105, 63, 43, 82, 16, 239, 162, 171, 69, 92, 59, 204, 132, 59, 138, 170, 196, 179, 39, 12, 181, 143, 46, 161, 111, 2, 145, 17, 217, 57, 139, 237, 1, 46, 78, 65, 2, 129, 129, 0, 248, 34, 89, 112, 101, 22, 226, 88, 86, 145, 92, 221, 56, 109, 72, 34, 22, 48, 80, 142, 64, 140, 43, 35, 228, 213, 191, 76, 175, 38, 134, 226, 66, 32, 196, 223, 88, 65, 24, 176, 192, 85, 174, 121, 165, 184, 167, 225, 242, 186, 52, 252, 164, 9, 120, 198, 142, 81, 108, 125, 251, 196, 163, 234, 128, 240, 144, 213, 124, 199, 246, 99, 232, 236, 204, 116, 56, 44, 175, 244, 102, 128, 94, 90, 184, 22, 168, 137, 83, 148, 249, 196, 94, 146, 161, 100, 160, 37, 50, 160, 84, 247, 205, 180, 98, 192, 205, 102, 151, 151, 240, 81, 99, 237, 125, 1, 48, 11, 214, 68, 145, 92, 170, 187, 112, 122, 73, 143, 2, 129, 129, 0, 202, 21, 118, 105, 171, 148, 15, 94, 77, 251, 8, 77, 97, 164, 222, 156, 179, 199, 10, 146, 247, 212, 226, 134, 170, 87, 137, 66, 251, 132, 229, 1, 168, 124, 38, 59, 115, 92, 220, 128, 108, 171, 70, 117, 95, 80, 246, 72, 44, 128, 48, 94, 105, 48, 88, 224, 198, 107, 182, 120, 97, 232, 149, 88, 222, 220, 6, 105, 16, 106, 234, 151, 47, 208, 137, 94, 105, 85, 97, 167, 183, 142, 192, 192, 56, 91, 144, 24, 74, 118, 122, 45, 168, 69, 99, 70, 5, 203, 49, 94, 18, 12, 179, 171, 6, 208, 148, 85, 254, 3, 218, 47, 93, 127, 131, 38, 80, 104, 51, 170, 103, 129, 202, 48, 126, 163, 114, 227, 2, 129, 128, 5, 108, 242, 217, 179, 76, 41, 204, 214, 175, 189, 1, 21, 79, 198, 105, 0, 101, 52, 13, 184, 57, 152, 99, 227, 136, 12, 243, 199, 76, 167, 92, 97, 39, 200, 70, 61, 238, 198, 116, 110, 240, 48, 173, 118, 67, 48, 96, 143, 103, 36, 235, 117, 70, 195, 190, 75, 180, 90, 19, 243, 34, 92, 151, 47, 20, 147, 134, 39, 129, 83, 208, 225, 113, 244, 18, 130, 123, 239, 168, 255, 104, 197, 39, 100, 169, 18, 44, 86, 136, 134, 97, 149, 211, 204, 245, 159, 78, 208, 233, 146, 146, 12, 140, 106, 48, 95, 13, 100, 57, 45, 71, 10, 81, 82, 15, 105, 150, 136, 171, 221, 37, 210, 145, 224, 166, 187, 223, 2, 129, 129, 0, 176, 92, 120, 194, 17, 222, 158, 102, 243, 241, 80, 54, 144, 47, 221, 163, 174, 117, 215, 241, 153, 94, 109, 239, 142, 187, 228, 107, 211, 172, 16, 92, 25, 25, 120, 24, 76, 62, 207, 165, 56, 177, 101, 69, 75, 209, 17, 142, 189, 95, 134, 86, 238, 192, 37, 224, 204, 233, 246, 14, 43, 140, 90, 194, 123, 132, 84, 7, 223, 47, 31, 218, 159, 253, 3, 213, 164, 97, 194, 95, 39, 159, 234, 242, 22, 125, 58, 77, 40, 183, 43, 59, 171, 110, 27, 12, 98, 68, 9, 170, 138, 96, 17, 113, 1, 250, 136, 106, 95, 204, 38, 223, 77, 94, 218, 43, 86, 227, 9, 171, 254, 183, 83, 168, 108, 236, 226, 119, 2, 129, 128, 3, 62, 108, 216, 17, 9, 67, 133, 121, 120, 61, 212, 219, 38, 231, 167, 27, 77, 4, 96, 61, 159, 25, 140, 240, 182, 101, 131, 205, 23, 118, 16, 95, 173, 106, 120, 226, 176, 143, 57, 107, 25, 17, 73, 72, 243, 77, 32, 203, 122, 158, 172, 93, 91, 103, 44, 41, 90, 64, 242, 116, 154, 215, 76, 51, 182, 100, 104, 40, 197, 11, 218, 65, 247, 64, 134, 54, 10, 120, 63, 187, 253, 143, 76, 252, 6, 156, 65, 32, 218, 96, 231, 151, 1, 212, 125, 127, 168, 73, 173, 255, 181, 72, 112, 6, 142, 166, 218, 81, 123, 145, 176, 138, 89, 37, 186, 10, 191, 187, 161, 28, 56, 70, 38, 199, 79, 255, 198}))
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
