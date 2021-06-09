package rds

import (
	"fmt"
	"strings"
)

// return only those hostnames whose name ends with ":<port>"
func filterOnTargetPort(hostnames []string, port int) []string {
	newHostnames := make([]string, 0)
	toMatch := fmt.Sprintf(":%d", port)
	for _, name := range hostnames {
		if strings.HasSuffix(name, toMatch) {
			newHostnames = append(newHostnames, name)
		}
	}
	if len(newHostnames) == 0 {
		return joinTargetPort(hostnames, port)
	}
	return newHostnames
}

// join port on all hostnames
func joinTargetPort(hostnames []string, port int) []string {
	newHostnames := make([]string, 0)
	portStr := fmt.Sprintf(":%d", port)
	for _, name := range hostnames {
		if !strings.Contains(name, ":") {
			newHostname := name + portStr
			newHostnames = append(newHostnames, newHostname)
		}
	}
	return newHostnames
}

