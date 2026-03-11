//go:build windows

package manager

import (
	"os/exec"
	"syscall"
)

const detachedProcessFlag = 0x00000008

func configureDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcessFlag,
	}
}
