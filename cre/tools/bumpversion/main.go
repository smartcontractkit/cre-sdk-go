// Package main provides a CLI tool for computing and creating version tags
// for the EVM capability and SDK releases.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	evmTagPrefix = "capabilities/blockchain/evm/"
	sdkTagPrefix = "v"
)

// TagOutput represents the JSON output structure for CI integration.
type TagOutput struct {
	EVMTag    string `json:"evm_tag"`
	SDKTag    string `json:"sdk_tag"`
	EVMPushed bool   `json:"evm_pushed,omitempty"`
	SDKPushed bool   `json:"sdk_pushed,omitempty"`
}

func main() {
	push := flag.Bool("push", false, "Push the created tags to origin")
	dryRun := flag.Bool("dry-run", false, "Print the tags that would be created without creating them")
	outputJSON := flag.Bool("output-json", false, "Output results as JSON for CI integration")
	flag.Parse()

	evmTag, err := computeNextEVMTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing EVM tag: %v\n", err)
		os.Exit(1)
	}

	sdkTag, err := computeNextSDKTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing SDK tag: %v\n", err)
		os.Exit(1)
	}

	output := TagOutput{
		EVMTag: evmTag,
		SDKTag: sdkTag,
	}

	if *dryRun {
		if *outputJSON {
			printJSON(output)
		} else {
			fmt.Printf("Next EVM tag: %s\n", evmTag)
			fmt.Printf("Next SDK tag: %s\n", sdkTag)
			fmt.Println("Dry run mode - no tags created")
		}
		return
	}

	if err := createTag(evmTag); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating EVM tag: %v\n", err)
		os.Exit(1)
	}
	if !*outputJSON {
		fmt.Printf("Created tag: %s\n", evmTag)
	}

	if err := createTag(sdkTag); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating SDK tag: %v\n", err)
		os.Exit(1)
	}
	if !*outputJSON {
		fmt.Printf("Created tag: %s\n", sdkTag)
	}

	if *push {
		if err := pushTags(evmTag, sdkTag); err != nil {
			fmt.Fprintf(os.Stderr, "Error pushing tags: %v\n", err)
			os.Exit(1)
		}

		// Verify tags were pushed successfully
		if err := verifyTagExists(evmTag); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not verify EVM tag: %v\n", err)
		} else {
			output.EVMPushed = true
		}

		if err := verifyTagExists(sdkTag); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not verify SDK tag: %v\n", err)
		} else {
			output.SDKPushed = true
		}

		if !*outputJSON {
			fmt.Println("Tags pushed to origin")
			if output.EVMPushed {
				fmt.Printf("Verified: %s exists\n", evmTag)
			}
			if output.SDKPushed {
				fmt.Printf("Verified: %s exists\n", sdkTag)
			}
		}
	}

	if *outputJSON {
		printJSON(output)
	}
}

// printJSON outputs the TagOutput as JSON.
func printJSON(output TagOutput) {
	data, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// computeNextEVMTag finds the latest capabilities/blockchain/evm/v* beta tag
// and increments the beta number.
func computeNextEVMTag() (string, error) {
	tags, err := getTagsWithPrefix(evmTagPrefix)
	if err != nil {
		return "", err
	}

	// Filter to beta tags and find the latest
	betaRegex := regexp.MustCompile(`^capabilities/blockchain/evm/v(\d+)\.(\d+)\.(\d+)-beta\.(\d+)$`)
	var latestMajor, latestMinor, latestPatch, latestBeta int
	found := false

	for _, tag := range tags {
		matches := betaRegex.FindStringSubmatch(tag)
		if matches == nil {
			continue
		}

		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch, _ := strconv.Atoi(matches[3])
		beta, _ := strconv.Atoi(matches[4])

		if !found || compareBetaVersions(major, minor, patch, beta, latestMajor, latestMinor, latestPatch, latestBeta) > 0 {
			latestMajor, latestMinor, latestPatch, latestBeta = major, minor, patch, beta
			found = true
		}
	}

	if !found {
		// No beta tags found, start with v1.0.0-beta.0
		return fmt.Sprintf("%sv1.0.0-beta.0", evmTagPrefix), nil
	}

	// Increment beta number
	return fmt.Sprintf("%sv%d.%d.%d-beta.%d", evmTagPrefix, latestMajor, latestMinor, latestPatch, latestBeta+1), nil
}

// computeNextSDKTag finds the latest v* SDK tag and increments the patch version.
func computeNextSDKTag() (string, error) {
	tags, err := getTagsWithPrefix(sdkTagPrefix)
	if err != nil {
		return "", err
	}

	// Filter to stable version tags (not beta, not prefixed paths)
	versionRegex := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	var latestMajor, latestMinor, latestPatch int
	found := false

	for _, tag := range tags {
		matches := versionRegex.FindStringSubmatch(tag)
		if matches == nil {
			continue
		}

		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch, _ := strconv.Atoi(matches[3])

		if !found || compareVersions(major, minor, patch, latestMajor, latestMinor, latestPatch) > 0 {
			latestMajor, latestMinor, latestPatch = major, minor, patch
			found = true
		}
	}

	if !found {
		// No SDK tags found, start with v0.0.1
		return "v0.0.1", nil
	}

	// Increment patch version
	return fmt.Sprintf("v%d.%d.%d", latestMajor, latestMinor, latestPatch+1), nil
}

// getTagsWithPrefix returns all git tags that start with the given prefix.
func getTagsWithPrefix(prefix string) ([]string, error) {
	cmd := exec.Command("git", "tag", "-l", prefix+"*")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list git tags: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var tags []string
	for _, line := range lines {
		if line != "" {
			tags = append(tags, line)
		}
	}

	sort.Strings(tags)
	return tags, nil
}

// compareBetaVersions compares two beta versions, returning:
//
//	1 if a > b
//	-1 if a < b
//	0 if a == b
func compareBetaVersions(aMajor, aMinor, aPatch, aBeta, bMajor, bMinor, bPatch, bBeta int) int {
	if aMajor != bMajor {
		if aMajor > bMajor {
			return 1
		}
		return -1
	}
	if aMinor != bMinor {
		if aMinor > bMinor {
			return 1
		}
		return -1
	}
	if aPatch != bPatch {
		if aPatch > bPatch {
			return 1
		}
		return -1
	}
	if aBeta != bBeta {
		if aBeta > bBeta {
			return 1
		}
		return -1
	}
	return 0
}

// compareVersions compares two versions, returning:
//
//	1 if a > b
//	-1 if a < b
//	0 if a == b
func compareVersions(aMajor, aMinor, aPatch, bMajor, bMinor, bPatch int) int {
	if aMajor != bMajor {
		if aMajor > bMajor {
			return 1
		}
		return -1
	}
	if aMinor != bMinor {
		if aMinor > bMinor {
			return 1
		}
		return -1
	}
	if aPatch != bPatch {
		if aPatch > bPatch {
			return 1
		}
		return -1
	}
	return 0
}

// createTag creates a git tag at the current HEAD.
func createTag(tag string) error {
	cmd := exec.Command("git", "tag", tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushTags pushes the specified tags to origin.
func pushTags(tags ...string) error {
	args := append([]string{"push", "origin"}, tags...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// verifyTagExists checks if a tag exists in the local repository.
func verifyTagExists(tag string) error {
	cmd := exec.Command("git", "rev-parse", tag)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tag %s not found: %w", tag, err)
	}
	return nil
}
