package standard_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO this should be a tag on common
const commonVersion = "c2f203274b69ab67e2e1bdaaad86b7d60e883fb6"

func TestRunStandardTest(t *testing.T) {
	t.Parallel()

	require.NoError(t, os.MkdirAll(".tools", os.ModePerm))

	fmt.Printf("Downloading standard test version %s\n", commonVersion)
	cmd := exec.Command("go", "mod", "download", "-json", "github.com/smartcontractkit/chainlink-common@"+commonVersion)
	out, err := cmd.Output()
	require.NoError(t, err)

	var mod struct{ Dir string }
	require.NoError(t, json.Unmarshal(out, &mod))

	absDir, err := filepath.Abs(".tools")
	require.NoError(t, err)

	fmt.Println("Building standard tests")
	cmd = exec.Command("go", "test", "-c", "-o", absDir, ".")
	cmd.Dir = mod.Dir
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, out)

	fmt.Println("Verifying standard tests")
	cmd = exec.Command(absDir, "-")

}
