#!/usr/bin/env python3
# jarvis-agent.py - JARVIS 寄生虫代理
# 部署在目标服务器上，提供持久Shell能力

import subprocess
import socket
import json
import threading
import time
import select

PORT = 38888  # 高位端口，避免冲突

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

def handle_client(client_socket):
    """处理JARVIS的命令请求"""
    try:
        # 接收命令
        data = client_socket.recv(4096).decode('utf-8')
        request = json.loads(data)
        command = request['command']
        
        # 在持久Shell中执行
        output = shell.execute(command)
        
        # 返回结果
        response = {
            'output': output,
            'success': True
        }
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
