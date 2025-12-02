package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appconfig "ai_assistant/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// Machine 控制机信息
type Machine struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Type        string `json:"type"` // "local", "agent", "ssh"
	Description string `json:"description"`
	SecretKey   string `json:"secret_key"` // JWT密钥
	CurrentDir  string `json:"-"`          // 当前工作目录（不持久化）
}

// State 全局状态
type State struct {
	CurrentMachine string              `json:"current_machine"`
	Machines       map[string]*Machine `json:"machines"`
}

// Manager 状态管理器
type Manager struct {
	state          *State
	terminalBuffer []string
	stateFile      string
	mutex          sync.RWMutex
}

// NewManager 创建状态管理器
func NewManager() *Manager {
	stateFile := filepath.Join(appconfig.ConfigDir, "state.json")

	m := &Manager{
		state: &State{
			CurrentMachine: "local",
			Machines:       make(map[string]*Machine),
		},
		terminalBuffer: []string{},
		stateFile:      stateFile,
	}

	// 加载持久化状态
	m.load()

	// 确保本地机器存在
	if _, exists := m.state.Machines["local"]; !exists {
		m.state.Machines["local"] = &Machine{
			ID:          "local",
			Host:        "127.0.0.1",
			Port:        0,
			Type:        "local",
			Description: "本地机器",
			CurrentDir:  ".",
		}
	}

	return m
}

// load 加载状态
func (m *Manager) load() {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return // 文件不存在，使用默认状态
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	m.state = &state
	// 初始化运行时字段
	for _, machine := range m.state.Machines {
		if machine.CurrentDir == "" {
			machine.CurrentDir = "/"
		}
	}
}

// Save 保存状态
func (m *Manager) Save() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile, data, 0644)
}

// GetCurrentMachineID 获取当前控制机ID
func (m *Manager) GetCurrentMachineID() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state.CurrentMachine
}

// GetCurrentMachine 获取当前控制机
func (m *Manager) GetCurrentMachine() *Machine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state.Machines[m.state.CurrentMachine]
}

// GetCurrentDir 获取当前目录
func (m *Manager) GetCurrentDir() string {
	machine := m.GetCurrentMachine()
	if machine == nil {
		return "/"
	}
	return machine.CurrentDir
}

// SwitchMachine 切换控制机
func (m *Manager) SwitchMachine(machineID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.state.Machines[machineID]; !exists {
		return fmt.Errorf("机器不存在: %s", machineID)
	}

	m.state.CurrentMachine = machineID
	return m.Save()
}

// ListMachines 列出所有机器
func (m *Manager) ListMachines() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result string
	for id, machine := range m.state.Machines {
		marker := "○"
		if id == m.state.CurrentMachine {
			marker = "●"
		}
		result += fmt.Sprintf("  %s %s (%s)\n", marker, machine.ID, machine.Description)
	}
	return result
}

// AppendTerminalOutput 追加终端输出
func (m *Manager) AppendTerminalOutput(machineID, command, output string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	machine := m.state.Machines[machineID]
	if machine == nil {
		return
	}

	// 构建终端行（模拟真实终端）
	termLine := fmt.Sprintf("[%s] $ %s\n%s\n[%s:%s] $ ",
		machineID,
		command,
		output,
		machineID,
		machine.CurrentDir,
	)

	m.terminalBuffer = append(m.terminalBuffer, termLine)

	// 保留最近100行
	if len(m.terminalBuffer) > 100 {
		m.terminalBuffer = m.terminalBuffer[len(m.terminalBuffer)-100:]
	}
}

// GetTerminalSnapshot 获取终端快照
func (m *Manager) GetTerminalSnapshot(lines int) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.terminalBuffer) == 0 {
		return "[终端为空]"
	}

	// 返回最近N行
	start := len(m.terminalBuffer) - lines
	if start < 0 {
		start = 0
	}

	result := ""
	for i := start; i < len(m.terminalBuffer); i++ {
		result += m.terminalBuffer[i]
	}

	return result
}

// generateSecretKey 生成随机密钥
func generateSecretKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateJWT 生成JWT token
func (m *Manager) generateJWT(machineID string) (string, error) {
	machine := m.state.Machines[machineID]
	if machine == nil {
		return "", fmt.Errorf("机器不存在")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"machine_id": machineID,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(24 * time.Hour).Unix(), // 24小时过期
	})

	return token.SignedString([]byte(machine.SecretKey))
}

// ExecuteOnAgent 在寄生虫上执行命令
func (m *Manager) ExecuteOnAgent(machineID, command string) (string, error) {
	m.mutex.RLock()
	machine := m.state.Machines[machineID]
	m.mutex.RUnlock()

	if machine == nil {
		return "", fmt.Errorf("机器不存在: %s", machineID)
	}

	// 生成JWT token
	token, err := m.generateJWT(machineID)
	if err != nil {
		return "", fmt.Errorf("生成token失败: %v", err)
	}

	// 连接寄生虫
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", machine.Host, machine.Port), 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("无法连接到寄生虫: %v", err)
	}
	defer conn.Close()

	// 发送命令（带token）
	request := map[string]string{
		"command": command,
		"token":   token,
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return "", err
	}

	// 接收结果
	var response map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return "", err
	}

	if errMsg, ok := response["error"].(string); ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	output := response["output"].(string)

	// TODO: 更新当前目录（可以从寄生虫返回当前目录）

	return output, nil
}

// GetState 获取状态快照
func (m *Manager) GetState() *State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state
}

// InfectServer 寄生目标服务器
func (m *Manager) InfectServer(host, user, password, alias string) error {
	// 生成密钥
	secretKey := generateSecretKey()

	// 调用infect.sh脚本，传递密钥
	cmd := exec.Command("bash", "./scripts/infect.sh", host, user, password, alias, secretKey)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	// 解析脚本输出，获取机器信息
	// 格式: MACHINE_INFO:alias:host:port
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MACHINE_INFO:") {
			parts := strings.Split(line, ":")
			if len(parts) == 4 {
				machineID := parts[1]
				machineHost := parts[2]
				var machinePort int
				fmt.Sscanf(parts[3], "%d", &machinePort)

				// 添加到控制机列表（包含密钥）
				m.mutex.Lock()
				m.state.Machines[machineID] = &Machine{
					ID:          machineID,
					Host:        machineHost,
					Port:        machinePort,
					Type:        "agent",
					Description: fmt.Sprintf("远程服务器 (%s)", host),
					SecretKey:   secretKey,
					CurrentDir:  "/root",
				}
				m.mutex.Unlock()

				// 保存状态
				return m.Save()
			}
		}
	}

	return fmt.Errorf("无法解析寄生结果")
}
