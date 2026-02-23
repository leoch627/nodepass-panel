#!/bin/bash
set -e

# 解决 macOS 下 tr 可能出现的非法字节序列问题
export LANG=en_US.UTF-8
export LC_ALL=C



# 全局下载地址配置
DOCKER_COMPOSE_URL="https://github.com/0xNetuser/flux-panel/releases/latest/download/docker-compose.yml"

COUNTRY=$(curl -s https://ipinfo.io/country)
if [ "$COUNTRY" = "CN" ]; then
    # 拼接 URL
    DOCKER_COMPOSE_URL="https://ghfast.top/${DOCKER_COMPOSE_URL}"
fi

# 检查 docker-compose 或 docker compose 命令
check_docker() {
  if command -v docker-compose &> /dev/null; then
    DOCKER_CMD="docker-compose"
  elif command -v docker &> /dev/null; then
    if docker compose version &> /dev/null; then
      DOCKER_CMD="docker compose"
    else
      echo "错误：检测到 docker，但不支持 'docker compose' 命令。请安装 docker-compose 或更新 docker 版本。"
      exit 1
    fi
  else
    echo "错误：未检测到 docker 或 docker-compose 命令。请先安装 Docker。"
    exit 1
  fi
  echo "检测到 Docker 命令：$DOCKER_CMD"
}

# 检测系统是否支持 IPv6
check_ipv6_support() {
  echo "🔍 检测 IPv6 支持..."

  # 检查是否有 IPv6 地址（排除 link-local 地址）
  if ip -6 addr show | grep -v "scope link" | grep -q "inet6"; then
    echo "✅ 检测到系统支持 IPv6"
    return 0
  elif ifconfig 2>/dev/null | grep -v "fe80:" | grep -q "inet6"; then
    echo "✅ 检测到系统支持 IPv6"
    return 0
  else
    echo "⚠️ 未检测到 IPv6 支持"
    return 1
  fi
}



# 配置 Docker 启用 IPv6
configure_docker_ipv6() {
  echo "🔧 配置 Docker IPv6 支持..."

  # 检查操作系统类型
  OS_TYPE=$(uname -s)

  if [[ "$OS_TYPE" == "Darwin" ]]; then
    # macOS 上 Docker Desktop 已默认支持 IPv6
    echo "✅ macOS Docker Desktop 默认支持 IPv6"
    return 0
  fi

  # Docker daemon 配置文件路径
  DOCKER_CONFIG="/etc/docker/daemon.json"

  # 检查是否需要 sudo
  if [[ $EUID -ne 0 ]]; then
    SUDO_CMD="sudo"
  else
    SUDO_CMD=""
  fi

  # 检查 Docker 配置文件
  if [ -f "$DOCKER_CONFIG" ]; then
    # 检查是否已经配置了 IPv6
    if grep -q '"ipv6"' "$DOCKER_CONFIG"; then
      echo "✅ Docker 已配置 IPv6 支持"
    else
      echo "📝 更新 Docker 配置以启用 IPv6..."
      # 备份原配置
      $SUDO_CMD cp "$DOCKER_CONFIG" "${DOCKER_CONFIG}.backup"

      # 使用 jq 或 sed 添加 IPv6 配置
      if command -v jq &> /dev/null; then
        $SUDO_CMD jq '. + {"ipv6": true, "fixed-cidr-v6": "fd00::/80"}' "$DOCKER_CONFIG" > /tmp/daemon.json && $SUDO_CMD mv /tmp/daemon.json "$DOCKER_CONFIG"
      else
        # 如果没有 jq，使用 sed
        $SUDO_CMD sed -i 's/^{$/{\n  "ipv6": true,\n  "fixed-cidr-v6": "fd00::\/80",/' "$DOCKER_CONFIG"
      fi

      echo "🔄 重启 Docker 服务..."
      if command -v systemctl &> /dev/null; then
        $SUDO_CMD systemctl restart docker
      elif command -v service &> /dev/null; then
        $SUDO_CMD service docker restart
      else
        echo "⚠️ 请手动重启 Docker 服务"
      fi
      sleep 5
    fi
  else
    # 创建新的配置文件
    echo "📝 创建 Docker 配置文件..."
    $SUDO_CMD mkdir -p /etc/docker
    echo '{
  "ipv6": true,
  "fixed-cidr-v6": "fd00::/80"
}' | $SUDO_CMD tee "$DOCKER_CONFIG" > /dev/null

    echo "🔄 重启 Docker 服务..."
    if command -v systemctl &> /dev/null; then
      $SUDO_CMD systemctl restart docker
    elif command -v service &> /dev/null; then
      $SUDO_CMD service docker restart
    else
      echo "⚠️ 请手动重启 Docker 服务"
    fi
    sleep 5
  fi
}

# 显示菜单
show_menu() {
  echo "==============================================="
  echo "          面板管理脚本"
  echo "==============================================="
  echo "请选择操作："
  echo "1. 安装面板"
  echo "2. 更新面板"
  echo "3. 卸载面板"
  echo "4. 导出备份"
  echo "5. 退出"
  echo "==============================================="
}

generate_random() {
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c16
}

# 删除脚本自身
delete_self() {
  echo ""
  echo "🗑️ 操作已完成，正在清理脚本文件..."
  SCRIPT_PATH="$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")"
  sleep 1
  rm -f "$SCRIPT_PATH" && echo "✅ 脚本文件已删除" || echo "❌ 删除脚本文件失败"
}



# 获取用户输入的配置参数
get_config_params() {
  echo "🔧 请输入配置参数："



  read -p "面板端口（默认 6366）: " PANEL_PORT
  PANEL_PORT=${PANEL_PORT:-6366}

  DB_NAME=$(generate_random)
  DB_USER=$(generate_random)
  DB_PASSWORD=$(generate_random)
  JWT_SECRET=$(generate_random)
}

# 安装功能
install_panel() {
  echo "🚀 开始安装面板..."
  check_docker
  get_config_params

  echo "🔽 下载必要文件..."
  echo "📡 下载配置文件..."
  curl -L -o docker-compose.yml "$DOCKER_COMPOSE_URL"

  echo "✅ 文件准备完成"

  # 自动检测并配置 IPv6 支持
  ENABLE_IPV6=false
  if check_ipv6_support; then
    echo "🚀 系统支持 IPv6，自动启用 IPv6 配置..."
    ENABLE_IPV6=true
    configure_docker_ipv6
  fi

  cat > .env <<EOF
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
JWT_SECRET=$JWT_SECRET
PANEL_PORT=$PANEL_PORT
ENABLE_IPV6=$ENABLE_IPV6
EOF

  echo "🚀 启动 docker 服务..."
  $DOCKER_CMD up -d

  echo "🎉 部署完成"
  echo "🌐 访问地址: http://服务器IP:$PANEL_PORT"
  echo "📖 部署完成后请阅读下使用文档，求求了啊，不要上去就是一顿操作"
  echo "📚 文档地址: https://tes.cc/guide.html"
  echo "💡 默认管理员账号: admin_user"
  echo "🔑 首次启动会自动重置默认密码，请查看日志获取新密码："
  echo "   docker logs go-backend 2>&1 | grep '新密码'"


}

# 更新功能
update_panel() {
  echo "🔄 开始更新面板..."
  check_docker

  echo "🔽 下载最新配置文件..."
  curl -L -o docker-compose.yml "$DOCKER_COMPOSE_URL"
  echo "✅ 下载完成"

  # 自动检测并配置 IPv6 支持
  if check_ipv6_support; then
    echo "🚀 系统支持 IPv6，自动启用 IPv6 配置..."
    configure_docker_ipv6
    # 确保 .env 中有 ENABLE_IPV6
    if [[ -f ".env" ]] && ! grep -q "^ENABLE_IPV6=" .env; then
      echo "ENABLE_IPV6=true" >> .env
    elif [[ -f ".env" ]]; then
      sed -i 's/^ENABLE_IPV6=.*/ENABLE_IPV6=true/' .env
    fi
  fi

  echo "🛑 停止当前服务..."
  $DOCKER_CMD down

  echo "⬇️ 拉取最新镜像..."
  $DOCKER_CMD pull

  echo "🚀 启动更新后的服务..."
  $DOCKER_CMD up -d

  # 等待服务启动
  echo "⏳ 等待服务启动..."

  # 检查后端容器健康状态
  echo "🔍 检查后端服务状态..."
  for i in {1..90}; do
    if docker ps --format "{{.Names}}" | grep -q "^go-backend$"; then
      BACKEND_HEALTH=$(docker inspect -f '{{.State.Health.Status}}' go-backend 2>/dev/null || echo "unknown")
      if [[ "$BACKEND_HEALTH" == "healthy" ]]; then
        echo "✅ 后端服务健康检查通过"
        break
      elif [[ "$BACKEND_HEALTH" == "starting" ]]; then
        # 继续等待
        :
      elif [[ "$BACKEND_HEALTH" == "unhealthy" ]]; then
        echo "⚠️ 后端健康状态：$BACKEND_HEALTH"
      fi
    else
      echo "⚠️ 后端容器未找到或未运行"
      BACKEND_HEALTH="not_running"
    fi
    if [ $i -eq 90 ]; then
      echo "❌ 后端服务启动超时（90秒）"
      echo "🔍 当前状态：$(docker inspect -f '{{.State.Health.Status}}' go-backend 2>/dev/null || echo '容器不存在')"
      echo "🛑 更新终止"
      return 1
    fi
    # 每15秒显示一次进度
    if [ $((i % 15)) -eq 1 ]; then
      echo "⏳ 等待后端服务启动... ($i/90) 状态：${BACKEND_HEALTH:-unknown}"
    fi
    sleep 1
  done

  # 检查数据库容器健康状态
  echo "🔍 检查数据库服务状态..."
  for i in {1..60}; do
    if docker ps --format "{{.Names}}" | grep -q "^flux-mysql$"; then
      DB_HEALTH=$(docker inspect -f '{{.State.Health.Status}}' flux-mysql 2>/dev/null || echo "unknown")
      if [[ "$DB_HEALTH" == "healthy" ]]; then
        echo "✅ 数据库服务健康检查通过"
        break
      elif [[ "$DB_HEALTH" == "starting" ]]; then
        # 继续等待
        :
      elif [[ "$DB_HEALTH" == "unhealthy" ]]; then
        echo "⚠️ 数据库健康状态：$DB_HEALTH"
      fi
    else
      echo "⚠️ 数据库容器未找到或未运行"
      DB_HEALTH="not_running"
    fi
    if [ $i -eq 60 ]; then
      echo "❌ 数据库服务启动超时（60秒）"
      echo "🔍 当前状态：$(docker inspect -f '{{.State.Health.Status}}' flux-mysql 2>/dev/null || echo '容器不存在')"
      echo "🛑 更新终止"
      return 1
    fi
    # 每10秒显示一次进度
    if [ $((i % 10)) -eq 1 ]; then
      echo "⏳ 等待数据库服务启动... ($i/60) 状态：${DB_HEALTH:-unknown}"
    fi
    sleep 1
  done

  # 从容器环境变量获取数据库信息
  echo "🔍 获取数据库配置信息..."

  # 等待一下让服务完全就绪
  echo "⏳ 等待服务完全就绪..."
  sleep 5

  # 先检查后端容器是否在运行
  if ! docker ps --format "{{.Names}}" | grep -q "^go-backend$"; then
    echo "❌ 后端容器未运行，无法获取数据库配置"
    echo "🔍 当前运行的容器："
    docker ps --format "table {{.Names}}\t{{.Status}}"
    echo "🛑 更新终止"
    return 1
  fi

  DB_INFO=$(docker exec go-backend env | grep "^DB_" 2>/dev/null || echo "")

  if [[ -n "$DB_INFO" ]]; then
    DB_NAME=$(echo "$DB_INFO" | grep "^DB_NAME=" | cut -d'=' -f2)
    DB_PASSWORD=$(echo "$DB_INFO" | grep "^DB_PASSWORD=" | cut -d'=' -f2)
    DB_USER=$(echo "$DB_INFO" | grep "^DB_USER=" | cut -d'=' -f2)
    DB_HOST=$(echo "$DB_INFO" | grep "^DB_HOST=" | cut -d'=' -f2)

    echo "📋 数据库配置："
    echo "   数据库名: $DB_NAME"
    echo "   用户名: $DB_USER"
    echo "   主机: $DB_HOST"
  else
    echo "❌ 无法获取数据库配置信息"
    echo "🔍 尝试诊断问题："
    echo "   容器状态: $(docker inspect -f '{{.State.Status}}' go-backend 2>/dev/null || echo '容器不存在')"
    echo "   健康状态: $(docker inspect -f '{{.State.Health.Status}}' go-backend 2>/dev/null || echo '无健康检查')"

    # 尝试从 .env 文件读取配置
    if [[ -f ".env" ]]; then
      echo "🔄 尝试从 .env 文件读取配置..."
      DB_NAME=$(grep "^DB_NAME=" .env | cut -d'=' -f2 2>/dev/null)
      DB_PASSWORD=$(grep "^DB_PASSWORD=" .env | cut -d'=' -f2 2>/dev/null)
      DB_USER=$(grep "^DB_USER=" .env | cut -d'=' -f2 2>/dev/null)

      if [[ -n "$DB_NAME" && -n "$DB_PASSWORD" && -n "$DB_USER" ]]; then
        echo "✅ 从 .env 文件成功读取数据库配置"
        echo "📋 数据库配置："
        echo "   数据库名: $DB_NAME"
        echo "   用户名: $DB_USER"
      else
        echo "❌ .env 文件中的数据库配置不完整"
        echo "🛑 更新终止"
        return 1
      fi
    else
      echo "❌ 未找到 .env 文件"
      echo "🛑 更新终止"
      return 1
    fi
  fi

  # 检查必要的数据库配置
  if [[ -z "$DB_PASSWORD" || -z "$DB_USER" || -z "$DB_NAME" ]]; then
    echo "❌ 数据库配置不完整（缺少必要参数）"
    echo "🛑 更新终止"
    return 1
  fi

  # 执行数据库字段变更
  echo "🔄 执行数据库结构更新..."

  # 创建临时迁移文件
  # 注：建表和加列由后端 GORM AutoMigrate 自动完成，此处只处理 GORM 无法执行的操作：
  #   1. DROP COLUMN（清理旧版废弃字段）
  #   2. 数据迁移 UPDATE（为老数据填充默认值）
  cat > temp_migration.sql <<EOF
USE \`$DB_NAME\`;

-- ============ DROP 废弃字段（GORM 不支持删列）============

-- user.name（旧版遗留）
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='user' AND column_name='name'), 'ALTER TABLE \`user\` DROP COLUMN \`name\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- node.port（已拆分为 port_sta/port_end）
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='node' AND column_name='port'), 'ALTER TABLE \`node\` DROP COLUMN \`port\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- tunnel 废弃字段
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='tunnel' AND column_name='in_port_sta'), 'ALTER TABLE \`tunnel\` DROP COLUMN \`in_port_sta\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='tunnel' AND column_name='in_port_end'), 'ALTER TABLE \`tunnel\` DROP COLUMN \`in_port_end\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='tunnel' AND column_name='out_ip_sta'), 'ALTER TABLE \`tunnel\` DROP COLUMN \`out_ip_sta\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='tunnel' AND column_name='out_ip_end'), 'ALTER TABLE \`tunnel\` DROP COLUMN \`out_ip_end\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- forward.proxy_protocol（已移除）
SET @sql = (SELECT IF(EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE table_schema=DATABASE() AND table_name='forward' AND column_name='proxy_protocol'), 'ALTER TABLE \`forward\` DROP COLUMN \`proxy_protocol\`', 'SELECT 1'));
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ============ 数据迁移（为老数据填充默认值）============

-- 旧版 node.ip → server_ip
UPDATE \`node\` SET \`server_ip\` = \`ip\` WHERE \`server_ip\` IS NULL OR \`server_ip\` = '';

-- 默认端口范围
UPDATE \`node\` SET \`port_sta\` = 1000, \`port_end\` = 65535 WHERE \`port_sta\` IS NULL OR \`port_end\` IS NULL;

-- 默认 http/tls/socks
UPDATE \`node\` SET \`http\` = IFNULL(\`http\`, 0), \`tls\` = IFNULL(\`tls\`, 0), \`socks\` = IFNULL(\`socks\`, 0);

-- 默认流量倍率
UPDATE \`tunnel\` SET \`traffic_ratio\` = 1.0 WHERE \`traffic_ratio\` IS NULL;

-- 默认负载策略
UPDATE \`forward\` SET \`strategy\` = 'fifo' WHERE \`strategy\` IS NULL;

-- 默认排序索引
UPDATE \`forward\` SET \`inx\` = 0 WHERE \`inx\` IS NULL;

-- statistics_flow 时间戳回填
UPDATE \`statistics_flow\` SET \`created_time\` = UNIX_TIMESTAMP() * 1000 WHERE \`created_time\` = 0 OR \`created_time\` IS NULL;

EOF

  # 检查数据库容器
  if ! docker ps --format "{{.Names}}" | grep -q "^flux-mysql$"; then
    echo "❌ 数据库容器 flux-mysql 未运行"
    echo "🔍 当前运行的容器："
    docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"
    echo "❌ 数据库结构更新失败，请手动执行 temp_migration.sql"
    echo "📁 迁移文件已保存为 temp_migration.sql"
    return 1
  fi

  # 执行数据库迁移
  if docker exec -i flux-mysql mysql -u "$DB_USER" -p"$DB_PASSWORD" < temp_migration.sql 2>/dev/null; then
    echo "✅ 数据库结构更新完成"
  else
    echo "⚠️ 使用用户密码失败，尝试root密码..."
    if docker exec -i flux-mysql mysql -u root -p"$DB_PASSWORD" < temp_migration.sql 2>/dev/null; then
      echo "✅ 数据库结构更新完成"
    else
      echo "❌ 数据库结构更新失败，请手动执行 temp_migration.sql"
      echo "📁 迁移文件已保存为 temp_migration.sql"
      echo "🔍 数据库容器状态: $(docker inspect -f '{{.State.Status}}' flux-mysql 2>/dev/null || echo '容器不存在')"
      echo "🛑 更新终止"
      return 1
    fi
  fi

  # 清理临时文件
  rm -f temp_migration.sql

  echo "✅ 更新完成"
}

# 导出数据库备份
export_migration_sql() {
  echo "📄 开始导出数据库备份..."

  # 获取数据库配置信息
  echo "🔍 获取数据库配置信息..."

  # 先检查后端容器是否在运行
  if ! docker ps --format "{{.Names}}" | grep -q "^go-backend$"; then
    echo "❌ 后端容器未运行，尝试从 .env 文件读取配置..."

    # 从 .env 文件读取配置
    if [[ -f ".env" ]]; then
      DB_NAME=$(grep "^DB_NAME=" .env | cut -d'=' -f2 2>/dev/null)
      DB_PASSWORD=$(grep "^DB_PASSWORD=" .env | cut -d'=' -f2 2>/dev/null)
      DB_USER=$(grep "^DB_USER=" .env | cut -d'=' -f2 2>/dev/null)

      if [[ -n "$DB_NAME" && -n "$DB_PASSWORD" && -n "$DB_USER" ]]; then
        echo "✅ 从 .env 文件读取数据库配置成功"
      else
        echo "❌ .env 文件中的数据库配置不完整"
        return 1
      fi
    else
      echo "❌ 未找到 .env 文件"
      return 1
    fi
  else
    # 从容器环境变量获取数据库信息
    DB_INFO=$(docker exec go-backend env | grep "^DB_" 2>/dev/null || echo "")

    if [[ -n "$DB_INFO" ]]; then
      DB_NAME=$(echo "$DB_INFO" | grep "^DB_NAME=" | cut -d'=' -f2)
      DB_PASSWORD=$(echo "$DB_INFO" | grep "^DB_PASSWORD=" | cut -d'=' -f2)
      DB_USER=$(echo "$DB_INFO" | grep "^DB_USER=" | cut -d'=' -f2)

      echo "✅ 从容器环境变量读取数据库配置成功"
    else
      echo "❌ 无法从容器获取数据库配置，尝试从 .env 文件读取..."

      if [[ -f ".env" ]]; then
        DB_NAME=$(grep "^DB_NAME=" .env | cut -d'=' -f2 2>/dev/null)
        DB_PASSWORD=$(grep "^DB_PASSWORD=" .env | cut -d'=' -f2 2>/dev/null)
        DB_USER=$(grep "^DB_USER=" .env | cut -d'=' -f2 2>/dev/null)

        if [[ -n "$DB_NAME" && -n "$DB_PASSWORD" && -n "$DB_USER" ]]; then
          echo "✅ 从 .env 文件读取数据库配置成功"
        else
          echo "❌ .env 文件中的数据库配置不完整"
          return 1
        fi
      else
        echo "❌ 未找到 .env 文件"
        return 1
      fi
    fi
  fi

  # 检查必要的数据库配置
  if [[ -z "$DB_PASSWORD" || -z "$DB_USER" || -z "$DB_NAME" ]]; then
    echo "❌ 数据库配置不完整（缺少必要参数）"
    return 1
  fi

  echo "📋 数据库配置："
  echo "   数据库名: $DB_NAME"
  echo "   用户名: $DB_USER"

  # 检查数据库容器是否运行
  if ! docker ps --format "{{.Names}}" | grep -q "^flux-mysql$"; then
    echo "❌ 数据库容器未运行，无法导出数据"
    echo "🔍 当前运行的容器："
    docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"
    return 1
  fi

  # 生成数据库备份文件
  SQL_FILE="database_backup_$(date +%Y%m%d_%H%M%S).sql"
  echo "📝 导出数据库备份: $SQL_FILE"

  # 使用 mysqldump 导出数据库
  echo "⏳ 正在导出数据库..."
  if docker exec flux-mysql mysqldump -u "$DB_USER" -p"$DB_PASSWORD" --single-transaction --routines --triggers "$DB_NAME" > "$SQL_FILE" 2>/dev/null; then
    echo "✅ 数据库导出成功"
  else
    echo "⚠️ 使用用户密码失败，尝试root密码..."
    if docker exec flux-mysql mysqldump -u root -p"$DB_PASSWORD" --single-transaction --routines --triggers "$DB_NAME" > "$SQL_FILE" 2>/dev/null; then
      echo "✅ 数据库导出成功"
    else
      echo "❌ 数据库导出失败"
      rm -f "$SQL_FILE"
      return 1
    fi
  fi

  # 检查文件大小
  if [[ -f "$SQL_FILE" ]] && [[ -s "$SQL_FILE" ]]; then
    FILE_SIZE=$(du -h "$SQL_FILE" | cut -f1)
    echo "📁 文件位置: $(pwd)/$SQL_FILE"
    echo "📊 文件大小: $FILE_SIZE"
  else
    echo "❌ 导出的文件为空或不存在"
    rm -f "$SQL_FILE"
    return 1
  fi
}


# 卸载功能
uninstall_panel() {
  echo "🗑️ 开始卸载面板..."
  check_docker

  if [[ ! -f "docker-compose.yml" ]]; then
    echo "⚠️ 未找到 docker-compose.yml 文件，正在下载以完成卸载..."
    curl -L -o docker-compose.yml "$DOCKER_COMPOSE_URL"
    echo "✅ docker-compose.yml 下载完成"
  fi

  read -p "确认卸载面板吗？此操作将停止并删除所有容器和数据 (y/N): " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "❌ 取消卸载"
    return 0
  fi

  echo "🛑 停止并删除容器、镜像、卷..."
  $DOCKER_CMD down --rmi all --volumes --remove-orphans
  echo "🧹 删除配置文件..."
  rm -f docker-compose.yml .env
  echo "✅ 卸载完成"
}

# 主逻辑
main() {

  # 显示交互式菜单
  while true; do
    show_menu
    read -p "请输入选项 (1-5): " choice

    case $choice in
      1)
        install_panel
        delete_self
        exit 0
        ;;
      2)
        update_panel
        delete_self
        exit 0
        ;;
      3)
        uninstall_panel
        delete_self
        exit 0
        ;;
      4)
        export_migration_sql
        delete_self
        exit 0
        ;;
      5)
        echo "👋 退出脚本"
        delete_self
        exit 0
        ;;
      *)
        echo "❌ 无效选项，请输入 1-5"
        echo ""
        ;;
    esac
  done
}

# 执行主函数
main
