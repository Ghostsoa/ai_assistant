package process

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
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
	return &Manager{
		processes: make(map[string]*ProcessInfo),
	}
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
