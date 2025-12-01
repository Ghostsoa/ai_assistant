package keyboard

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// InterruptMonitor 打断监听器
type InterruptMonitor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	interrupted  bool
	mu           sync.Mutex
	interruptKey string
}

// NewInterruptMonitor 创建新的打断监听器
func NewInterruptMonitor(interruptKey string) *InterruptMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &InterruptMonitor{
		ctx:          ctx,
		cancel:       cancel,
		interruptKey: interruptKey,
	}
}

// Start 开始监听（在goroutine中运行）
func (im *InterruptMonitor) Start() {
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			select {
			case <-im.ctx.Done():
				return
			default:
				// 尝试读取一行输入
				input, err := reader.ReadString('\n')
				if err != nil {
					continue
				}

				input = strings.TrimSpace(input)
				// 检查是否是打断键
				if input == im.interruptKey {
					im.mu.Lock()
					im.interrupted = true
					im.mu.Unlock()
					fmt.Println("\n[打断] 操作已中止")
					return
				}
			}
		}
	}()
}

// Stop 停止监听
func (im *InterruptMonitor) Stop() {
	im.cancel()
}

// IsInterrupted 检查是否已被打断
func (im *InterruptMonitor) IsInterrupted() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.interrupted
}

// Reset 重置打断状态
func (im *InterruptMonitor) Reset() {
	im.mu.Lock()
	im.interrupted = false
	im.mu.Unlock()
}
