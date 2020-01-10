package azure

import "errors"

var errUnableToObtainArmAuth = errors.New("unable to obtain ARM authorizer")
var errIncorrectAzureURI = errors.New("incorrect Azure URI")
