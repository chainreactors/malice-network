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
    log_task_status "in_progress" "Downloading $dest..."
    echo $url
    curl --retry 4 --silent -L -o "$dest" "$url" 
}

# check and install docker
check_install_docker(){
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

        echo "Docker 配置已更新, 你可到 $config_file 查看配置内容。"
    }

    if [ -f /etc/os-release ]; then
        . /etc/os-release
    else
        log_task_status ended "当前操作系统不支持"
        exit 1
    fi
    if ! command -v docker &> /dev/null; then
        log_task_status in_progress "检测到Docker 未安装，正在安装..."
        if [ "$ID" = "centos" ] ; then
            yum_install_docker
        elif [ "$ID" = "ubuntu" ] || [ "$ID" = "debian" ]; then
            apt_install_docker
        else
            log_task_status ended "当前操作系统不支持"
            exit 1
        fi
    else
        log_task_status completed "Docker 已安装，跳过安装..."
    fi
    
    echo "是否需要配置 Docker 镜像加速源？[y/n]"
    read -r user_choice
    if [[ "$user_choice" == "y" || "$user_choice" == "Y" || "$user_choice" == "yes" || "$user_choice" == "YES" ]]; then
        change_docker_daemon
    else
        log_task_status in_progress "操作已取消，Docker 镜像加速源保持不变。"
    fi
    
    log_task_status completed "Docker 安装完成，当前 Docker 版本为：$(docker --version)"
}

# pull images for compilation
docker_pull_image(){
    SOURCE_NAME_SPACE=${SOURCE_NAME_SPACE:="chainreactors"}
    FINAL_NAME_SPACE=${FINAL_NAME_SPACE:="ghcr.io/chainreactors"}
    # if you want more images, add them to the array
    images=(
        "x86_64-pc-windows-msvc:nightly-2023-09-18-latest"
        "i686-pc-windows-msvc:nightly-2023-09-18-latest"
        "x86_64-pc-windows-gnu:nightly-2023-09-18-latest"
        "i686-pc-windows-gnu:nightly-2023-09-18-latest"
        "x86_64-unknown-linux-musl:nightly-2023-09-18-latest"
        "i686-unknown-linux-musl:nightly-2023-09-18-latest"
        "aarch64-apple-darwin:nightly-2023-09-18-latest"
    )
    log_task_status in_progress "Pulling Docker image for compilation..."
    for image in "${images[@]}"; do
        log_task_status in_progress "Pulling $image ..."
        docker pull "$SOURCE_NAME_SPACE/$image"
        docker tag "$SOURCE_NAME_SPACE/$image" "$FINAL_NAME_SPACE/$image"       
        if [ "$SOURCE_NAME_SPACE" != "$FINAL_NAME_SPACE" ]; then
                docker rmi "$SOURCE_NAME_SPACE/$image"
        fi
    done
}
# set your server ip

setup_environment(){
    set_server_ip(){
        default_ip=$(curl --noproxy -4 -s ifconfig.me)
        if [[ -t 0 ]]; then
            read -p "Please input your IP Address for the server to start [default: $default_ip]: " input_ip
            ip_address=${input_ip:-$default_ip}
        else
            ip_address=$default_ip
            log_task_status "completed" "No interactive shell detected. Using default IP Address: $ip_address"
        fi
        log_task_status completed "Using IP Address: $ip_address"
    }

    set_base_dir(){
        local DEFAULT_DIR="/opt/iom"
        if [[ -t 0 ]]; then
            read -p "Please input the base directory for the installation [default: $DEFAULT_DIR]: " input_dir
            IoM_ROOT_DIR=${input_dir:-$DEFAULT_DIR}
        else
            IoM_ROOT_DIR=$DEFAULT_DIR
            log_task_status "completed" "No interactive shell detected. Using default base directory: $IoM_ROOT_DIR"
        fi
        log_task_status completed "Using base directory: $IoM_ROOT_DIR"
    }
    set_base_dir
    set_server_ip
}

# install malice-network's artifacts
install_malice_network() {
    local md="${IoM_ROOT_DIR}/malice-network"
    local MALICE_NETWORK_RELEASES_URL=${MALICE_NETWORK_RELEASES_URL:="https://github.com/chainreactors/malice-network/releases/latest/download"}
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
    download_file "https://raw.githubusercontent.com/chainreactors/malice-network/refs/heads/dev/server/config.yaml" "config.yaml"

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
# install malefic's artifacts sourcecode 、sgn 、malefic_mutant
install_malefic(){
    local MALEFIC_ROOT_DIR="$IoM_ROOT_DIR/malefic"
    
    install_source_code(){
        local MALEFIC_REPO_URL="https://github.com/chainreactors/malefic"
        local source_dir="${MALEFIC_ROOT_DIR}/build/src"
        if [ -d "${source_dir}" ]; then
            echo "[+] Backing up existing src directory..."
            mv "$SRC_DIR" "$SRC_DIR.backup"
        fi
        git clone --recurse-submodules --depth=1 "${MALEFIC_REPO_URL}" "${source_dir}"
        log_task_status "completed" "Source code downloaded successfully!"
    }

    install_resources(){
        local MALEFIC_RELEASES_URL=${MALEFIC_RELEASES_URL:="https://github.com/chainreactors/malefic/releases/latest/download"}
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

    install_malefic_mutant(){
        local MALEFIC_RELEASES_URL=${MALEFIC_RELEASES_URL:="https://github.com/chainreactors/malefic/releases/latest/download"}
        local FILES=(
            "malefic-mutant-x86_64-unknown-linux-musl"
        )
        local md="${MALEFIC_ROOT_DIR}/build/bin"
        mkdir -p "$md"
        pushd "${md}"
        for file in "${FILES[@]}"; do
            download_file "$MALEFIC_RELEASES_URL/$file" "$file"
        done
        mv malefic-mutant-x86_64-unknown-linux-musl malefic-mutant && chmod +x malefic-mutant
        log_task_status "completed" "Malefic-Mutant downloaded successfully!"
        popd
    }
    
    install_sgn(){
        local SGN_RELEASES_URL="https://github.com/EgeBalci/sgn/releases/download/v2.0.1/sgn_linux_amd64_2.0.1.zip"
        local md="${MALEFIC_ROOT_DIR}/build/bin"
        mkdir -p "$md"
        pushd "${md}"
        download_file "$SGN_RELEASES_URL" "sgn_linux_amd64_2.0.1.zip"
        unzip sgn_linux_amd64_2.0.1.zip && rm -f sgn_linux_amd64_2.0.1.zip && chmod +x sgn
        popd
        log_task_status "completed" "Sgn downloaded successfully!"
    }

    add_to_path() {
        local new_path="${MALEFIC_ROOT_DIR}/build/bin"
        
        local shell_name=$(basename "$SHELL")
        local rc_file=""
        
        if [[ "$shell_name" == "bash" ]]; then
            rc_file="${HOME}/.bashrc"
        elif [[ "$shell_name" == "zsh" ]]; then
            rc_file="${HOME}/.zshrc"
        else
            log_task_status "ended" "Unsupported shell: $shell_name. Only bash and zsh are supported."
            return 1
        fi

        if grep -q "export PATH=.*${new_path}" "$rc_file"; then
            log_task_status "completed" "Path ${new_path} is already in $rc_file"
        else
            echo "export PATH=\"${new_path}:\$PATH\"" >> "$rc_file"
            log_task_status "completed" "Path ${new_path} has been added to $rc_file"
        fi
        source "$rc_file"
        log_task_status "completed" "Sourced $rc_file successfully!"
    }
    
    install_source_code # before install resources
    install_resources
    install_malefic_mutant
    install_sgn
    add_to_path
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
log_task_status "in_progress" "Starting the Malice Network service..."
systemctl daemon-reload
systemctl enable malice-network
systemctl start malice-network
systemctl status malice-network
log_task_status "completed" "Malice Network service started successfully!"

}

if [[ "$EUID" -ne 0 ]]; then
    echo "Please run as root"
    exit 1
fi

# --- get Ip ---
setup_environment
# --- Install Docker if not installed ---
check_install_docker
# --- Install docker image for compilation ---
docker_pull_image
# --- Install Malice Network ---
install_malice_network
install_malefic
# --- Create systemd service ---
create_systemd_service