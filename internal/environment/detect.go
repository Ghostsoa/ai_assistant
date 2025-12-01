package environment

import (
	"os/exec"
	"runtime"
)

// SystemEnvironment 系统环境信息
type SystemEnvironment struct {
	OS            string
	PythonCommand string
	HasGrep       bool
	HasTree       bool
	HasGit        bool
	Shell         string
}

// Detect 检测系统环境
func Detect() SystemEnvironment {
	env := SystemEnvironment{
		OS: runtime.GOOS,
	}

	// 检测Shell
	if runtime.GOOS == "windows" {
		env.Shell = "powershell"
	} else {
		env.Shell = "bash"
	}

	// 检测Python命令
	if _, err := exec.Command("python3", "--version").CombinedOutput(); err == nil {
		env.PythonCommand = "python3"
	} else if _, err := exec.Command("python", "--version").CombinedOutput(); err == nil {
		env.PythonCommand = "python"
	} else {
		env.PythonCommand = "none"
	}

	// 检测grep
	if runtime.GOOS == "windows" {
		env.HasGrep = false // Windows默认用findstr
	} else {
		_, err := exec.Command("which", "grep").CombinedOutput()
		env.HasGrep = err == nil
	}

	// 检测tree
	_, err := exec.Command("which", "tree").CombinedOutput()
	if runtime.GOOS == "windows" {
		_, err = exec.Command("where", "tree").CombinedOutput()
	}
	env.HasTree = err == nil

	// 检测git
	_, err = exec.Command("git", "--version").CombinedOutput()
	env.HasGit = err == nil

	return env
}
