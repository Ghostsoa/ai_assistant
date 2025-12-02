#!/bin/bash
# infect.sh - 自动寄生目标服务器
# 用法: ./infect.sh <host> <user> <password> [alias]

HOST=$1
USER=$2
PASS=$3
ALIAS=${4:-"server-$(date +%s)"}

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_SCRIPT="$SCRIPT_DIR/../agent/jarvis-agent.py"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🦠 正在寄生目标服务器..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "目标: $USER@$HOST"
echo "别名: $ALIAS"
echo ""

# 使用expect自动化部署
expect << EOF
set timeout 30

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

# 步骤4: 配置防火墙
puts "\[4/5\] 配置防火墙..."

# 检测防火墙类型
send "which firewall-cmd 2>/dev/null && echo FIREWALLD || echo NONE\r"
expect {
    "FIREWALLD" {
        send "firewall-cmd --permanent --add-port=7788/tcp\r"
        expect "#"
        send "firewall-cmd --reload\r"
        expect "#"
        puts "  ✓ firewalld已配置"
    }
    "NONE" {
        send "which ufw 2>/dev/null && echo UFW || echo NONE\r"
        expect {
            "UFW" {
                send "ufw allow 7788/tcp\r"
                expect "#"
                puts "  ✓ ufw已配置"
            }
            "NONE" {
                send "which iptables 2>/dev/null && echo IPTABLES || echo NONE\r"
                expect {
                    "IPTABLES" {
                        send "iptables -I INPUT -p tcp --dport 7788 -j ACCEPT\r"
                        expect "#"
                        send "service iptables save 2>/dev/null || iptables-save > /etc/iptables/rules.v4 2>/dev/null\r"
                        expect "#"
                        puts "  ✓ iptables已配置"
                    }
                    "NONE" {
                        puts "  ! 未检测到防火墙，跳过"
                    }
                }
            }
        }
    }
}

# 步骤5: 启动服务
puts "\[5/5\] 启动服务..."
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
    echo "地址:   $HOST:7788"
    echo ""
    
    # 返回机器信息（供Go程序解析）
    echo "MACHINE_INFO:$ALIAS:$HOST:7788"
else
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✗ 寄生失败"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 1
fi
