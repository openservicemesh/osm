package certificate

import (
	"fmt"
)

// IssueOption is an option that can be passed to IssueCertificate.
type IssueOption func(*issueOptions)

type issueOptions struct {
	fullCNProvided bool
}

func (o *issueOptions) formatCN(prefix, trustDomain string) CommonName {
	if o.fullCNProvided {
		return CommonName(prefix)
	}
	return CommonName(fmt.Sprintf("%s.%s", prefix, trustDomain))
}

// FullCNProvided tells IssueCertificate that the provided prefix is actually the full trust domain, and not to append
// the issuer's trust domain.
func FullCNProvided() IssueOption {
	return func(opts *issueOptions) {
		opts.fullCNProvided = true
	}
}
