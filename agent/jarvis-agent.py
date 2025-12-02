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
    """持久Shell - 完整终端能力，支持工作目录和环境变量持久化"""
    def __init__(self):
        # 当前工作目录
        self.cwd = os.path.expanduser('~')
        # 环境变量（继承系统环境）
        self.env = os.environ.copy()
        # 上次工作目录（用于 cd -）
        self.oldpwd = self.cwd
    
    def execute(self, command):
        """执行命令并返回输出"""
        try:
            # 保存当前目录（用于cd -）
            old_cwd = self.cwd
            
            # 在当前工作目录和环境中执行命令
            result = subprocess.run(
                command,
                shell=True,
                cwd=self.cwd,
                env=self.env,
                capture_output=True,
                text=True,
                timeout=30,
                executable='/bin/bash'
            )
            
            # 合并stdout和stderr
            output = result.stdout
            if result.stderr:
                output += result.stderr
            
            # 截断输出（最多50行）
            lines = output.split('\n')
            if len(lines) > 50:
                output = '\n'.join(lines[:50]) + f"\n... [省略 {len(lines) - 50} 行]"
            
            # 执行成功后，获取真实的当前工作目录
            if result.returncode == 0:
                # 执行pwd获取真实路径（支持cd、cd -、cd ~等所有情况）
                pwd_result = subprocess.run(
                    f'cd {self.cwd} && {command} && pwd',
                    shell=True,
                    capture_output=True,
                    text=True,
                    timeout=5,
                    executable='/bin/bash'
                )
                
                if pwd_result.returncode == 0:
                    new_cwd = pwd_result.stdout.strip().split('\n')[-1]
                    if new_cwd and os.path.isabs(new_cwd):
                        # 更新OLDPWD（cd -需要）
                        if self.cwd != new_cwd:
                            self.oldpwd = self.cwd
                            self.env['OLDPWD'] = self.oldpwd
                        
                        self.cwd = new_cwd
                        self.env['PWD'] = self.cwd
            
            return output.strip()
            
        except subprocess.TimeoutExpired:
            return "[✗] 命令执行超时（30秒）"
        except Exception as e:
            return f"[✗] 命令执行失败: {str(e)}"

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

def handle_tar_upload(data):
    """接收tar.gz压缩包并直接解压（流式处理，无临时文件）"""
    import subprocess
    import tempfile
    
    target_path = data['path']
    content_b64 = data['content']
    
    # Base64解码
    content = base64.b64decode(content_b64)
    
    # 创建目标目录
    os.makedirs(target_path, exist_ok=True)
    
    # 使用管道直接解压，避免临时文件
    # echo content | tar xzf - -C target_path
    proc = subprocess.Popen(
        ['tar', 'xzf', '-', '-C', target_path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )
    
    stdout, stderr = proc.communicate(input=content)
    
    if proc.returncode != 0:
        raise Exception(f"解压失败: {stderr.decode('utf-8')}")
    
    return {
        'success': True,
        'path': target_path,
        'size': len(content)
    }

def handle_tar_download(data):
    """打包目录并返回tar.gz（流式处理，无临时文件）"""
    import subprocess
    
    source_path = data['path']
    
    if not os.path.exists(source_path):
        raise FileNotFoundError(f"Path not found: {source_path}")
    
    # 使用管道直接打包，避免临时文件
    # tar czf - -C source_path .
    proc = subprocess.Popen(
        ['tar', 'czf', '-', '-C', source_path, '.'],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )
    
    stdout, stderr = proc.communicate()
    
    if proc.returncode != 0:
        raise Exception(f"打包失败: {stderr.decode('utf-8')}")
    
    # Base64编码返回
    content_b64 = base64.b64encode(stdout).decode('utf-8')
    
    return {
        'success': True,
        'content': content_b64,
        'size': len(stdout)
    }

def handle_client(client_socket):
    """处理JARVIS的请求（支持大数据）"""
    try:
        # 设置接收超时（3秒内没新数据则认为接收完毕）
        client_socket.settimeout(3.0)
        
        # 接收请求（支持大数据）
        chunks = []
        while True:
            try:
                chunk = client_socket.recv(1024 * 1024)  # 1MB缓冲区
                if not chunk:
                    break
                chunks.append(chunk)
            except socket.timeout:
                # 超时说明数据接收完毕
                break
        
        if not chunks:
            raise ValueError("No data received")
        
        data = b''.join(chunks).decode('utf-8')
        request = json.loads(data)
        
        # 验证API Key
        api_key = request.get('api_key')
        if not api_key or api_key != API_KEY:
            raise ValueError("Invalid API key")
        
        # 获取操作类型
        action = request.get('action', 'execute')
        
        # 路由到不同处理函数
        if action == 'execute':
            # 命令执行（从data中获取）
            command = request.get('data', {}).get('command') or request.get('command')
            if not command:
                raise ValueError("Missing 'command' parameter")
            
            print(f"[DEBUG] 执行命令: {command}")  # 调试日志
            output = shell.execute(command)
            print(f"[DEBUG] 命令输出长度: {len(output)}")  # 调试日志
            print(f"[DEBUG] 当前工作目录: {shell.cwd}")  # 调试日志
            
            response = {
                'output': output,
                'cwd': shell.cwd,  # 返回当前工作目录
                'success': True
            }
            
        elif action == 'upload':
            response = handle_upload(request['data'])
            
        elif action == 'download':
            response = handle_download(request['data'])
            
        elif action == 'file_info':
            response = handle_file_info(request['data'])
            
        elif action == 'list_dir':
            response = handle_list_dir(request['data'])
            
        elif action == 'tar_upload':
            # 直接接收tar流并解压
            response = handle_tar_upload(request['data'])
            
        elif action == 'tar_download':
            # 直接打包并发送tar流
            response = handle_tar_download(request['data'])
            
        else:
            raise ValueError(f"Unknown action: {action}")
        
        # 返回结果（使用sendall确保大数据完整发送）
        client_socket.sendall(json.dumps(response).encode('utf-8'))
        
    except Exception as e:
        error_response = {
            'error': str(e),
            'success': False
        }
        client_socket.sendall(json.dumps(error_response).encode('utf-8'))
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
