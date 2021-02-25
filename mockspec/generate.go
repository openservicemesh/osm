package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	// mockRulesFile is the file corresponding to the rules used to generate mocks
	mockRulesFile = "mockspec/rules"

	// commentPrefix is the prefix used for comments in the rules file
	commentPrefix = "#"
)

func main() {
	rulesfile, err := os.Open(mockRulesFile)
	if err != nil {
		log.Fatal(err)
	}
	defer rulesfile.Close() //nolint: gosec,go-lint,errcheck

	scanner := bufio.NewScanner(rulesfile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, commentPrefix) || line == "" {
			continue
		}
		genMock(line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func genMock(ruleStr string) {
	ruleOptions := strings.Split(ruleStr, ";")
	for i := range ruleOptions {
		ruleOptions[i] = strings.TrimSpace(ruleOptions[i])
	}
	if len(ruleOptions) != 4 {
		log.Fatalf("Invalid syntax for mockgen rule: %v", ruleOptions)
	}

	packageName := ruleOptions[0]
	destinationPath := ruleOptions[1]
	selfPackagePath := ruleOptions[2]
	importPath := ruleOptions[2]
	interfaces := ruleOptions[3]

	// Generate the mocks
	cmdList := []string{
		"run", "github.com/golang/mock/mockgen",
		"-package", packageName,
		"-destination", destinationPath,
		"-self_package", selfPackagePath,
		importPath,
		interfaces,
	}
	cmd := exec.Command("go", cmdList...) // nolint gosec
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error generating mocks for rule: %v, err: %s", ruleOptions, err)
	}

	// Sort imports using goimports
	cmdList = []string{
		"run", "golang.org/x/tools/cmd/goimports",
		"-w", ruleOptions[1],
	}
	cmd = exec.Command("go", cmdList...) // nolint gosec
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error generating mocks for rule: %v, err: %s", ruleOptions, err)
	}
}
