package ns

import "golang.org/x/sys/unix"

// Gettid returns thread id
func Gettid() (tid int) {
	return unix.Gettid()
}

// Setns set fd's netns
func Setns(fd int, nstype int) (err error) {
	return unix.Setns(fd, nstype)
}
