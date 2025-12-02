package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"ai_assistant/internal/server"
)

// Send 发送请求到 JARVIS Server
func Send(query string) error {
	// 检查 Server 是否运行
	if !isServerRunning() {
		fmt.Fprintln(os.Stderr, "⚠️  JARVIS Server 未运行")
		fmt.Fprintln(os.Stderr, "正在启动...")

		// 尝试启动 Server
		if err := startServer(); err != nil {
			return fmt.Errorf("启动 Server 失败: %v", err)
		}

		// 等待启动
		time.Sleep(2 * time.Second)

		if !isServerRunning() {
			return fmt.Errorf("Server 启动失败")
		}
	}

	// 连接到 Server
	conn, err := net.Dial("unix", server.SocketPath)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	// 发送查询
	fmt.Fprintf(conn, "%s\n", query)

	// 接收流式响应
	reader := bufio.NewReader(conn)
	for {
		char, err := reader.ReadByte()
		if err != nil {
			break
		}
		fmt.Print(string(char))
	}

	return nil
}

// isServerRunning 检查 Server 是否运行
func isServerRunning() bool {
	conn, err := net.DialTimeout("unix", server.SocketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// startServer 启动 Server（后台）
func startServer() error {
	// 获取当前可执行文件路径
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// 后台启动
	cmd := exec.Command(executable, "--daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	// 不等待进程结束
	go cmd.Wait()

	return nil
}
