package util

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	NSBinary = "nsenter"
)

var (
	cmdTimeout = time.Minute // one minute by default
)

type NamespaceExecutor struct {
	ns string
}

func NewNamespaceExecutor(ns string) (*NamespaceExecutor, error) {
	ne := &NamespaceExecutor{
		ns: ns,
	}

	if ns == "" {
		return ne, nil
	}
	mntNS := filepath.Join(ns, "mnt")
	netNS := filepath.Join(ns, "net")
	if _, err := Execute(NSBinary, []string{"-V"}); err != nil {
		return nil, errors.Wrap(err, "cannot find nsenter for namespace switching")
	}
	if _, err := Execute(NSBinary, []string{"--mount=" + mntNS, "mount"}); err != nil {
		return nil, errors.Wrapf(err, "invalid mount namespace %v", mntNS)
	}
	if _, err := Execute(NSBinary, []string{"--net=" + netNS, "ip", "addr"}); err != nil {
		return nil, errors.Wrapf(err, "invalid net namespace %v", netNS)
	}
	return ne, nil
}

func (ne *NamespaceExecutor) prepareCommandArgs(name string, args []string) []string {
	cmdArgs := []string{
		"--mount=" + filepath.Join(ne.ns, "mnt"),
		"--net=" + filepath.Join(ne.ns, "net"),
		name,
	}
	return append(cmdArgs, args...)
}

func (ne *NamespaceExecutor) Execute(name string, args []string) (string, error) {
	if ne.ns == "" {
		return Execute(name, args)
	}
	return Execute(NSBinary, ne.prepareCommandArgs(name, args))
}

func (ne *NamespaceExecutor) ExecuteWithTimeout(timeout time.Duration, name string, args []string) (string, error) {
	if ne.ns == "" {
		return ExecuteWithTimeout(timeout, name, args)
	}
	return ExecuteWithTimeout(timeout, NSBinary, ne.prepareCommandArgs(name, args))
}

func (ne *NamespaceExecutor) ExecuteWithoutTimeout(name string, args []string) (string, error) {
	if ne.ns == "" {
		return ExecuteWithoutTimeout(name, args)
	}
	return ExecuteWithoutTimeout(NSBinary, ne.prepareCommandArgs(name, args))
}

func Execute(binary string, args []string) (string, error) {
	return ExecuteWithTimeout(cmdTimeout, binary, args)
}

func ExecuteWithTimeout(timeout time.Duration, binary string, args []string) (string, error) {
	var err error
	cmd := exec.Command(binary, args...)
	done := make(chan struct{})

	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	go func() {
		err = cmd.Run()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				logrus.Warnf("Problem killing process pid=%v: %s", cmd.Process.Pid, err)
			}

		}
		return "", errors.Wrapf(err, "timeout executing: %v %v, output %s, stderr %s",
			binary, args, output.String(), stderr.String())
	}

	if err != nil {
		return "", errors.Wrapf(err, "failed to execute: %v %v, output %s, stderr %s",
			binary, args, output.String(), stderr.String())
	}
	return output.String(), nil
}

// TODO: Merge this with ExecuteWithTimeout

func ExecuteWithoutTimeout(binary string, args []string) (string, error) {
	var err error
	var output, stderr bytes.Buffer

	cmd := exec.Command(binary, args...)
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "failed to execute: %v %v, output %s, stderr %s",
			binary, args, output.String(), stderr.String())
	}
	return output.String(), nil
}

func (ne *NamespaceExecutor) ExecuteWithStdin(name string, args []string, stdinString string) (string, error) {
	if ne.ns == "" {
		return ExecuteWithStdin(name, args, stdinString)
	}
	cmdArgs := []string{
		"--mount=" + filepath.Join(ne.ns, "mnt"),
		"--net=" + filepath.Join(ne.ns, "net"),
		name,
	}
	cmdArgs = append(cmdArgs, args...)
	return ExecuteWithStdin(NSBinary, cmdArgs, stdinString)
}

func ExecuteWithStdin(binary string, args []string, stdinString string) (string, error) {
	var err error
	cmd := exec.Command(binary, args...)
	done := make(chan struct{})

	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, stdinString)
	}()

	go func() {
		err = cmd.Run()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(cmdTimeout):
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				logrus.Warnf("Problem killing process pid=%v: %s", cmd.Process.Pid, err)
			}

		}
		return "", errors.Wrapf(err, "timeout executing: %v %v, output %s, stderr %s",
			binary, args, output.String(), stderr.String())
	}

	if err != nil {
		return "", errors.Wrapf(err, "failed to execute: %v %v, output %s, stderr %s",
			binary, args, output.String(), stderr.String())
	}
	return output.String(), nil
}
