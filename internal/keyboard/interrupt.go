package keyboard

import (
	"context"
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
// 注意：由于 Go 标准库的限制，无法真正实现非阻塞的标准输入读取
// 这里采用简化方案：不实际监听标准输入，而是在主循环中检查
func (im *InterruptMonitor) Start() {
	// 重置打断状态
	im.mu.Lock()
	im.interrupted = false
	im.mu.Unlock()

	// 注意：这里不再启动 goroutine 读取标准输入
	// 因为会与主程序的输入冲突
	// 打断功能暂时禁用，避免输入冲突
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
