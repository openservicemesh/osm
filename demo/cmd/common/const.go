package common

const (
	// Success is the string constant emitted at the end of the Bookbuyer/Bookthief logs when the test succeeded.
	Success = "MAESTRO! THIS TEST SUCCEEDED!"

	// Failure is the string constant emitted at the end of the Bookbuyer/Bookthief logs when the test failed.
	Failure = "MAESTRO, WE HAVE A PROBLEM! THIS TEST FAILED!"

	// BooksBoughtHeader is the header returned by the bookstore and observed by the bookbuyer.
	BooksBoughtHeader = "Booksbought"

	// IdentityHeader is the header returned by the bookstore and observed by the bookbuyer.
	IdentityHeader = "Identity"

	// BookstoreNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookstoreNamespaceEnvVar = "BOOKSTORE_NAMESPACE"

	// BookwarehouseNamespaceEnvVar is the environment variable for the Bookwarehouse namespace.
	BookwarehouseNamespaceEnvVar = "BOOKWAREHOUSE_NAMESPACE"

	// BookthiefExpectedResponseCodeEnvVar is the environment variable for Bookthief's expected HTTP response code
	BookthiefExpectedResponseCodeEnvVar = "BOOKTHIEF_EXPECTED_RESPONSE_CODE"

	// EnableEgressEnvVar is the environment variable to enable egress requests in the demo
	EnableEgressEnvVar = "ENABLE_EGRESS"
)
