package certificate

import (
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	tassert "github.com/stretchr/testify/assert"
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

func TestIssueOptions_SpiffeUrl(t *testing.T) {
	tests := []struct {
		name          string
		trustDomain   string
		issueOption   []IssueOption
		spiffeEnabled bool
		want          string
	}{
		{
			name:          "should return spiffe id composed with spiffe id",
			trustDomain:   "cluster.local",
			issueOption:   []IssueOption{ForServiceIdentity("sa.ns")},
			spiffeEnabled: true,
			want:          "spiffe://cluster.local/sa/ns",
		},
		{
			name:          "should return spiffe id composed with spiffe id for simple commonname",
			trustDomain:   "cluster.local",
			issueOption:   []IssueOption{ForCommonNamePrefix("ads")},
			spiffeEnabled: true,
			want:          "spiffe://cluster.local/ads",
		},
		{
			name:          "should return spiffe id composed with spiffe id for ingress gateway",
			trustDomain:   "cluster.local",
			issueOption:   []IssueOption{ForIngressGateway("ingress.svc.cluster.local")},
			spiffeEnabled: true,
			want:          "spiffe://cluster.local/ingress/svc",
		},
		{
			name:          "should return empty if spiffe id not enabled",
			trustDomain:   "cluster.local",
			issueOption:   []IssueOption{ForCommonNamePrefix("test.test")},
			spiffeEnabled: false,
			want:          "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)

			if tt.spiffeEnabled {
				tt.issueOption = append(tt.issueOption, withSpiffeEnabled())
			}

			o := NewCertOptions(tt.issueOption...)

			o.trustDomain = tt.trustDomain
			o.spiffeEnabled = tt.spiffeEnabled

			assert.Equal(tt.want, o.URISAN().String())

			// validate it is a valid SPIFFE ID using spiffe package which runs validation checks
			id, err := spiffeid.FromURI(o.URISAN())
			if tt.spiffeEnabled {
				assert.Nil(err)
				assert.Equal(tt.want, id.String())
			} else {
				assert.ErrorContains(err, "cannot be empty")
			}
		})
	}
}
