// This file contains functions to abstract the usage of the mcl library.
// Ideally there should be a proper wrapper supporting the whole API of mcl but
// this is out-of-scope for now.

package cluster

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

// Mcl calls the mcl binary.
func Mcl(mciFilePath string, granularity int) error {
	cmd := exec.Command("mcl", mciFilePath, "-I", fmt.Sprintf("%f", float64(granularity)/10.0))
	err := run(cmd)
	return err
}

// McxLoadAbc calls the mcxload binary for an abc-file.
func McxLoadAbc(abcFilePath string, tabFilePath string, mciFilePath string) error {
	cmd := exec.Command("mcxload", "-abc", abcFilePath, "--stream-mirror", "-write-tab", tabFilePath, "-o", mciFilePath)
	err := run(cmd)
	return err
}

// McxDump calls the mcxdump binary.
func McxDump(clusterFileName string, tabFileName string, outFileName string, granularity int) error {
	cmd := exec.Command("mcxdump", "-icl", fmt.Sprintf("%s.I%d", clusterFileName, granularity), "-tabr", tabFileName, "-o", fmt.Sprintf("%s.I%d", outFileName, granularity))
	fmt.Println("mcxdump", "-icl", fmt.Sprintf("%s.I%d", clusterFileName, granularity), "-tabr", tabFileName, "-o", fmt.Sprintf("%s.I%d", outFileName, granularity))
	err := run(cmd)
	return err
}

// run wraps the call to cmd.Run with logging and error handling.
func run(cmd *exec.Cmd) error {
	bin := filepath.Base(cmd.Path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		msg := stderr.String()
		if len(msg) > 0 {
			fmt.Printf("\n%s:\nerr: %s", bin, stderr.String())
			return errors.New(fmt.Sprintf("clustering failed: %s", msg))
		}

		return errors.New(fmt.Sprintf("clustering failed: missing mcl binary \"%s\"", bin))
	}

	fmt.Printf("\n%s:\nout: %s", bin, stdout.String())
	return nil
}
