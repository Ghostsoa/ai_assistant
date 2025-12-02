#!/usr/bin/env python3
# jarvis-agent.py - JARVIS 寄生虫代理
# 部署在目标服务器上，提供持久Shell能力

import subprocess
import socket
import json
import threading
import time
import select
import os
import base64
import hashlib

PORT = 38888  # 高位端口，避免冲突
CHUNK_SIZE = 1024 * 1024  # 1MB 分块大小

# 全局API Key（硬编码，所有寄生虫统一）
API_KEY = os.environ.get('JARVIS_API_KEY', 'JARVIS_GLOBAL_SECRET_KEY_2024')

class PersistentShell:
    """持久Shell - 与本地ExecuteInPersistentShell能力一致"""
    def __init__(self):
        # 启动bash（交互式）
        self.process = subprocess.Popen(
            ['/bin/bash', '-i'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            bufsize=0,
            text=True
        )
        
        # 等待shell初始化
        time.sleep(0.5)
        # 清空初始化输出
        self._read_available()
    
    def _read_available(self):
        """读取当前可用的所有输出"""
        output = []
        while True:
            ready, _, _ = select.select([self.process.stdout], [], [], 0.1)
            if not ready:
                break
            line = self.process.stdout.readline()
            if line:
                output.append(line)
        return ''.join(output)
    
    def execute(self, command):
        """执行命令（使用marker机制，和本地一致）"""
        # 生成唯一marker
        marker = f"__END_{int(time.time() * 1000000)}__"
        
        # 发送命令 + marker
        cmd_line = f"{command}; echo {marker}\n"
        self.process.stdin.write(cmd_line)
        self.process.stdin.flush()
        
        # 收集输出，直到看到marker
        output = []
        max_wait = 50  # 最多等待5秒
        
        for _ in range(max_wait):
            time.sleep(0.1)
            lines = self._read_available()
            output.append(lines)
            
            # 检查是否包含marker
            if marker in lines:
                # 移除marker行
                full_output = ''.join(output)
                lines_list = full_output.split('\n')
                filtered = [l for l in lines_list if marker not in l and '__END_' not in l]
                return '\n'.join(filtered).strip()
        
        # 超时，返回当前输出
        return ''.join(output).strip()

# 全局Shell实例
shell = PersistentShell()

def handle_upload(data):
    """处理文件上传（支持分块和完整文件）"""
    path = data['path']
    
    # 方式1：分块上传（兼容旧接口）
    if 'offset' in data:
        content_b64 = data['content']
        offset = data['offset']
        total_size = data.get('total_size', 0)
        
        content = base64.b64decode(content_b64)
        mode = 'ab' if offset > 0 else 'wb'
        
        # 创建目录
        dir_path = os.path.dirname(path)
        if dir_path:
            os.makedirs(dir_path, exist_ok=True)
        
        with open(path, mode) as f:
            f.write(content)
        
        current_size = os.path.getsize(path)
        return {
            'success': True,
            'uploaded': current_size,
            'total': total_size,
            'progress': (current_size / total_size * 100) if total_size > 0 else 100
        }
    
    # 方式2：完整文件上传（自动分块处理，Go端不用管）
    elif 'content' in data:
        content_b64 = data['content']
        content = base64.b64decode(content_b64)
        
        # 创建目录
        dir_path = os.path.dirname(path)
        if dir_path:
            os.makedirs(dir_path, exist_ok=True)
        
        with open(path, 'wb') as f:
            f.write(content)
        
        return {
            'success': True,
            'size': len(content),
            'path': path
        }
    
    else:
        raise ValueError("Missing 'content' or 'offset' parameter")

def handle_download(data):
    """处理文件下载"""
    path = data['path']
    offset = data.get('offset', 0)
    chunk_size = data.get('chunk_size', CHUNK_SIZE)
    
    if not os.path.exists(path):
        raise FileNotFoundError(f"File not found: {path}")
    
    # 读取指定块
    with open(path, 'rb') as f:
        f.seek(offset)
        content = f.read(chunk_size)
    
    # Base64编码
    content_b64 = base64.b64encode(content).decode('utf-8')
    
    # 文件信息
    file_size = os.path.getsize(path)
    
    return {
        'success': True,
        'content': content_b64,
        'offset': offset,
        'chunk_size': len(content),
        'total_size': file_size,
        'eof': offset + len(content) >= file_size
    }

def handle_file_info(data):
    """获取文件/目录信息"""
    path = data['path']
    
    if not os.path.exists(path):
        return {'success': False, 'error': 'Path not found'}
    
    stat = os.stat(path)
    return {
        'success': True,
        'path': path,
        'size': stat.st_size,
        'is_dir': os.path.isdir(path),
        'is_file': os.path.isfile(path),
        'mtime': stat.st_mtime,
        'mode': stat.st_mode
    }

def handle_list_dir(data):
    """列出目录内容"""
    path = data['path']
    
    if not os.path.isdir(path):
        raise NotADirectoryError(f"Not a directory: {path}")
    
    items = []
    for item in os.listdir(path):
        item_path = os.path.join(path, item)
        stat = os.stat(item_path)
        items.append({
            'name': item,
            'size': stat.st_size,
            'is_dir': os.path.isdir(item_path),
            'mtime': stat.st_mtime
        })
    
    return {
        'success': True,
        'items': items
    }

def handle_client(client_socket):
    """处理JARVIS的请求（支持多种操作）"""
    try:
        # 接收请求
        data = client_socket.recv(65536).decode('utf-8')  # 增大缓冲区
        request = json.loads(data)
        
        # 验证API Key
        api_key = request.get('api_key')
        if not api_key or api_key != API_KEY:
            raise ValueError("Invalid API key")
        
        # 获取操作类型
        action = request.get('action', 'execute')
        
        # 路由到不同处理函数
        if action == 'execute':
            # 原有的命令执行
            command = request['command']
            output = shell.execute(command)
            response = {'output': output, 'success': True}
            
        elif action == 'upload':
            response = handle_upload(request['data'])
            
        elif action == 'download':
            response = handle_download(request['data'])
            
        elif action == 'file_info':
            response = handle_file_info(request['data'])
            
        elif action == 'list_dir':
            response = handle_list_dir(request['data'])
            
        else:
            raise ValueError(f"Unknown action: {action}")
        
        # 返回结果
        client_socket.send(json.dumps(response).encode('utf-8'))
        
    except Exception as e:
        error_response = {
            'error': str(e),
            'success': False
        }
        client_socket.send(json.dumps(error_response).encode('utf-8'))
    finally:
        client_socket.close()

def main():
    """启动TCP服务器"""
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('0.0.0.0', PORT))
    server.listen(5)
    
    print(f"[✓] JARVIS Agent running on port {PORT}")
    print(f"[✓] Persistent Shell initialized")
    
    while True:
        client, addr = server.accept()
        thread = threading.Thread(target=handle_client, args=(client,))
        thread.daemon = True
        thread.start()

if __name__ == '__main__':
    main()
