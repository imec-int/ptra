package cluster

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

func Mcl(mciFilePath string, granularity int) error {
	cmd := exec.Command("mcl", mciFilePath, "-I", fmt.Sprintf("%f", float64(granularity)/10.0))
	err := runCmd(cmd)
	return err
}

func McxLoadAbc(abcFilePath string, tabFilePath string, mciFilePath string) error {
	cmd := exec.Command("mcxload", "-abc", abcFilePath, "--stream-mirror", "-write-tab", tabFilePath, "-o", mciFilePath)
	err := runCmd(cmd)
	return err
}

func McxDump(clusterFileName string, tabFileName string, outFileName string, granularity int) error {
	cmd := exec.Command("mcxdump", "-icl", fmt.Sprintf("%s.I%d", clusterFileName, granularity), "-tabr", tabFileName, "-o", fmt.Sprintf("%s.I%d", outFileName, granularity))
	fmt.Println("mcxdump", "-icl", fmt.Sprintf("%s.I%d", clusterFileName, granularity), "-tabr", tabFileName, "-o", fmt.Sprintf("%s.I%d", outFileName, granularity))
	err := runCmd(cmd)
	return err
}

func runCmd(cmd *exec.Cmd) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Printf("\n%s:\nout: %s\nerr: %s", filepath.Base(cmd.Path), stdout.String(), stderr.String())
	return err
}
