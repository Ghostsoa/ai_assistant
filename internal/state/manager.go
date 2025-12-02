package state

import (
	"encoding/base64"
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

// TerminalSlot 终端槽位
type TerminalSlot struct {
	MachineID string   `json:"machine_id"` // 当前机器ID
	Active    bool     `json:"active"`     // 是否激活
	Buffer    []string `json:"-"`          // 终端快照（不持久化）
}

// State 全局状态
type State struct {
	Machines      map[string]*Machine      `json:"machines"`
	TerminalSlots map[string]*TerminalSlot `json:"terminal_slots"` // "slot1", "slot2"
}

// Manager 状态管理器
type Manager struct {
	state     *State
	stateFile string
	mutex     sync.RWMutex
}

// NewManager 创建状态管理器
func NewManager() *Manager {
	stateFile := filepath.Join(appconfig.ConfigDir, "state.json")

	m := &Manager{
		state: &State{
			Machines:      make(map[string]*Machine),
			TerminalSlots: make(map[string]*TerminalSlot),
		},
		stateFile: stateFile,
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

	// 初始化双slot终端（默认只激活slot1-本地）
	needsSave := false
	if len(m.state.TerminalSlots) == 0 {
		m.state.TerminalSlots["slot1"] = &TerminalSlot{
			MachineID: "local",
			Active:    true,
			Buffer:    []string{},
		}
		m.state.TerminalSlots["slot2"] = &TerminalSlot{
			MachineID: "",
			Active:    false,
			Buffer:    []string{},
		}
		needsSave = true
	}

	// 确保Buffer已初始化
	for _, slot := range m.state.TerminalSlots {
		if slot.Buffer == nil {
			slot.Buffer = []string{}
		}
	}

	// 如果初始化了新字段，立即保存
	if needsSave {
		m.Save()
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

	// 确保map已初始化（兼容旧版本state.json）
	if m.state.Machines == nil {
		m.state.Machines = make(map[string]*Machine)
	}
	if m.state.TerminalSlots == nil {
		m.state.TerminalSlots = make(map[string]*TerminalSlot)
	}

	// 初始化运行时字段
	for _, machine := range m.state.Machines {
		if machine.CurrentDir == "" {
			machine.CurrentDir = "/"
		}
	}
}

// Save 保存状态（调用者需要确保已加锁或数据一致性）
func (m *Manager) Save() error {
	// 不加锁，由调用者负责
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile, data, 0644)
}

// GetSlot1Machine 获取slot1的机器（默认主机器）
func (m *Manager) GetSlot1Machine() *Machine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	slot1 := m.state.TerminalSlots["slot1"]
	if slot1 == nil || !slot1.Active {
		return m.state.Machines["local"]
	}
	return m.state.Machines[slot1.MachineID]
}

// GetMachineForSlot 获取指定slot的机器
func (m *Manager) GetMachineForSlot(slotID string) *Machine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	slot := m.state.TerminalSlots[slotID]
	if slot == nil || !slot.Active {
		return nil
	}
	return m.state.Machines[slot.MachineID]
}

// GetMachine 获取指定机器
func (m *Manager) GetMachine(machineID string) *Machine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state.Machines[machineID]
}

// OpenTerminalSlot 打开终端槽位
func (m *Manager) OpenTerminalSlot(slotID, machineID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	slot := m.state.TerminalSlots[slotID]
	if slot == nil {
		return fmt.Errorf("槽位不存在: %s", slotID)
	}

	if slot.Active {
		return fmt.Errorf("槽位已激活，当前机器: %s", slot.MachineID)
	}

	slot.MachineID = machineID
	slot.Active = true
	slot.Buffer = []string{}

	return m.Save()
}

// CloseTerminalSlot 关闭终端槽位
func (m *Manager) CloseTerminalSlot(slotID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	slot := m.state.TerminalSlots[slotID]
	if slot == nil {
		return fmt.Errorf("槽位不存在: %s", slotID)
	}

	slot.Active = false
	slot.MachineID = ""
	slot.Buffer = []string{}

	return m.Save()
}

// SwitchTerminalSlot 切换槽位到另一个机器
func (m *Manager) SwitchTerminalSlot(slotID, machineID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	slot := m.state.TerminalSlots[slotID]
	if slot == nil {
		return fmt.Errorf("槽位不存在: %s", slotID)
	}

	slot.MachineID = machineID
	slot.Active = true
	slot.Buffer = []string{} // 清空旧buffer

	return m.Save()
}

// GetTerminalStatus 获取终端状态
func (m *Manager) GetTerminalStatus() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result string

	for _, slotID := range []string{"slot1", "slot2"} {
		slot := m.state.TerminalSlots[slotID]
		if slot == nil {
			continue
		}

		if slot.Active {
			machine := m.state.Machines[slot.MachineID]
			desc := slot.MachineID
			if machine != nil {
				desc = machine.Description
			}
			result += fmt.Sprintf("%s: [%s] ● 激活\n", slotID, desc)
		} else {
			result += fmt.Sprintf("%s: ○ 未激活\n", slotID)
		}
	}

	return result
}

// ListMachines 列出所有机器（标记slot中的机器）
func (m *Manager) ListMachines() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	slot1ID := m.state.TerminalSlots["slot1"].MachineID
	slot2ID := ""
	if m.state.TerminalSlots["slot2"].Active {
		slot2ID = m.state.TerminalSlots["slot2"].MachineID
	}

	var result string
	for id, machine := range m.state.Machines {
		marker := "○"
		if id == slot1ID {
			marker = "●1" // Slot 1
		} else if id == slot2ID {
			marker = "●2" // Slot 2
		}
		result += fmt.Sprintf("  %s %s (%s)\n", marker, machine.ID, machine.Description)
	}
	return result
}

// AppendTerminalOutput 追加终端输出到对应slot
func (m *Manager) AppendTerminalOutput(machineID, command, output string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	machine := m.state.Machines[machineID]
	if machine == nil {
		return
	}

	// 截断输出（最多15行）
	truncated := truncateTerminalOutput(output, 15)

	// 构建提示符（模拟真实终端）
	// 格式: root@hostname:~/dir#
	prompt := buildPrompt(machine)

	// 构建终端行（真实终端格式）
	var termLine string
	if truncated == "" {
		// 无输出时，只显示提示符和命令
		termLine = fmt.Sprintf("%s %s\n", prompt, command)
	} else {
		// 有输出时，显示提示符、命令和输出
		termLine = fmt.Sprintf("%s %s\n%s\n", prompt, command, truncated)
	}

	// 找到对应的slot并追加
	for _, slot := range m.state.TerminalSlots {
		if slot.Active && slot.MachineID == machineID {
			slot.Buffer = append(slot.Buffer, termLine)
			// 保留最近30条命令
			if len(slot.Buffer) > 30 {
				slot.Buffer = slot.Buffer[len(slot.Buffer)-30:]
			}
		}
	}
}

// buildPrompt 构建终端提示符
func buildPrompt(machine *Machine) string {
	// 使用机器的ID作为主机名（与sync工具的machine_id一致）
	machineName := machine.ID

	// 获取当前目录（简化显示，~代表home）
	dir := machine.CurrentDir
	if dir == "" {
		dir = "~"
	} else if strings.HasPrefix(dir, "/root") {
		dir = "~" + strings.TrimPrefix(dir, "/root")
	} else if strings.HasPrefix(dir, "/home/") {
		parts := strings.SplitN(strings.TrimPrefix(dir, "/home/"), "/", 2)
		if len(parts) == 2 {
			dir = "~" + parts[1]
		}
	}
	if dir == "" {
		dir = "~"
	}

	// 返回格式: root@机器名:dir#
	return fmt.Sprintf("root@%s:%s#", machineName, dir)
}

// truncateTerminalOutput 截断终端输出
func truncateTerminalOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		kept := strings.Join(lines[:maxLines], "\n")
		omitted := len(lines) - maxLines
		return fmt.Sprintf("%s\n... [省略 %d 行]", kept, omitted)
	}
	return output
}

// GetTerminalSnapshot 获取双slot终端快照
func (m *Manager) GetTerminalSnapshot() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result string

	// Slot 1
	slot1 := m.state.TerminalSlots["slot1"]
	if slot1 != nil && slot1.Active {
		machine := m.state.Machines[slot1.MachineID]
		machineDesc := slot1.MachineID
		if machine != nil {
			machineDesc = machine.Description
		}

		result += fmt.Sprintf("### Slot 1: [%s] ●\n", machineDesc)
		if len(slot1.Buffer) == 0 {
			result += "[终端为空]\n\n"
		} else {
			// 最多显示最近10条
			start := 0
			if len(slot1.Buffer) > 10 {
				start = len(slot1.Buffer) - 10
			}
			for i := start; i < len(slot1.Buffer); i++ {
				result += slot1.Buffer[i]
			}
			// 添加当前提示符（显示当前状态）
			result += buildPrompt(machine) + " \n\n"
		}
	}

	// Slot 2
	slot2 := m.state.TerminalSlots["slot2"]
	if slot2 != nil && slot2.Active {
		machine := m.state.Machines[slot2.MachineID]
		machineDesc := slot2.MachineID
		if machine != nil {
			machineDesc = machine.Description
		}

		result += fmt.Sprintf("### Slot 2: [%s]\n", machineDesc)
		if len(slot2.Buffer) == 0 {
			result += "[终端为空]\n\n"
		} else {
			// 最多显示最近10条
			start := 0
			if len(slot2.Buffer) > 10 {
				start = len(slot2.Buffer) - 10
			}
			for i := start; i < len(slot2.Buffer); i++ {
				result += slot2.Buffer[i]
			}
			// 添加当前提示符（显示当前状态）
			result += buildPrompt(machine) + " \n"
		}
	}

	if result == "" {
		return "[无激活的终端]"
	}

	return result
}

// generateAPIKey 已删除 - 现在使用全局硬编码密钥

// CallAgentAPI 调用寄生虫的通用API（支持多种action）
func (m *Manager) CallAgentAPI(machineID, action string, data map[string]interface{}) (map[string]interface{}, error) {
	m.mutex.RLock()
	machine := m.state.Machines[machineID]
	m.mutex.RUnlock()

	if machine == nil {
		return nil, fmt.Errorf("机器不存在: %s", machineID)
	}

	// 连接寄生虫
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", machine.Host, machine.Port), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("无法连接到寄生虫: %v", err)
	}
	defer conn.Close()

	// 构建请求
	request := map[string]interface{}{
		"action":  action,
		"api_key": appconfig.GlobalConfig.AgentAPIKey,
		"data":    data,
	}

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return nil, err
	}

	// 接收响应
	var response map[string]interface{}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, err
	}

	if success, ok := response["success"].(bool); !ok || !success {
		if errMsg, ok := response["error"].(string); ok {
			return nil, fmt.Errorf("%s", errMsg)
		}
		return nil, fmt.Errorf("操作失败")
	}

	return response, nil
}

// ExecuteOnAgent 在寄生虫上执行shell命令（使用execute action）
func (m *Manager) ExecuteOnAgent(machineID, command string) (string, error) {
	// 直接调用通用API
	resp, err := m.CallAgentAPI(machineID, "execute", map[string]interface{}{
		"command": command,
	})
	if err != nil {
		return "", err
	}

	// 调试日志
	fmt.Printf("[DEBUG] resp keys: %v\n", getKeys(resp))

	// 更新机器的当前工作目录
	if cwd, ok := resp["cwd"].(string); ok {
		m.mutex.Lock()
		if machine := m.state.Machines[machineID]; machine != nil {
			machine.CurrentDir = cwd
		}
		m.mutex.Unlock()
	}

	if output, ok := resp["output"].(string); ok {
		fmt.Printf("[DEBUG] output length: %d\n", len(output))
		return output, nil
	}
	fmt.Printf("[DEBUG] No output field in response!\n")
	return "", nil
}

// getKeys 获取map的所有key（用于调试）
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// UploadFile 上传文件到远程（自动分块）
func (m *Manager) UploadFile(machineID, remotePath string, content []byte) error {
	const chunkSize = 1024 * 1024 // 1MB分块
	totalSize := int64(len(content))

	for offset := int64(0); offset < totalSize; offset += chunkSize {
		end := offset + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := content[offset:end]
		encoded := base64.StdEncoding.EncodeToString(chunk)

		data := map[string]interface{}{
			"path":       remotePath,
			"content":    encoded,
			"offset":     offset,
			"total_size": totalSize,
		}

		_, err := m.CallAgentAPI(machineID, "upload", data)
		if err != nil {
			return fmt.Errorf("上传失败(offset %d): %v", offset, err)
		}
	}
	return nil
}

// DownloadFile 从远程下载文件（自动分块）
func (m *Manager) DownloadFile(machineID, remotePath string) ([]byte, error) {
	const chunkSize = 1024 * 1024 // 1MB分块
	var allContent []byte
	offset := int64(0)

	for {
		data := map[string]interface{}{
			"path":       remotePath,
			"offset":     offset,
			"chunk_size": chunkSize,
		}

		resp, err := m.CallAgentAPI(machineID, "download", data)
		if err != nil {
			return nil, fmt.Errorf("下载失败(offset %d): %v", offset, err)
		}

		contentB64, ok := resp["content"].(string)
		if !ok {
			return nil, fmt.Errorf("响应格式错误")
		}

		decoded, err := base64.StdEncoding.DecodeString(contentB64)
		if err != nil {
			return nil, fmt.Errorf("解码失败: %v", err)
		}

		allContent = append(allContent, decoded...)
		offset += int64(len(decoded))

		if eof, ok := resp["eof"].(bool); ok && eof {
			break
		}
	}
	return allContent, nil
}

// UploadFileChunk 上传单个文件块（供sync使用，支持进度回调）
func (m *Manager) UploadFileChunk(machineID, remotePath string, chunk []byte, offset, totalSize int64) (int64, error) {
	encoded := base64.StdEncoding.EncodeToString(chunk)

	data := map[string]interface{}{
		"path":       remotePath,
		"content":    encoded,
		"offset":     offset,
		"total_size": totalSize,
	}

	resp, err := m.CallAgentAPI(machineID, "upload", data)
	if err != nil {
		return 0, err
	}

	if uploaded, ok := resp["uploaded"].(float64); ok {
		return int64(uploaded), nil
	}
	return offset + int64(len(chunk)), nil
}

// DownloadFileChunk 下载单个文件块（供sync使用）
func (m *Manager) DownloadFileChunk(machineID, remotePath string, offset, chunkSize int64) ([]byte, bool, error) {
	data := map[string]interface{}{
		"path":       remotePath,
		"offset":     offset,
		"chunk_size": chunkSize,
	}

	resp, err := m.CallAgentAPI(machineID, "download", data)
	if err != nil {
		return nil, false, err
	}

	contentB64, ok := resp["content"].(string)
	if !ok {
		return nil, false, fmt.Errorf("响应格式错误")
	}

	decoded, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return nil, false, err
	}

	eof, _ := resp["eof"].(bool)
	return decoded, eof, nil
}

// GetState 获取状态快照
func (m *Manager) GetState() *State {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state
}

// InfectServer 寄生目标服务器
func (m *Manager) InfectServer(host, user, password, alias string) error {
	// 使用全局API Key
	apiKey := appconfig.GlobalConfig.AgentAPIKey

	// 调用infect.sh脚本，传递API Key
	cmd := exec.Command("bash", "./scripts/infect.sh", host, user, password, alias, apiKey)
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

				// 添加到控制机列表（不保存密钥，使用全局密钥）
				m.mutex.Lock()
				defer m.mutex.Unlock()

				m.state.Machines[machineID] = &Machine{
					ID:          machineID,
					Host:        machineHost,
					Port:        machinePort,
					Type:        "agent",
					Description: fmt.Sprintf("远程服务器 (%s)", host),
					// SecretKey 不再保存，统一使用全局配置
					CurrentDir: "/root",
				}

				// 保存状态
				return m.Save()
			}
		}
	}

	return fmt.Errorf("无法解析寄生结果")
}

// ...
