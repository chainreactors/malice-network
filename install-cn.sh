#!/bin/bash
set -e

log_task_status() {
    local status="$1"
    local message="$2"
    if [ "$status" = "completed" ]; then
        echo "[✔]: $message"
    elif [ "$status" = "in_progress" ]; then
        echo "[⏳]: $message"
    elif [ "$status" = "ended" ]; then
        echo "[✘]: $message"
    else
        echo "[?]: Unknown status"
    fi
}

download_file() {
    local url="$1"
    local dest="$2"

    log_task_status "in_progress" "正在从 $url 下载到 $dest"
    curl --retry 4 --silent -L -o "$dest" "$url"
}

# set your server ip
setup_environment(){
    set_server_ip() {
        default_ip=$(curl --noproxy -4 -s ifconfig.me)
        if [[ -t 0 ]]; then
            read -p "请输入你的服务器公网(或内网)IP地址 [default: $default_ip]: " input_ip
            ip_address=${input_ip:-$default_ip}
        else
            ip_address=$default_ip
            log_task_status "completed" "No interactive shell detected. Using default IP Address: $ip_address"
        fi
        log_task_status "completed" "使用IP地址：$ip_address"
    }

    set_base_dir(){
        local DEFAULT_DIR="/opt/iom"
        if [[ -t 0 ]]; then
            read -p "请输入安装的根目录 [默认: $DEFAULT_DIR]: " input_dir
            IoM_ROOT_DIR=${input_dir:-$DEFAULT_DIR}
        else
            IoM_ROOT_DIR=$DEFAULT_DIR
            log_task_status "completed" "无输出, 将使用默认根目录：$IoM_ROOT_DIR"
        fi
        log_task_status completed "Using base directory: $IoM_ROOT_DIR"
    }
    set_base_dir
    set_server_ip
}

# check and install docker
check_and_install_docker(){
    log_task_status in_progress "Malefic的自动编译使用如下两种方式至少一种:"
    echo "  1. Docker (安装docker以及编译镜像)"
    echo "  2. Github Action (配置参考: https://chainreactors.github.io/wiki/IoM/manual/manual/deploy/#config)"
    yum_install_docker(){
        yum install -y yum-utils curl unzip git 
        yum-config-manager --add-repo http://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
        yum makecache fast
        yum install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }
    apt_install_docker(){
        apt update && apt install -y ca-certificates curl unzip git
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL "https://mirrors.aliyun.com/docker-ce/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
        chmod a+r /etc/apt/keyrings/docker.asc
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://mirrors.aliyun.com/docker-ce/linux/$ID $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
        apt update -y && apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }
    change_docker_daemon(){
        config_file="/etc/docker/daemon.json"
        mkdir -p $(dirname "$config_file")

        # 默认的镜像列表
        default_mirrors=(
            "https://mirror.ccs.tencentyun.com" # 腾讯云
            "https://dockerhub.azk8s.cn"        # Azure 中国
            "https://docker.fxxk.dedyn.io"
            "https://docker.1ms.run"
            "https://dockerpull.org"
            "https://dockerhub.timeweb.cloud" 
        )

        # 提示用户输入自定义镜像地址
        echo "==========================================================="
        echo "可以根据当前的服务器厂商添加Docker 镜像加速源，以加速 Docker 镜像的拉取。"
        echo "以下是一些常见的云厂商内部镜像地址供参考："
        echo "  1. 腾讯云服务器可用: https://mirror.ccs.tencentyun.com"
        echo "  2. Azure 可用: https://dockerhub.azk8s.cn"
        echo "  3. 阿里云可用: https://<your_code>.mirror.aliyuncs.com"
        echo "     （登录阿里云查看 https://cr.console.aliyun.com/cn-hongkong/instances/mirrors）"
        echo "  4. 华为云可用: https://<your_code>.mirror.swr.myhuaweicloud.com"
        echo "     （登录华为云查看https://console.huaweicloud.com/swr/?region=cn-north-4#/swr/mirror）"
        echo "-----------------------------------------------------------"
        echo "请输入自定义的 Docker 镜像加速源（每行一个如http://aa.bb.cc, 输入完成后按 Enter 跳过）："
        echo "==========================================================="

        user_mirrors=()
        while read -e -p "自定义镜像地址 (按 Enter 跳过或结束输入): " line; do
            if [ -z "$line" ]; then
                break
            fi
            if echo "$line" | grep -Eq "^https?://[a-zA-Z0-9.-]+(/[a-zA-Z0-9._~:/?\#\[\]@!\$&\'\(\)\*\+,;=-]*)?$"; then
                if [[ " ${default_mirrors[*]} " =~ " $line " ]]; then
                    echo "输入的地址 '$line' 已存在于默认镜像中，跳过添加。"
                else
                    user_mirrors+=("\"$line\"")
                fi
            else
                echo "输入的地址 '$line' 格式无效，请确保以 'http://' 或 'https://' 开头并符合 URL 格式。"
            fi
        done
        # 组合用户输入和默认值
        combined_mirrors=("${user_mirrors[@]}")
        for mirror in "${default_mirrors[@]}"; do
            combined_mirrors+=("\"$mirror\"")
        done

        # 备份现有配置文件
        if [ -f "$config_file" ]; then
            echo "备份现有配置文件到 $config_file.bak"
            mv "$config_file" "$config_file.bak"
        fi

        # 写入新配置文件
        echo "正在生成新的 Docker 配置文件..."
        {
            echo "{"
            echo "  \"registry-mirrors\": ["
            echo "    $(IFS=,; echo "${combined_mirrors[*]}")"
            echo "  ]"
            echo "}"
        } > "$config_file"

        # 重启 Docker 服务
        systemctl daemon-reload
        systemctl restart docker
        systemctl status docker | head -n 16
        echo "Docker 配置已更新, 你可到 $config_file 查看配置内容。"

    }

    if [ -f /etc/os-release ]; then
        . /etc/os-release
    else
        log_task_status ended "当前操作系统不支持"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_task_status in_progress "检测到Docker 未安装..."
        while true; do
            read -p "是否需要安装 Docker? [y/n]" install_docker
            install_docker=${install_docker,,}
            if [[ "$install_docker" == "y" || "$install_docker" == "yes" ]]; then
                log_task_status in_progress "开始安装Docker..."
                break
            elif [[ "$install_docker" == "n" || "$install_docker" == "no" ]]; then
                log_task_status in_progress "docker 安装已取消"
                return
            else
                echo "无效输入，请输入 y(yes) 或 n(no)。"
            fi
        done
        if [ "$ID" = "centos" ] ; then
            yum_install_docker
        elif [ "$ID" = "ubuntu" ] || [ "$ID" = "debian" ]; then
            apt_install_docker
        else
            log_task_status ended "当前操作系统不支持"
            exit 1
        fi
        log_task_status completed "Docker 安装完成，版本：$(docker --version)"
    else
        log_task_status completed "检测到Docker 已安装，版本：$(docker --version)"
    fi

    while true; do
        read -p "是否需要配置 Docker 加速源？[y/n]" change_docker_daemon
        change_docker_daemon=${change_docker_daemon,,}
        if [[ "$change_docker_daemon" == "y" || "$change_docker_daemon" == "Y" || "$change_docker_daemon" == "yes" || "$change_docker_daemon" == "YES" ]]; then
            change_docker_daemon
            break
        elif [[ "$change_docker_daemon" == "n" || "$change_docker_daemon" == "no" ]]; then
            log_task_status in_progress "操作已取消，Docker 镜像加速源保持你的原配置。"
            break
        else
            echo "无效输入，请输入 y(yes) 或 n(no)。"
        fi
    done
    
    # pull images for compilation
    docker_pull_image(){
        log_task_status in_progress "正在拉取用于Malefic编译的Docker镜像..."
        SOURCE_IMAGE=${SOURCE_IMAGE:="chainreactors/malefic-builder:v0.1.0"}
        FINAL_IMAGE=${FINAL_IMAGE:="ghcr.io/chainreactors/malefic-builder:v0.1.0"}
        docker pull $SOURCE_IMAGE
        docker tag $SOURCE_IMAGE $FINAL_IMAGE
        if [ "$SOURCE_IMAGE" != "$FINAL_IMAGE" ]; then
            docker rmi $SOURCE_IMAGE
        fi
        log_task_status completed "Docker 镜像拉取完成."
    }
    docker_pull_image
}

# install malice-network's artifacts
install_malice_network() {
    local PROXY_PREFIX="https://ghfast.top/"
    local MALICE_NETWORK=${MALICE_NETWORK:="v0.1.0"}
    local md="${IoM_ROOT_DIR}/malice-network"
    local MALICE_NETWORK_RELEASES_URL=${MALICE_NETWORK_RELEASES_URL:="${PROXY_PREFIX}https://github.com/chainreactors/malice-network/releases/download/$MALICE_NETWORK"}
    local FILES=(
        "malice_network_linux_amd64"
        "iom_linux_amd64"
        "malice_checksums.txt"
    )
    # --- Init Install Directory ---
    mkdir -p "$md"
    pushd "${md}"
    
    # --- Download Malice Network Components ---
    log_task_status "in_progress" "Downloading Malice Network components..."
    
    # Download all necessary files
    for file in "${FILES[@]}"; do
        download_file "$MALICE_NETWORK_RELEASES_URL/$file" "$file"
    done
    download_file "${PROXY_PREFIX}https://raw.githubusercontent.com/chainreactors/malice-network/$MALICE_NETWORK/server/config.yaml" "config.yaml"

    log_task_status "completed" "All components downloaded successfully."

    # --- Verify Checksums ---
    log_task_status "in_progress" "Verifying the downloaded files..."
    grep -E "linux_amd64" "malice_checksums.txt" | sha256sum -c - 2>/dev/null 
    rm -f "malice_checksums.txt"
    log_task_status "completed" 'Files verified successfully.'
    # --- Make downloaded files executable ---
    log_task_status "in_progress" "Setting executable permissions on downloaded files..."
    chmod +x "malice_network_linux_amd64" "iom_linux_amd64"
    log_task_status "completed" "Malice Network installation completed successfully!"
}
# install malefic's artifacts、sourcecode
install_malefic(){
    local PROXY_PREFIX="https://ghfast.top/"
    local MALEFIC_VERSION=${MALEFIC_VERSION:="v0.1.0"}
    local MALEFIC_ROOT_DIR="$IoM_ROOT_DIR/malefic"
    
    install_source_code(){
        local MALEFIC_REPO_URL="${PROXY_PREFIX}https://github.com/chainreactors/malefic"
        local MALEFIC_PROTO="${PROXY_PREFIX}https://github.com/chainreactors/proto"
        local SRC_DIR="${MALEFIC_ROOT_DIR}/build/src"
        if [ -d "${SRC_DIR}" ]; then
            BACKUP_DIR="${MALEFIC_ROOT_DIR}/build/src_backup_$(date +%Y%m%d_%H%M%S)"
            mv "$SRC_DIR" "$BACKUP_DIR"
            log_task_status in_progress "${SRC_DIR} 存在，已备份到 ${BACKUP_DIR}，如果你不需要可以删除此目录"
        fi
        git clone --branch $MALEFIC_VERSION --depth=1 "${MALEFIC_REPO_URL}" "${SRC_DIR}"
        pushd "${SRC_DIR}"
        rm -rf proto
        git clone --branch $MALEFIC_VERSION --depth=1 "${MALEFIC_PROTO}" "proto"
        popd
        log_task_status "completed" "Source code downloaded successfully!"
    }
    
    install_resources(){
        # install win kit lib
        local MALEFIC_RELEASES_URL=${MALEFIC_RELEASES_URL:="${PROXY_PREFIX}https://github.com/chainreactors/malefic/releases/download/$MALEFIC_VERSION"}
        local FILES=(
            "resources.zip"
        )
        local md="${MALEFIC_ROOT_DIR}/build/src/resources"
        pushd "${md}"
        for file in "${FILES[@]}"; do
            download_file "$MALEFIC_RELEASES_URL/$file" "$file"
        done
        unzip resources.zip && rm -f resources.zip
        log_task_status "completed" 'Resources files downloaded successfully!'
        popd
    }

    install_source_code # before install resources
    install_resources
    
}

create_systemd_service(){
    local SERVER_FILE="${IoM_ROOT_DIR}/malice-network/malice_network_linux_amd64"
    local LOG_DIR="/var/log/malice-network"
    mkdir -p "$LOG_DIR"
    chmod 755 "$LOG_DIR"
    cat > /etc/systemd/system/malice-network.service <<-EOF
[Unit]
Description=Malice Network Service
After=network.target
StartLimitIntervalSec=0

[Service]
WorkingDirectory=$IoM_ROOT_DIR/malice-network
Restart=always
RestartSec=5
User=root
ExecStart=$SERVER_FILE -i $ip_address

StandardOutput=append:$LOG_DIR/debug.log
StandardError=append:$LOG_DIR/error.log

[Install]
WantedBy=multi-user.target
EOF

    chown root:root /etc/systemd/system/malice-network.service
    chmod 600 /etc/systemd/system/malice-network.service

    # --- Reload systemd and start the service ---
    log_task_status "in_progress" "正在启动服务..."
    systemctl daemon-reload
    systemctl enable malice-network
    systemctl start malice-network
    systemctl status malice-network
    # --- Show the final status ---
    log_task_status "in_progress" "你的项目根目录: $IoM_ROOT_DIR"
    log_task_status "in_progress" "Server日志: $LOG_DIR/debug.log"
    log_task_status "completed" "安装完成"
}

if [[ "$EUID" -ne 0 ]]; then
    echo "Please run as root"
    exit 1
fi

# --- get Ip ---
setup_environment
# --- Install Malice Network ---
install_malice_network
# --- Install Malefic ---
install_malefic
# --- Install Docker if not installed ---
check_and_install_docker
# --- Create systemd service ---
create_systemd_service