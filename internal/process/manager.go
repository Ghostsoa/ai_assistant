package process

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	ID       string
	Cmd      *exec.Cmd
	Stdin    io.WriteCloser
	Output   []string
	Mutex    sync.Mutex
	Done     bool
	ExitCode int
}

// Manager 进程管理器
type Manager struct {
	processes map[string]*ProcessInfo
	mutex     sync.Mutex
	counter   int
}

// NewManager 创建进程管理器
func NewManager() *Manager {
	m := &Manager{
		processes: make(map[string]*ProcessInfo),
	}
	// 自动启动持久Shell
	m.startPersistentShell()
	return m
}

// StartProcess 启动进程
func (pm *Manager) StartProcess(command string) (string, error) {
	pm.mutex.Lock()
	pm.counter++
	processID := fmt.Sprintf("%d", time.Now().Unix()*1000+int64(pm.counter))
	pm.mutex.Unlock()

	cmd := exec.Command("bash", "-c", command)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	process := &ProcessInfo{
		ID:     processID,
		Cmd:    cmd,
		Stdin:  stdin,
		Output: []string{},
	}

	pm.mutex.Lock()
	pm.processes[processID] = process
	pm.mutex.Unlock()

	if err := cmd.Start(); err != nil {
		return "", err
	}

	go pm.collectOutput(process, stdout, stderr)

	go func() {
		cmd.Wait()
		process.Mutex.Lock()
		process.Done = true
		if cmd.ProcessState != nil {
			process.ExitCode = cmd.ProcessState.ExitCode()
		}
		process.Mutex.Unlock()
	}()

	return processID, nil
}

func (pm *Manager) collectOutput(process *ProcessInfo, stdout, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for scanner.Scan() {
		line := scanner.Text()
		process.Mutex.Lock()
		process.Output = append(process.Output, line)
		if len(process.Output) > 1000 {
			process.Output = process.Output[len(process.Output)-1000:]
		}
		process.Mutex.Unlock()
	}
}

// SendInput 发送输入到进程
func (pm *Manager) SendInput(processID, input string) error {
	pm.mutex.Lock()
	process, exists := pm.processes[processID]
	pm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("进程不存在: %s", processID)
	}

	process.Mutex.Lock()
	defer process.Mutex.Unlock()

	if process.Done {
		return fmt.Errorf("进程已结束")
	}

	_, err := process.Stdin.Write([]byte(input))
	return err
}

// GetOutput 获取进程输出
func (pm *Manager) GetOutput(processID string) (string, string, error) {
	pm.mutex.Lock()
	process, exists := pm.processes[processID]
	pm.mutex.Unlock()

	if !exists {
		return "", "", fmt.Errorf("进程不存在: %s", processID)
	}

	process.Mutex.Lock()
	defer process.Mutex.Unlock()

	output := strings.Join(process.Output, "\n")
	process.Output = []string{}

	var status string
	if process.Done {
		status = fmt.Sprintf("已退出(code=%d)", process.ExitCode)
	} else {
		status = "运行中"
	}

	return output, status, nil
}

// KillProcess 终止进程
func (pm *Manager) KillProcess(processID string) error {
	pm.mutex.Lock()
	process, exists := pm.processes[processID]
	pm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("进程不存在: %s", processID)
	}

	if process.Cmd.Process != nil {
		return process.Cmd.Process.Kill()
	}
	return nil
}

// startPersistentShell 启动持久Shell（程序启动时自动调用）
func (pm *Manager) startPersistentShell() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-Command", "-")
	} else {
		// 使用非交互式bash，避免终端控制权冲突
		cmd = exec.Command("bash")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	process := &ProcessInfo{
		ID:     "__persistent__",
		Cmd:    cmd,
		Stdin:  stdin,
		Output: []string{},
	}

	pm.processes["__persistent__"] = process

	if err := cmd.Start(); err != nil {
		return
	}

	go pm.collectOutput(process, stdout, stderr)

	go func() {
		cmd.Wait()
		process.Mutex.Lock()
		process.Done = true
		if cmd.ProcessState != nil {
			process.ExitCode = cmd.ProcessState.ExitCode()
		}
		process.Mutex.Unlock()
	}()

	// 等待Shell初始化
	time.Sleep(500 * time.Millisecond)
	process.Mutex.Lock()
	process.Output = []string{}
	process.Mutex.Unlock()
}

// ExecuteInPersistentShell 在持久Shell中执行命令（保持状态）
func (pm *Manager) ExecuteInPersistentShell(command string) (string, error) {
	pm.mutex.Lock()
	process, exists := pm.processes["__persistent__"]
	pm.mutex.Unlock()

	if !exists {
		return "", fmt.Errorf("持久Shell未启动")
	}

	process.Mutex.Lock()
	if process.Done {
		process.Mutex.Unlock()
		return "", fmt.Errorf("Shell已退出")
	}

	// 清空之前的输出
	process.Output = []string{}
	process.Mutex.Unlock()

	// 生成唯一标记
	marker := fmt.Sprintf("__END_%d__", time.Now().UnixNano())

	// 发送命令
	var cmdLine string
	if runtime.GOOS == "windows" {
		cmdLine = fmt.Sprintf("%s; Write-Host '%s'\n", command, marker)
	} else {
		cmdLine = fmt.Sprintf("%s; echo '%s'\n", command, marker)
	}

	if _, err := process.Stdin.Write([]byte(cmdLine)); err != nil {
		return "", err
	}

	// 等待命令执行完成（最多5秒）
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		process.Mutex.Lock()
		output := strings.Join(process.Output, "\n")
		if strings.Contains(output, marker) {
			// 找到结束标记
			lines := strings.Split(output, "\n")
			var result []string
			for _, line := range lines {
				// 过滤掉标记和空行
				if !strings.Contains(line, marker) && !strings.HasPrefix(line, "__END_") {
					result = append(result, line)
				}
			}
			process.Output = []string{}
			process.Mutex.Unlock()
			return strings.TrimSpace(strings.Join(result, "\n")), nil
		}
		process.Mutex.Unlock()
	}

	// 超时，返回当前输出
	process.Mutex.Lock()
	output := strings.Join(process.Output, "\n")
	process.Output = []string{}
	process.Mutex.Unlock()

	return strings.TrimSpace(output), nil
}
