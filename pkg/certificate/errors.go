package certificate

import (
	"errors"
)

var errEncodeKey = errors.New("encode key")
var errEncodeCert = errors.New("encode cert")
var errMarshalPrivateKey = errors.New("marshal private key")
var errNoPrivateKeyInPEM = errors.New("no private Key in PEM")

// ErrNoCertificateInPEM is the error for no certificate in PEM
var ErrNoCertificateInPEM = errors.New("no certificate in PEM")

// ErrInvalidMRCIntentCombination is the error that should be returned if the combination of MRC intents is invalid.
var ErrInvalidMRCIntentCombination = errors.New("invalid mrc intent combination")

// ErrNumMRCExceedsMaxSupported is the error that should be returned if there are more than 2 MRCs with active
// and/or passive intent in the mesh.
var ErrNumMRCExceedsMaxSupported = errors.New("found more than the max number of MRCs supported in the control plane namespace")

// ErrExpectedActiveMRC is the error that should be returned when there is only 1 MRC in the mesh and it does not
// have an active intent.
var ErrExpectedActiveMRC = errors.New("found single MRC with non active intent")

// ErrUnexpectedMRCIntent is the error that should be returned if the intent value is not passive or active.
// The MRC reconciler should only consider MRCs with passive or active intents for the validating and signing
// issuers.
var ErrUnexpectedMRCIntent = errors.New("found unexpected MRC intent. Expected passive or active")

// ErrUnexpectedNilMRC is the the error that should be returned if the MRC is nil.
var ErrUnexpectedNilMRC = errors.New("received nil MRC")

// ErrNoMRCsFound is the the error that should be returned if no MRCs were found in the control plane.
var ErrNoMRCsFound = errors.New("found no MRCs")

// All of the below errors should be returned by the StorageEngine for each described scenario. The errors may be
// wrapped

// ErrInvalidCertSecret is the error that should be returned if the secret is stored incorrectly in the underlying infra
var ErrInvalidCertSecret = errors.New("invalid secret for certificate")

// ErrSecretNotFound should be returned if the secret isn't present in the underlying infra, on a Get
var ErrSecretNotFound = errors.New("secret not found")
