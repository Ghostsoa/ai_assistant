#!/bin/bash
# infect.sh - 自动寄生目标服务器
# 用法: ./infect.sh <host> <user> <password> [alias] [secret_key]

HOST=$1
USER=$2
PASS=$3
ALIAS=${4:-"server-$(date +%s)"}
SECRET_KEY=${5:-$(openssl rand -hex 32)}

if [ -z "$HOST" ] || [ -z "$USER" ] || [ -z "$PASS" ]; then
    echo "用法: $0 <host> <user> <password> [alias] [secret_key]"
    echo "示例: $0 192.168.1.100 root password123 web-server"
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🦠 正在寄生目标服务器..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "目标: $USER@$HOST"
echo "别名: $ALIAS"
echo ""

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_SCRIPT="$SCRIPT_DIR/../agent/jarvis-agent.py"

# 使用expect自动化部署
expect << EOF
#!/usr/bin/expect
set timeout 30
set port 38888

# 步骤1: 上传寄生虫脚本
puts "\[1/5\] 上传寄生虫脚本..."
spawn scp -o StrictHostKeyChecking=no $AGENT_SCRIPT $USER@$HOST:/tmp/jarvis-agent.py
expect {
    "password:" { send "$PASS\r" }
    timeout { puts "\[✗\] 连接超时"; exit 1 }
}
expect eof

# 步骤2: SSH登录并部署
puts "\[2/5\] 配置服务..."
spawn ssh -o StrictHostKeyChecking=no $USER@$HOST
expect "password:" { send "$PASS\r" }
expect "#" 

# 移动脚本到系统目录
send "mv /tmp/jarvis-agent.py /usr/local/bin/\r"
expect "#"
send "chmod +x /usr/local/bin/jarvis-agent.py\r"
expect "#"

# 步骤3: 创建systemd服务
puts "\[3/5\] 创建systemd服务..."
send "cat > /etc/systemd/system/jarvis-agent.service << 'SERVICE'\r"
send "\[Unit\]\r"
send "Description=JARVIS Agent\r"
send "After=network.target\r"
send "\r"
send "\[Service\]\r"
send "Type=simple\r"
send "ExecStart=/usr/bin/python3 /usr/local/bin/jarvis-agent.py\r"
send "Restart=always\r"
send "RestartSec=5\r"
send "User=root\r"
send "\r"
send "\[Install\]\r"
send "WantedBy=multi-user.target\r"
send "SERVICE\r"
expect "#"

# 步骤4: 配置JWT密钥
puts "\[4/6\] 配置JWT密钥..."
send "echo 'export JARVIS_SECRET_KEY=\"$SECRET_KEY\"' >> /etc/environment\r"
expect "#"
puts "  ✓ JWT密钥已配置"

# 步骤5: 配置防火墙（ufw）
puts "\[5/6\] 配置防火墙..."

# 允许SSH和寄生虫端口
send "ufw --force enable\r"
expect "#"
send "ufw allow 22/tcp\r"
expect "#"
send "ufw allow \$port/tcp\r"
expect "#"
send "ufw reload\r"
expect "#"
puts "  ✓ ufw已启用并配置 (22, \$port 端口已开放)"

# 步骤6: 启动服务
puts "\[6/6\] 启动服务..."
send "systemctl daemon-reload\r"
expect "#"
send "systemctl enable jarvis-agent\r"
expect "#"
send "systemctl restart jarvis-agent\r"
expect "#"

# 等待启动
send "sleep 2\r"
expect "#"

# 检查状态
send "systemctl is-active jarvis-agent\r"
expect {
    "active" {
        puts "  ✓ 服务运行正常"
    }
    "inactive" {
        puts "  ✗ 服务启动失败"
        send "journalctl -u jarvis-agent -n 20\r"
        expect "#"
    }
}

send "exit\r"
expect eof
EOF

# 检查expect执行结果
if [ $? -eq 0 ]; then
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✓ 寄生成功！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "机器ID: $ALIAS"
    echo "地址:   $HOST:38888"
    echo ""
    
    # 返回机器信息（供Go程序解析）
    echo "MACHINE_INFO:$ALIAS:$HOST:38888"
else
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✗ 寄生失败"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 1
fi
