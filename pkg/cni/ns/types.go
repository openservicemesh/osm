// Package ns On Linux each OS thread can have a different network namespace.
// Go's thread scheduling model switches goroutines between OS threads based on OS thread load
// and whether the goroutine would block other goroutines. This can result in a goroutine
// switching network namespaces without notice and lead to errors in your code.
package ns

const (
	coneNewNet = 0x40000000

	// https://github.com/torvalds/linux/blob/master/include/uapi/linux/magic.h
	nsFsMagic      = 0x6e736673
	procSuperMagic = 0x9fa0

	// SoMark mark packets sent from a specific socket.
	SoMark = 0x24
)
