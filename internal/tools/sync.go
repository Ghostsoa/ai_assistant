package tools

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai_assistant/internal/state"
)

// 同步任务管理
var (
	syncTasks      = make(map[string]*SyncTask)
	syncTasksMutex sync.RWMutex
)

// SyncTask 同步任务
type SyncTask struct {
	ID           string
	Direction    string
	LocalPath    string
	RemotePath   string
	Machine      string
	TotalSize    int64
	Transferred  int64
	Speed        float64 // bytes/sec
	StartTime    time.Time
	Status       string // "running", "completed", "failed"
	Error        string
	EstimatedETA int // 秒
}

// ExecuteSync 统一同步入口（push/pull/status）
func ExecuteSync(args map[string]interface{}, sm *state.Manager) string {
	action := args["action"].(string)

	switch action {
	case "push":
		localPath := args["local"].(string)
		remotePath := args["remote"].(string)
		remoteMachine := args["machine"].(string)
		return pushFileToRemote(localPath, remotePath, remoteMachine, sm)

	case "pull":
		localPath := args["local"].(string)
		remotePath := args["remote"].(string)
		remoteMachine := args["machine"].(string)
		return pullFileFromRemote(localPath, remotePath, remoteMachine, sm)

	case "status":
		return ExecuteSyncStatus(args)

	default:
		return fmt.Sprintf("[✗] 未知同步操作: %s", action)
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// pushFileToRemote 推送本地文件到远程（智能后台）
func pushFileToRemote(localPath, remotePath, remoteMachine string, sm *state.Manager) string {
	// 1. 获取文件信息
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Sprintf("[✗] 文件不存在: %v", err)
	}

	fileSize := info.Size()
	isDir := info.IsDir()

	// 2. 创建任务
	taskID := generateTaskID()
	task := &SyncTask{
		ID:          taskID,
		Direction:   "push",
		LocalPath:   localPath,
		RemotePath:  remotePath,
		Machine:     remoteMachine,
		TotalSize:   fileSize,
		Transferred: 0,
		StartTime:   time.Now(),
		Status:      "running",
	}

	syncTasksMutex.Lock()
	syncTasks[taskID] = task
	syncTasksMutex.Unlock()

	// 3. 开始传输，5秒内尝试完成
	done := make(chan string, 1)

	go func() {
		var result string
		if isDir {
			result = pushDirectorySync(task, sm)
		} else {
			result = pushFileSync(task, sm)
		}
		done <- result
	}()

	// 4. 等待5秒
	select {
	case result := <-done:
		// 5秒内完成
		syncTasksMutex.Lock()
		delete(syncTasks, taskID)
		syncTasksMutex.Unlock()
		return result

	case <-time.After(5 * time.Second):
		// 超过5秒，后台运行
		syncTasksMutex.RLock()
		speed := float64(task.Transferred) / 5.0 // bytes/sec
		eta := 0
		if speed > 0 {
			remaining := float64(task.TotalSize - task.Transferred)
			eta = int(remaining / speed)
		}
		task.Speed = speed
		task.EstimatedETA = eta
		syncTasksMutex.RUnlock()

		return fmt.Sprintf(`[⏳] 文件传输已后台运行

任务ID: %s
文件: %s -> %s:%s
大小: %.2f MB
已传输: %.2f MB (%.1f%%)
平均速度: %.2f KB/s
预计剩余: %d 秒

提示: 使用 sync_status({task_id: "%s"}) 查询进度`,
			taskID,
			localPath, remoteMachine, remotePath,
			float64(task.TotalSize)/(1024*1024),
			float64(task.Transferred)/(1024*1024),
			float64(task.Transferred)/float64(task.TotalSize)*100,
			speed/1024,
			eta,
			taskID,
		)
	}
}

// pushFileSync 同步推送文件（使用Manager封装）
func pushFileSync(task *SyncTask, sm *state.Manager) string {
	// 读取文件
	content, err := os.ReadFile(task.LocalPath)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 读取失败: %v", err)
	}

	fileSize := int64(len(content))
	const chunkSize = 1024 * 1024 // 1MB分块

	// 分块上传
	for offset := int64(0); offset < fileSize; offset += chunkSize {
		end := offset + chunkSize
		if end > fileSize {
			end = fileSize
		}

		chunk := content[offset:end]

		// 使用Manager封装方法
		uploaded, err := sm.UploadFileChunk(task.Machine, task.RemotePath, chunk, offset, fileSize)
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			return fmt.Sprintf("[✗] 上传失败: %v", err)
		}

		// 更新进度
		syncTasksMutex.Lock()
		task.Transferred = uploaded
		syncTasksMutex.Unlock()
	}

	task.Status = "completed"
	elapsed := time.Since(task.StartTime).Seconds()
	speed := float64(fileSize) / elapsed / 1024 // KB/s

	return fmt.Sprintf("[✓] 文件已推送: %s -> %s:%s\n大小: %.2f MB\n耗时: %.1f秒\n速度: %.2f KB/s",
		task.LocalPath, task.Machine, task.RemotePath,
		float64(fileSize)/(1024*1024), elapsed, speed)
}

// pushDirectorySync 同步推送目录（递归同步）
func pushDirectorySync(task *SyncTask, sm *state.Manager) string {
	// 1. 创建远程目录
	mkdirCmd := fmt.Sprintf("mkdir -p '%s'", task.RemotePath)
	_, err := sm.ExecuteOnAgent(task.Machine, mkdirCmd)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 创建远程目录失败: %v", err)
	}

	// 2. 递归遍历本地目录
	var files []string
	var totalSize int64

	err = filepath.Walk(task.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 遍历目录失败: %v", err)
	}

	// 更新总大小
	task.TotalSize = totalSize

	// 3. 并发同步文件（使用upload API）
	const maxConcurrency = 10 // 最大并发数
	var failed []string
	var failedMutex sync.Mutex
	var wg sync.WaitGroup

	fmt.Printf("[DEBUG] 开始同步 %d 个文件（并发数: %d）...\n", len(files), maxConcurrency)

	// 使用channel控制并发数
	semaphore := make(chan struct{}, maxConcurrency)

	for i, localFile := range files {
		wg.Add(1)

		go func(idx int, file string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 计算相对路径
			relPath, _ := filepath.Rel(task.LocalPath, file)
			// 转换为Unix路径（远程是Linux）
			relPath = strings.ReplaceAll(relPath, "\\", "/")
			remoteFile := task.RemotePath + "/" + relPath

			fmt.Printf("[DEBUG] [%d/%d] 同步: %s -> %s\n", idx+1, len(files), file, remoteFile)

			// 读取文件
			content, err := os.ReadFile(file)
			if err != nil {
				errMsg := fmt.Sprintf("读取失败: %v", err)
				failedMutex.Lock()
				failed = append(failed, file+": "+errMsg)
				failedMutex.Unlock()
				fmt.Printf("[DEBUG] %s: %s\n", file, errMsg)
				return
			}

			// 创建远程子目录（使用Unix路径）
			lastSlash := strings.LastIndex(remoteFile, "/")
			if lastSlash > 0 {
				remoteDir := remoteFile[:lastSlash]
				mkdirCmd := fmt.Sprintf("mkdir -p '%s'", remoteDir)
				_, mkdirErr := sm.ExecuteOnAgent(task.Machine, mkdirCmd)
				if mkdirErr != nil {
					fmt.Printf("[DEBUG] 创建目录失败 %s: %v\n", remoteDir, mkdirErr)
				}
			}

			// 使用upload API传输
			encoded := base64.StdEncoding.EncodeToString(content)
			data := map[string]interface{}{
				"path":       remoteFile,
				"content":    encoded,
				"offset":     0,
				"total_size": len(content),
			}

			_, err = sm.CallAgentAPI(task.Machine, "upload", data)
			if err != nil {
				errMsg := fmt.Sprintf("上传失败: %v", err)
				failedMutex.Lock()
				failed = append(failed, file+": "+errMsg)
				failedMutex.Unlock()
				fmt.Printf("[DEBUG] %s: %s\n", file, errMsg)
			} else {
				// 更新进度
				syncTasksMutex.Lock()
				task.Transferred += int64(len(content))
				syncTasksMutex.Unlock()
				fmt.Printf("[DEBUG] ✓ %s (%.2f KB)\n", file, float64(len(content))/1024)
			}
		}(i, localFile)
	}

	// 等待所有上传完成
	wg.Wait()

	fmt.Printf("[DEBUG] 同步完成，成功: %d, 失败: %d\n", len(files)-len(failed), len(failed))

	if len(failed) > 0 {
		task.Status = "failed"
		task.Error = fmt.Sprintf("%d 个文件失败", len(failed))
		return fmt.Sprintf("[✗] 目录同步部分失败:\n%s", strings.Join(failed, "\n"))
	}

	task.Status = "completed"
	elapsed := time.Since(task.StartTime).Seconds()

	return fmt.Sprintf("[✓] 目录已同步: %s -> %s:%s\n文件数: %d\n耗时: %.1f秒",
		task.LocalPath, task.Machine, task.RemotePath, len(files), elapsed)
}

// ExecuteSyncStatus 查询同步任务状态
func ExecuteSyncStatus(args map[string]interface{}) string {
	taskID := args["task_id"].(string)

	syncTasksMutex.RLock()
	task, exists := syncTasks[taskID]
	syncTasksMutex.RUnlock()

	if !exists {
		return fmt.Sprintf("[✗] 任务不存在: %s（可能已完成）", taskID)
	}

	elapsed := time.Since(task.StartTime).Seconds()
	progress := float64(task.Transferred) / float64(task.TotalSize) * 100

	if task.Status == "completed" {
		// 完成后自动清理
		syncTasksMutex.Lock()
		delete(syncTasks, taskID)
		syncTasksMutex.Unlock()

		return fmt.Sprintf(`[✓] 任务已完成

任务ID: %s
文件: %s -> %s:%s
大小: %.2f MB
耗时: %.1f 秒`,
			taskID,
			task.LocalPath, task.Machine, task.RemotePath,
			float64(task.TotalSize)/(1024*1024),
			elapsed,
		)
	}

	if task.Status == "failed" {
		syncTasksMutex.Lock()
		delete(syncTasks, taskID)
		syncTasksMutex.Unlock()

		return fmt.Sprintf("[✗] 任务失败: %s\n错误: %s", taskID, task.Error)
	}

	// 运行中
	speed := float64(task.Transferred) / elapsed
	eta := 0
	if speed > 0 {
		remaining := float64(task.TotalSize - task.Transferred)
		eta = int(remaining / speed)
	}

	return fmt.Sprintf(`[⏳] 任务进行中

任务ID: %s
文件: %s -> %s:%s
大小: %.2f MB
已传输: %.2f MB (%.1f%%)
平均速度: %.2f KB/s
已用时间: %.1f 秒
预计剩余: %d 秒`,
		taskID,
		task.LocalPath, task.Machine, task.RemotePath,
		float64(task.TotalSize)/(1024*1024),
		float64(task.Transferred)/(1024*1024),
		progress,
		speed/1024,
		elapsed,
		eta,
	)
}

// pullDirectorySync 同步拉取目录（递归同步）
func pullDirectorySync(task *SyncTask, sm *state.Manager) string {
	// 1. 创建本地目录
	err := os.MkdirAll(task.LocalPath, 0755)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 创建本地目录失败: %v", err)
	}

	// 2. 获取远程目录文件列表
	listCmd := fmt.Sprintf("find '%s' -type f 2>/dev/null", task.RemotePath)
	output, err := sm.ExecuteOnAgent(task.Machine, listCmd)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 列出远程文件失败: %v", err)
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	if len(files) == 0 || files[0] == "" {
		return "[i] 远程目录为空"
	}

	// 3. 逐个拉取文件（使用download API）
	var failed []string
	var totalSize int64

	for _, remoteFile := range files {
		remoteFile = strings.TrimSpace(remoteFile)
		if remoteFile == "" {
			continue
		}

		// 计算相对路径和本地路径
		relPath := strings.TrimPrefix(remoteFile, task.RemotePath)
		relPath = strings.TrimPrefix(relPath, "/")
		localFile := filepath.Join(task.LocalPath, relPath)

		// 创建本地子目录
		localDir := filepath.Dir(localFile)
		os.MkdirAll(localDir, 0755)

		// 使用download API下载
		data := map[string]interface{}{
			"path":       remoteFile,
			"offset":     0,
			"chunk_size": 1024 * 1024,
		}

		resp, err := sm.CallAgentAPI(task.Machine, "download", data)
		if err != nil {
			failed = append(failed, remoteFile+": "+err.Error())
			continue
		}

		// 解码内容
		contentB64, ok := resp["content"].(string)
		if !ok {
			failed = append(failed, remoteFile+": 响应格式错误")
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(contentB64)
		if err != nil {
			failed = append(failed, remoteFile+": "+err.Error())
			continue
		}

		// 写入本地文件
		err = os.WriteFile(localFile, decoded, 0644)
		if err != nil {
			failed = append(failed, remoteFile+": "+err.Error())
		} else {
			totalSize += int64(len(decoded))
			// 更新进度
			syncTasksMutex.Lock()
			task.Transferred += int64(len(decoded))
			syncTasksMutex.Unlock()
		}
	}

	task.TotalSize = totalSize

	if len(failed) > 0 {
		task.Status = "failed"
		task.Error = fmt.Sprintf("%d 个文件失败", len(failed))
		return fmt.Sprintf("[✗] 目录同步部分失败:\n%s", strings.Join(failed, "\n"))
	}

	task.Status = "completed"
	elapsed := time.Since(task.StartTime).Seconds()

	return fmt.Sprintf("[✓] 目录已拉取: %s:%s -> %s\n文件数: %d\n耗时: %.1f秒",
		task.Machine, task.RemotePath, task.LocalPath, len(files), elapsed)
}

// pullFileFromRemote 拉取远程文件/目录到本地（智能后台）
func pullFileFromRemote(localPath, remotePath, remoteMachine string, sm *state.Manager) string {
	// 1. 先检查远程是文件还是目录
	typeCmd := fmt.Sprintf("[ -d '%s' ] && echo 'DIR' || echo 'FILE'", remotePath)
	typeOutput, err := sm.ExecuteOnAgent(remoteMachine, typeCmd)
	if err != nil {
		return fmt.Sprintf("[✗] 检查远程路径失败: %v", err)
	}

	isDir := strings.Contains(typeOutput, "DIR")

	// 2. 获取大小
	var fileSize int64
	if !isDir {
		sizeCmd := fmt.Sprintf("stat -f%%z '%s' 2>/dev/null || stat -c%%s '%s' 2>/dev/null", remotePath, remotePath)
		sizeOutput, err := sm.ExecuteOnAgent(remoteMachine, sizeCmd)
		if err != nil {
			return fmt.Sprintf("[✗] 获取文件信息失败: %v", err)
		}
		fmt.Sscanf(strings.TrimSpace(sizeOutput), "%d", &fileSize)
	}

	// 2. 创建任务
	taskID := generateTaskID()
	task := &SyncTask{
		ID:          taskID,
		Direction:   "pull",
		LocalPath:   localPath,
		RemotePath:  remotePath,
		Machine:     remoteMachine,
		TotalSize:   fileSize,
		Transferred: 0,
		StartTime:   time.Now(),
		Status:      "running",
	}

	syncTasksMutex.Lock()
	syncTasks[taskID] = task
	syncTasksMutex.Unlock()

	// 3. 开始传输，5秒内尝试完成
	done := make(chan string, 1)

	go func() {
		var result string
		if isDir {
			result = pullDirectorySync(task, sm)
		} else {
			result = pullFileSync(task, sm)
		}
		done <- result
	}()

	// 4. 等待5秒
	select {
	case result := <-done:
		// 5秒内完成
		syncTasksMutex.Lock()
		delete(syncTasks, taskID)
		syncTasksMutex.Unlock()
		return result

	case <-time.After(5 * time.Second):
		// 超过5秒，后台运行
		syncTasksMutex.RLock()
		speed := float64(task.Transferred) / 5.0
		eta := 0
		if speed > 0 {
			remaining := float64(task.TotalSize - task.Transferred)
			eta = int(remaining / speed)
		}
		task.Speed = speed
		task.EstimatedETA = eta
		syncTasksMutex.RUnlock()

		return fmt.Sprintf(`[⏳] 文件传输已后台运行

任务ID: %s
文件: %s:%s -> %s
大小: %.2f MB
已传输: %.2f MB (%.1f%%)
平均速度: %.2f KB/s
预计剩余: %d 秒

提示: 使用 sync({action: "status", task_id: "%s"}) 查询进度`,
			taskID,
			remoteMachine, remotePath, localPath,
			float64(task.TotalSize)/(1024*1024),
			float64(task.Transferred)/(1024*1024),
			float64(task.Transferred)/float64(task.TotalSize)*100,
			speed/1024,
			eta,
			taskID,
		)
	}
}

// pullFileSync 同步拉取文件（使用Manager封装）
func pullFileSync(task *SyncTask, sm *state.Manager) string {
	const chunkSize = 1024 * 1024 // 1MB分块
	var allContent []byte
	offset := int64(0)

	// 分块下载
	for {
		chunk, eof, err := sm.DownloadFileChunk(task.Machine, task.RemotePath, offset, chunkSize)
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			return fmt.Sprintf("[✗] 下载失败: %v", err)
		}

		allContent = append(allContent, chunk...)
		offset += int64(len(chunk))

		// 更新进度
		syncTasksMutex.Lock()
		task.Transferred = offset
		syncTasksMutex.Unlock()

		if eof {
			break
		}
	}

	// 更新总大小
	task.TotalSize = int64(len(allContent))

	// 写入本地文件
	err := os.WriteFile(task.LocalPath, allContent, 0644)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return fmt.Sprintf("[✗] 写入本地文件失败: %v", err)
	}

	task.Status = "completed"
	elapsed := time.Since(task.StartTime).Seconds()
	speed := float64(len(allContent)) / elapsed / 1024 // KB/s

	return fmt.Sprintf("[✓] 文件已拉取: %s:%s -> %s\n大小: %.2f MB\n耗时: %.1f秒\n速度: %.2f KB/s",
		task.Machine, task.RemotePath, task.LocalPath,
		float64(len(allContent))/(1024*1024), elapsed, speed)
}
