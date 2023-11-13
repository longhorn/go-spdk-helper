package target

import (
	"context"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-spdk-helper/pkg/spdk/client"
)

const (
	SPDKScriptsDir  = "scripts"
	SPDKSetupScript = "setup.sh"
	SPDKTGTBinary   = "build/bin/spdk_tgt"
)

func SetupTarget(spdkDir string, execute func(name string, args []string) (string, error)) (err error) {
	setupScriptPath := filepath.Join(spdkDir, SPDKScriptsDir, SPDKSetupScript)
	setupOpts := []string{
		"-c",
		setupScriptPath,
	}

	resetOpts := []string{
		"-c",
		setupScriptPath,
		"reset",
	}

	if _, err := execute("sh", resetOpts); err != nil {
		return err
	}
	if _, err := execute("sh", setupOpts); err != nil {
		return err
	}

	return nil
}

func StartTarget(spdkDir string, execute func(name string, args []string) (string, error)) (err error) {
	if spdkCli, err := client.NewClient(context.Background()); err == nil {
		if _, err := spdkCli.BdevGetBdevs("", 0); err == nil {
			logrus.Info("Detected running spdk_tgt, skipped the target starting")
			return nil
		}
	}

	tgtOpts := []string{
		"-c",
		filepath.Join(spdkDir, SPDKTGTBinary),
	}

	_, err = execute("sh", tgtOpts)
	return err
}
