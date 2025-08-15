package standard_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const commonVersion = "b4b2af4fa578ab2ae6eda980605073dece26ad53"

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
	cmd.Dir = path.Join(mod.Dir, "pkg", "workflows", "wasm", "host")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	fmt.Println("Running standard tests")
	cmd = exec.Command(path.Join(absDir, "host.test"), "-test.v", "-test.run", "^TestStandard", "-path=impl")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Run())
}
