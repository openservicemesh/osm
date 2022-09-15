package certificate

import (
	"testing"
)

func TestIssueOptions_CommonName(t *testing.T) {
	tests := []struct {
		name         string
		trustDomain  string
		issueOption  []IssueOption
		want         CommonName
		wantCertType certType
	}{
		{
			name:         "ForServiceIdentity appends trust domain",
			trustDomain:  "cluster.local",
			issueOption:  []IssueOption{ForServiceIdentity("sa.ns")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: service,
		},
		{
			name:         "ForCommonName uses common name passed without appending trust domain",
			issueOption:  []IssueOption{ForCommonName("sa.ns.cluster.local")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: internal,
		},
		{
			name:         "ForCommonName uses common name passed without appending trust domain when trust domain is present",
			trustDomain:  "dont.use.com",
			issueOption:  []IssueOption{ForCommonName("sa.ns.cluster.local")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: internal,
		},
		{
			name:         "ForIngressGateway uses commonname passed without appending trust domain",
			trustDomain:  "dont.use.com",
			issueOption:  []IssueOption{ForIngressGateway("sa.ns.cluster.local")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: ingressGateway,
		},
		{
			name:         "ForIngressGateway uses commonname passed without appending trust domain",
			issueOption:  []IssueOption{ForIngressGateway("sa.ns.cluster.local")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: ingressGateway,
		},
		{
			name:         "ForCommonNamePrefix appends trust domain",
			trustDomain:  "cluster.local",
			issueOption:  []IssueOption{ForCommonNamePrefix("sa.ns")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: internal,
		},
		{
			name:         "withCommonNamePrefix appends trust domain",
			trustDomain:  "cluster.local",
			issueOption:  []IssueOption{withCommonNamePrefix("sa.ns")},
			want:         CommonName("sa.ns.cluster.local"),
			wantCertType: "",
		},
		{
			name:         "withFullCommonName uses name passed",
			trustDomain:  "cluster.local",
			issueOption:  []IssueOption{withFullCommonName(), withCommonNamePrefix("sa.ns.example.com")},
			want:         CommonName("sa.ns.example.com"),
			wantCertType: "",
		},
		{
			name:         "withCertType uses name passed",
			issueOption:  []IssueOption{withCertType(service)},
			want:         CommonName(""),
			wantCertType: service,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewCertOptions(tt.issueOption...)
			o.trustDomain = tt.trustDomain
			if got := o.CommonName(); got != tt.want {
				t.Errorf("IssueOptions.CommonName() = %v, want %v", got, tt.want)
			}

			if got := o.certType; got != tt.wantCertType {
				t.Errorf("IssueOptions.certType = %v, want %v", got, tt.wantCertType)
			}
		})
	}
}
