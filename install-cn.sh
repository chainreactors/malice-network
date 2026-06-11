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
    curl --retry 4 --silent --show-error --fail -L -o "$dest" "$url"
}

download_optional_file() {
    local url="$1"
    local dest="$2"
    curl --retry 4 --silent --show-error --fail -L -o "$dest" "$url" 2>/dev/null
}

load_os_release() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
    else
        log_task_status ended "当前操作系统不支持"
        exit 1
    fi
}

install_packages() {
    load_os_release
    case "$ID" in
        centos|rhel|rocky|almalinux)
            yum install -y "$@"
            ;;
        ubuntu|debian)
            apt update -y
            apt install -y "$@"
            ;;
        *)
            log_task_status ended "当前操作系统不支持"
            exit 1
            ;;
    esac
}

ensure_prerequisites() {
    local missing=()
    local tool
    for tool in curl unzip; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing+=("$tool")
        fi
    done

    if [ "${#missing[@]}" -eq 0 ]; then
        return
    fi

    log_task_status in_progress "正在安装基础依赖: ${missing[*]}"
    install_packages "${missing[@]}"
    log_task_status completed "基础依赖已准备完成"
}

detect_default_ip() {
    local detected_ip=""

    detected_ip=$(curl -4 --noproxy "*" --silent --show-error --fail https://ifconfig.me 2>/dev/null || true)
    if [ -z "$detected_ip" ]; then
        detected_ip=$(curl -4 --noproxy "*" --silent --show-error --fail https://api.ipify.org 2>/dev/null || true)
    fi
    if [ -z "$detected_ip" ] && command -v hostname >/dev/null 2>&1; then
        detected_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    if [ -z "$detected_ip" ]; then
        detected_ip="127.0.0.1"
    fi

    printf "%s\n" "$detected_ip"
}

is_systemd_available() {
    command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]
}

configure_systemd_mode() {
    local mode="${INSTALL_SYSTEMD:-auto}"
    mode="${mode,,}"

    case "$mode" in
        y|yes|true|1)
            if is_systemd_available; then
                INSTALL_SYSTEMD=true
                log_task_status completed "已启用 systemd 服务安装"
            else
                log_task_status ended "已要求安装 systemd 服务，但当前环境不支持 systemd"
                exit 1
            fi
            ;;
        n|no|false|0)
            INSTALL_SYSTEMD=false
            log_task_status completed "已跳过 systemd 服务安装"
            ;;
        auto|"")
            if ! is_systemd_available; then
                INSTALL_SYSTEMD=false
                log_task_status completed "当前环境不可用 systemd，将跳过服务安装"
                return
            fi

            if [[ -t 0 ]]; then
                while true; do
                    read -p "是否安装并启动 systemd 服务？[Y/n] " use_systemd
                    use_systemd="${use_systemd,,}"
                    if [ -z "$use_systemd" ] || [[ "$use_systemd" == "y" || "$use_systemd" == "yes" ]]; then
                        INSTALL_SYSTEMD=true
                        break
                    elif [[ "$use_systemd" == "n" || "$use_systemd" == "no" ]]; then
                        INSTALL_SYSTEMD=false
                        break
                    else
                        echo "无效输入，请输入 y(yes) 或 n(no)。"
                    fi
                done
            else
                INSTALL_SYSTEMD=true
                log_task_status completed "检测到非交互环境，默认启用 systemd 服务安装"
            fi
            ;;
        *)
            log_task_status ended "INSTALL_SYSTEMD 参数无效: $mode"
            exit 1
            ;;
    esac
}

setup_paths() {
    INSTALL_DIR="${IoM_ROOT_DIR}/malice-network"
    SERVER_BIN_NAME="malice-network_linux_amd64"
    SERVER_COMPAT_BIN_NAME="malice_network_linux_amd64"
    CLIENT_BIN_NAME="iom_linux_amd64"
    MALEFIC_DIR="${INSTALL_DIR}/malefic"
}

setup_environment() {
    set_server_ip() {
        local default_ip
        default_ip=$(detect_default_ip)
        if [[ -t 0 ]]; then
            read -p "请输入你的服务器公网(或内网)IP地址 [default: $default_ip]: " input_ip
            ip_address=${input_ip:-$default_ip}
        else
            ip_address=$default_ip
            log_task_status "completed" "无交互输入，将使用默认 IP 地址: $ip_address"
        fi
        log_task_status "completed" "使用IP地址：$ip_address"
    }

    set_base_dir() {
        local DEFAULT_DIR="/opt/iom"
        if [[ -t 0 ]]; then
            read -p "请输入安装的根目录 [默认: $DEFAULT_DIR]: " input_dir
            IoM_ROOT_DIR=${input_dir:-$DEFAULT_DIR}
        else
            IoM_ROOT_DIR=$DEFAULT_DIR
            log_task_status "completed" "无交互输入，将使用默认根目录：$IoM_ROOT_DIR"
        fi
        log_task_status completed "使用根目录：$IoM_ROOT_DIR"
    }

    set_base_dir
    setup_paths
    set_server_ip
}

check_and_install_docker() {
    log_task_status in_progress "Malefic 的自动编译至少需要以下两种方式之一:"
    echo "  1. Docker (安装 Docker 以及编译镜像)"
    echo "  2. Github Action (配置参考: https://chainreactors.github.io/wiki/IoM/manual/manual/deploy/#config)"

    yum_install_docker() {
        yum install -y yum-utils ca-certificates curl
        yum-config-manager --add-repo http://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
        yum makecache fast
        yum install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }

    apt_install_docker() {
        apt update -y
        apt install -y ca-certificates curl
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL "https://mirrors.aliyun.com/docker-ce/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
        chmod a+r /etc/apt/keyrings/docker.asc
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://mirrors.aliyun.com/docker-ce/linux/$ID $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
        apt update -y
        apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    }

    change_docker_daemon() {
        local config_file="/etc/docker/daemon.json"
        mkdir -p "$(dirname "$config_file")"

        if ! command -v systemctl >/dev/null 2>&1; then
            log_task_status completed "当前环境没有 systemctl，跳过 Docker 镜像加速配置"
            return
        fi

        local default_mirrors=(
            "https://mirror.ccs.tencentyun.com"
            "https://dockerhub.azk8s.cn"
            "https://docker.1ms.run"
            "https://docker.xuanyuan.me"
        )

        echo "==========================================================="
        echo "可以根据当前的服务器厂商添加 Docker 镜像加速源，以加速 Docker 镜像的拉取。"
        echo "以下是一些常见的云厂商内部镜像地址供参考："
        echo "  1. 腾讯云服务器可用: https://mirror.ccs.tencentyun.com"
        echo "  2. Azure 可用: https://dockerhub.azk8s.cn"
        echo "  3. 阿里云可用: https://<your_code>.mirror.aliyuncs.com"
        echo "     （登录阿里云查看 https://cr.console.aliyun.com/cn-hongkong/instances/mirrors）"
        echo "  4. 华为云可用: https://<your_code>.mirror.swr.myhuaweicloud.com"
        echo "     （登录华为云查看 https://console.huaweicloud.com/swr/?region=cn-north-4#/swr/mirror）"
        echo "-----------------------------------------------------------"
        echo "请输入自定义的 Docker 镜像加速源（每行一个，输入完成后按 Enter 跳过）："
        echo "==========================================================="

        local user_mirrors=()
        while read -e -p "自定义镜像地址 (按 Enter 跳过或结束输入): " line; do
            if [ -z "$line" ]; then
                break
            fi
            if echo "$line" | grep -Eq "^https?://[a-zA-Z0-9.-]+(/[a-zA-Z0-9._~:/?\#\[\]@!\$&'\(\)\*\+,;=-]*)?$"; then
                if [[ " ${default_mirrors[*]} " =~ " $line " ]]; then
                    echo "输入的地址 '$line' 已存在于默认镜像中，跳过添加。"
                else
                    user_mirrors+=("\"$line\"")
                fi
            else
                echo "输入的地址 '$line' 格式无效，请确保以 http:// 或 https:// 开头。"
            fi
        done

        local combined_mirrors=("${user_mirrors[@]}")
        local mirror
        for mirror in "${default_mirrors[@]}"; do
            combined_mirrors+=("\"$mirror\"")
        done

        if [ -f "$config_file" ]; then
            echo "备份现有配置文件到 $config_file.bak"
            mv "$config_file" "$config_file.bak"
        fi

        echo "正在生成新的 Docker 配置文件..."
        {
            echo "{"
            echo "  \"registry-mirrors\": ["
            echo "    $(IFS=,; echo "${combined_mirrors[*]}")"
            echo "  ]"
            echo "}"
        } > "$config_file"

        systemctl daemon-reload
        systemctl restart docker
        systemctl status docker | head -n 16
        echo "Docker 配置已更新，你可以到 $config_file 查看配置内容。"
    }

    load_os_release
    if ! command -v docker >/dev/null 2>&1; then
        log_task_status in_progress "检测到 Docker 未安装..."
        if [[ -t 0 ]]; then
            while true; do
                read -p "是否需要安装 Docker? [y/n] " install_docker
                install_docker=${install_docker,,}
                if [[ "$install_docker" == "y" || "$install_docker" == "yes" ]]; then
                    log_task_status in_progress "开始安装 Docker..."
                    break
                elif [[ "$install_docker" == "n" || "$install_docker" == "no" ]]; then
                    log_task_status in_progress "Docker 安装已取消"
                    return
                else
                    echo "无效输入，请输入 y(yes) 或 n(no)。"
                fi
            done
        else
            log_task_status completed "检测到非交互环境，默认安装 Docker"
        fi

        case "$ID" in
            centos|rhel|rocky|almalinux)
                yum_install_docker
                ;;
            ubuntu|debian)
                apt_install_docker
                ;;
            *)
                log_task_status ended "当前操作系统不支持"
                exit 1
                ;;
        esac
        log_task_status completed "Docker 安装完成，版本：$(docker --version)"
    else
        log_task_status completed "检测到 Docker 已安装，版本：$(docker --version)"
    fi

    if [[ -t 0 ]]; then
        while true; do
            read -p "是否需要配置 Docker 加速源？[y/n] " update_docker_daemon
            update_docker_daemon=${update_docker_daemon,,}
            if [[ "$update_docker_daemon" == "y" || "$update_docker_daemon" == "yes" ]]; then
                change_docker_daemon
                break
            elif [[ "$update_docker_daemon" == "n" || "$update_docker_daemon" == "no" ]]; then
                log_task_status in_progress "已跳过 Docker 镜像加速源配置，保留原有设置。"
                break
            else
                echo "无效输入，请输入 y(yes) 或 n(no)。"
            fi
        done
    else
        log_task_status completed "检测到非交互环境，跳过 Docker 镜像加速源配置"
    fi

    docker_pull_image() {
        log_task_status in_progress "正在拉取用于 Malefic 编译的 Docker 镜像..."
        FINAL_IMAGE=${FINAL_IMAGE:="ghcr.io/chainreactors/malefic-builder:latest"}
        SOURCE_IMAGE=${SOURCE_IMAGE:="ghcr.1ms.run/${FINAL_IMAGE}"}
        docker pull "$SOURCE_IMAGE"
        docker tag "$SOURCE_IMAGE" "$FINAL_IMAGE"
        if [ "$SOURCE_IMAGE" != "$FINAL_IMAGE" ]; then
            docker rmi "$SOURCE_IMAGE"
        fi
        log_task_status completed "Docker 镜像拉取完成."
    }

    docker_pull_image
}

install_malice_network() {
    local PROXY_PREFIX="https://ghfast.top/"
    local releases_url="${MALICE_NETWORK_RELEASES_URL:-${PROXY_PREFIX}https://github.com/chainreactors/malice-network/releases/latest/download}"
    local checksum_asset="${MALICE_CHECKSUM_ASSET:-malice_checksums.txt}"
    mkdir -p "$INSTALL_DIR"
    pushd "$INSTALL_DIR" >/dev/null

    if [ -L "$SERVER_COMPAT_BIN_NAME" ]; then
        rm -f "$SERVER_COMPAT_BIN_NAME"
    fi

    log_task_status "in_progress" "正在下载 Malice Network 组件..."
    download_file "$releases_url/$SERVER_COMPAT_BIN_NAME" "$SERVER_COMPAT_BIN_NAME"
    download_file "$releases_url/$CLIENT_BIN_NAME" "$CLIENT_BIN_NAME"
    log_task_status "completed" "所有组件下载完成."

    if download_optional_file "$releases_url/$checksum_asset" "$checksum_asset"; then
        local verify_entries
        verify_entries=$(awk -v server="$SERVER_COMPAT_BIN_NAME" -v client="$CLIENT_BIN_NAME" '$2 == server || $2 == client { print }' "$checksum_asset")
        if [ -n "$verify_entries" ]; then
            log_task_status "in_progress" "正在校验下载文件..."
            printf "%s\n" "$verify_entries" | sha256sum -c -
            log_task_status "completed" "文件校验成功."
        else
            log_task_status "completed" "已下载 checksum 文件，但未找到匹配项，跳过校验."
        fi
        rm -f "$checksum_asset"
    else
        rm -f "$checksum_asset"
        log_task_status "completed" "未找到 checksum 文件，跳过校验."
    fi

    log_task_status "in_progress" "正在设置可执行权限..."
    if [ -e "$SERVER_COMPAT_BIN_NAME" ] && [ -e "$SERVER_BIN_NAME" ] && [ "$SERVER_COMPAT_BIN_NAME" -ef "$SERVER_BIN_NAME" ]; then
        rm -f "$SERVER_COMPAT_BIN_NAME"
    else
        mv -f "$SERVER_COMPAT_BIN_NAME" "$SERVER_BIN_NAME"
    fi
    ln -sfn "$SERVER_BIN_NAME" "$SERVER_COMPAT_BIN_NAME"
    chmod +x "$SERVER_BIN_NAME" "$CLIENT_BIN_NAME"
    log_task_status "completed" "Malice Network 安装完成!"
    popd >/dev/null
}

install_malefic() {
    local PROXY_PREFIX="https://ghfast.top/"
    local releases_url="${MALEFIC_RELEASES_URL:-${PROXY_PREFIX}https://github.com/chainreactors/malefic/releases/latest/download}"
    local archive_name="${MALEFIC_ARCHIVE_NAME:-malefic.zip}"
    local archive_path="${INSTALL_DIR}/${archive_name}"
    local tmp_dir
    local backup_dir

    mkdir -p "$INSTALL_DIR"
    download_file "$releases_url/$archive_name" "$archive_path"

    tmp_dir=$(mktemp -d)
    unzip -q "$archive_path" -d "$tmp_dir"
    rm -f "$archive_path"

    if [ ! -d "$tmp_dir/malefic/source_code" ]; then
        rm -rf "$tmp_dir"
        log_task_status ended "malefic.zip 中未找到预期的 malefic/source_code 目录结构"
        exit 1
    fi

    if [ -d "$MALEFIC_DIR" ]; then
        backup_dir="${INSTALL_DIR}/malefic_backup_$(date +%Y%m%d_%H%M%S)"
        mv "$MALEFIC_DIR" "$backup_dir"
        log_task_status in_progress "$MALEFIC_DIR 已存在，已备份到 $backup_dir，如果你不需要可以删除此目录"
    fi

    mv "$tmp_dir/malefic" "$MALEFIC_DIR"
    rm -rf "$tmp_dir"
    log_task_status completed "Malefic 源码包已安装到 $MALEFIC_DIR/source_code"
}

create_systemd_service() {
    local SERVER_FILE="${INSTALL_DIR}/${SERVER_BIN_NAME}"
    local LOG_DIR="/var/log/malice-network"
    mkdir -p "$LOG_DIR"
    chmod 755 "$LOG_DIR"
    cat > /etc/systemd/system/malice-network.service <<-EOF
[Unit]
Description=Malice Network Service
After=network.target
StartLimitIntervalSec=0

[Service]
WorkingDirectory=$INSTALL_DIR
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

    log_task_status "in_progress" "正在启动 Malice Network 服务..."
    systemctl daemon-reload
    systemctl enable malice-network
    systemctl start malice-network
    systemctl status malice-network
    log_task_status "in_progress" "你的项目根目录: $IoM_ROOT_DIR"
    log_task_status "in_progress" "工作目录: $INSTALL_DIR"
    log_task_status "in_progress" "Server 日志: $LOG_DIR/debug.log"
    log_task_status "completed" "安装完成"
}

print_manual_start_instructions() {
    echo ""
    log_task_status "completed" "已跳过 systemd，请使用以下命令手动启动服务："
    echo "  cd \"$INSTALL_DIR\""
    echo "  ./$SERVER_BIN_NAME -i \"$ip_address\""
    echo ""
    echo "  如需进入交互式 quickstart 向导："
    echo "  cd \"$INSTALL_DIR\""
    echo "  ./$SERVER_BIN_NAME --quickstart"
    echo ""
    echo "  如需后台运行："
    echo "  cd \"$INSTALL_DIR\""
    echo "  nohup ./$SERVER_BIN_NAME -i \"$ip_address\" > debug.log 2> error.log &"
    echo ""
    log_task_status "in_progress" "工作目录: $INSTALL_DIR"
    log_task_status "in_progress" "Malefic 源码目录: $MALEFIC_DIR/source_code"
}

if [[ "$EUID" -ne 0 ]]; then
    echo "Please run as root"
    exit 1
fi

ensure_prerequisites
setup_environment
configure_systemd_mode
install_malice_network
install_malefic
check_and_install_docker

if [ "$INSTALL_SYSTEMD" = "true" ]; then
    create_systemd_service
else
    print_manual_start_instructions
fi
