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
    log_task_status "in_progress" "Downloading from $url to $dest"
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
        log_task_status ended "Unsupported OS"
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
            log_task_status ended "Current operating system is not supported"
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

    log_task_status in_progress "Installing required tools: ${missing[*]}"
    install_packages "${missing[@]}"
    log_task_status completed "Required tools are ready"
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
                log_task_status completed "systemd service installation enabled"
            else
                log_task_status ended "systemd was requested but is not available"
                exit 1
            fi
            ;;
        n|no|false|0)
            INSTALL_SYSTEMD=false
            log_task_status completed "systemd service installation skipped by user"
            ;;
        auto|"")
            if ! is_systemd_available; then
                INSTALL_SYSTEMD=false
                log_task_status completed "systemd is not available. The server will not be installed as a service"
                return
            fi

            if [[ -t 0 ]]; then
                while true; do
                    read -p "Install and start the systemd service? [Y/n] " use_systemd
                    use_systemd="${use_systemd,,}"
                    if [ -z "$use_systemd" ] || [[ "$use_systemd" == "y" || "$use_systemd" == "yes" ]]; then
                        INSTALL_SYSTEMD=true
                        break
                    elif [[ "$use_systemd" == "n" || "$use_systemd" == "no" ]]; then
                        INSTALL_SYSTEMD=false
                        break
                    else
                        echo "Invalid input, please enter y(yes) or n(no)."
                    fi
                done
            else
                INSTALL_SYSTEMD=true
                log_task_status completed "No interactive shell detected. systemd service installation is enabled"
            fi
            ;;
        *)
            log_task_status ended "Invalid INSTALL_SYSTEMD value: $mode"
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
            read -p "Please input your IP Address for the server to start [default: $default_ip]: " input_ip
            ip_address=${input_ip:-$default_ip}
        else
            ip_address=$default_ip
            log_task_status "completed" "No interactive shell detected. Using default IP Address: $ip_address"
        fi
        log_task_status completed "Using IP Address: $ip_address"
    }

    set_base_dir() {
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
    setup_paths
    set_server_ip
}

check_and_install_docker() {
    log_task_status in_progress "Malefic's build can use the following two methods at least one:"
    echo "  1. Docker (install docker and compile image)"
    echo "  2. Github Action (configure reference: https://chainreactors.github.io/wiki/IoM/manual/manual/deploy/#config)"

    yum_install_docker() {
        yum install -y yum-utils ca-certificates curl
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum makecache fast
        yum install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }

    apt_install_docker() {
        apt update -y
        apt install -y ca-certificates curl
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL "https://download.docker.com/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
        chmod a+r /etc/apt/keyrings/docker.asc
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/$ID $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
        apt update -y
        apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    }

    load_os_release
    if ! command -v docker >/dev/null 2>&1; then
        log_task_status in_progress "Docker is not installed..."
        if [[ -t 0 ]]; then
            while true; do
                read -p "Do you want to install Docker? [y/n] " install_docker
                install_docker=${install_docker,,}
                if [[ "$install_docker" == "y" || "$install_docker" == "yes" ]]; then
                    log_task_status in_progress "Starting Docker installation..."
                    break
                elif [[ "$install_docker" == "n" || "$install_docker" == "no" ]]; then
                    log_task_status in_progress "Docker installation canceled"
                    return
                else
                    echo "Invalid input, please enter y(yes) or n(no)."
                fi
            done
        else
            log_task_status completed "No interactive shell detected. Installing Docker by default"
        fi

        case "$ID" in
            centos|rhel|rocky|almalinux)
                yum_install_docker
                ;;
            ubuntu|debian)
                apt_install_docker
                ;;
            *)
                log_task_status ended "Current operating system is not supported"
                exit 1
                ;;
        esac
        log_task_status completed "Docker installation complete, version: $(docker --version)"
    else
        log_task_status completed "Docker is already installed, version: $(docker --version)"
    fi

    docker_pull_image() {
        log_task_status in_progress "Pulling the Docker image for Malefic compilation..."
        SOURCE_IMAGE=${SOURCE_IMAGE:="ghcr.io/chainreactors/malefic-builder:latest"}
        FINAL_IMAGE=${FINAL_IMAGE:="ghcr.io/chainreactors/malefic-builder:latest"}
        docker pull "$SOURCE_IMAGE"
        docker tag "$SOURCE_IMAGE" "$FINAL_IMAGE"
        if [ "$SOURCE_IMAGE" != "$FINAL_IMAGE" ]; then
            docker rmi "$SOURCE_IMAGE"
        fi
        log_task_status completed "Docker image pulled successfully!"
    }

    docker_pull_image
}

install_malice_network() {
    local releases_url="${MALICE_NETWORK_RELEASES_URL:-https://github.com/chainreactors/malice-network/releases/latest/download}"
    local checksum_asset="${MALICE_CHECKSUM_ASSET:-malice_checksums.txt}"
    mkdir -p "$INSTALL_DIR"
    pushd "$INSTALL_DIR" >/dev/null

    if [ -L "$SERVER_COMPAT_BIN_NAME" ]; then
        rm -f "$SERVER_COMPAT_BIN_NAME"
    fi

    log_task_status "in_progress" "Downloading Malice Network components..."
    download_file "$releases_url/$SERVER_COMPAT_BIN_NAME" "$SERVER_COMPAT_BIN_NAME"
    download_file "$releases_url/$CLIENT_BIN_NAME" "$CLIENT_BIN_NAME"
    log_task_status "completed" "All components downloaded successfully."

    if download_optional_file "$releases_url/$checksum_asset" "$checksum_asset"; then
        local verify_entries
        verify_entries=$(awk -v server="$SERVER_COMPAT_BIN_NAME" -v client="$CLIENT_BIN_NAME" '$2 == server || $2 == client { print }' "$checksum_asset")
        if [ -n "$verify_entries" ]; then
            log_task_status "in_progress" "Verifying the downloaded files..."
            printf "%s\n" "$verify_entries" | sha256sum -c -
            log_task_status "completed" "Files verified successfully."
        else
            log_task_status "completed" "Checksum file downloaded, but matching entries were not found. Skipping verification."
        fi
        rm -f "$checksum_asset"
    else
        rm -f "$checksum_asset"
        log_task_status "completed" "Checksum file not found. Skipping verification."
    fi

    log_task_status "in_progress" "Setting executable permissions on downloaded files..."
    if [ -e "$SERVER_COMPAT_BIN_NAME" ] && [ -e "$SERVER_BIN_NAME" ] && [ "$SERVER_COMPAT_BIN_NAME" -ef "$SERVER_BIN_NAME" ]; then
        rm -f "$SERVER_COMPAT_BIN_NAME"
    else
        mv -f "$SERVER_COMPAT_BIN_NAME" "$SERVER_BIN_NAME"
    fi
    ln -sfn "$SERVER_BIN_NAME" "$SERVER_COMPAT_BIN_NAME"
    chmod +x "$SERVER_BIN_NAME" "$CLIENT_BIN_NAME"
    log_task_status "completed" "Malice Network installation completed successfully!"
    popd >/dev/null
}

install_malefic() {
    local releases_url="${MALEFIC_RELEASES_URL:-https://github.com/chainreactors/malefic/releases/latest/download}"
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
        log_task_status ended "malefic.zip does not contain the expected malefic/source_code structure"
        exit 1
    fi

    if [ -d "$MALEFIC_DIR" ]; then
        backup_dir="${INSTALL_DIR}/malefic_backup_$(date +%Y%m%d_%H%M%S)"
        mv "$MALEFIC_DIR" "$backup_dir"
        log_task_status in_progress "$MALEFIC_DIR exists, backed up to $backup_dir. You may delete this directory if it is no longer needed."
    fi

    mv "$tmp_dir/malefic" "$MALEFIC_DIR"
    rm -rf "$tmp_dir"
    log_task_status completed "Malefic source bundle installed successfully at $MALEFIC_DIR/source_code"
}

install_evilclaw() {
    if [[ -t 0 ]]; then
        while true; do
            read -p "Do you want to install EvilClaw (LLM Agent C2 Bridge)? [y/n] " install_ec
            install_ec=${install_ec,,}
            if [[ "$install_ec" == "y" || "$install_ec" == "yes" ]]; then
                break
            elif [[ "$install_ec" == "n" || "$install_ec" == "no" ]]; then
                log_task_status "completed" "EvilClaw installation skipped"
                return
            else
                echo "Invalid input, please enter y(yes) or n(no)."
            fi
        done
    else
        log_task_status "completed" "No interactive shell detected. Skipping EvilClaw installation."
        return
    fi

    local EVILCLAW_DIR="${IoM_ROOT_DIR}/evilclaw"
    local EVILCLAW_VERSION="${EVILCLAW_VERSION:-}"
    local EVILCLAW_ARCHIVE_NAME
    local EVILCLAW_RELEASES_URL

    if [ -n "$EVILCLAW_VERSION" ]; then
        EVILCLAW_ARCHIVE_NAME="${EVILCLAW_ARCHIVE_NAME:-EvilClaw_${EVILCLAW_VERSION#v}_linux_amd64.tar.gz}"
        EVILCLAW_RELEASES_URL="${EVILCLAW_RELEASES_URL:-https://github.com/chainreactors/EvilClaw/releases/download/$EVILCLAW_VERSION}"
    else
        EVILCLAW_ARCHIVE_NAME="${EVILCLAW_ARCHIVE_NAME:-EvilClaw_1.0.0_linux_amd64.tar.gz}"
        EVILCLAW_RELEASES_URL="${EVILCLAW_RELEASES_URL:-https://github.com/chainreactors/EvilClaw/releases/latest/download}"
    fi

    mkdir -p "$EVILCLAW_DIR"
    pushd "$EVILCLAW_DIR" >/dev/null
    log_task_status "in_progress" "Downloading EvilClaw..."
    download_file "$EVILCLAW_RELEASES_URL/$EVILCLAW_ARCHIVE_NAME" "evilclaw.tar.gz"
    tar -xzf evilclaw.tar.gz
    rm -f evilclaw.tar.gz
    chmod +x evilclaw
    popd >/dev/null

    INSTALL_EVILCLAW=true
    log_task_status "completed" "EvilClaw downloaded successfully"
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

    log_task_status "in_progress" "Starting the Malice Network service..."
    systemctl daemon-reload
    systemctl enable malice-network
    systemctl start malice-network
    systemctl status malice-network
    log_task_status "in_progress" "Your ROOT DIR : $IoM_ROOT_DIR"
    log_task_status "in_progress" "Working dir    : $INSTALL_DIR"
    log_task_status "in_progress" "Server log     : $LOG_DIR/debug.log"
    log_task_status "completed" "Malice Network service started successfully!"
}

print_manual_start_instructions() {
    echo ""
    log_task_status "completed" "systemd was skipped. Start the server manually with the commands below:"
    echo "  cd \"$INSTALL_DIR\""
    echo "  ./$SERVER_BIN_NAME -i \"$ip_address\""
    echo ""
    echo "  To launch the interactive quickstart wizard instead:"
    echo "  cd \"$INSTALL_DIR\""
    echo "  ./$SERVER_BIN_NAME --quickstart"
    echo ""
    echo "  To run it in the background:"
    echo "  cd \"$INSTALL_DIR\""
    echo "  nohup ./$SERVER_BIN_NAME -i \"$ip_address\" > debug.log 2> error.log &"
    echo ""
    log_task_status "in_progress" "Working dir    : $INSTALL_DIR"
    log_task_status "in_progress" "Malefic source : $MALEFIC_DIR/source_code"
}

print_evilclaw_manual_note() {
    if [ "$INSTALL_EVILCLAW" != "true" ]; then
        return
    fi

    local EVILCLAW_DIR="${IoM_ROOT_DIR}/evilclaw"
    echo ""
    log_task_status "completed" "EvilClaw was downloaded only. Automatic configuration and startup are disabled"
    echo "  Binary path: $EVILCLAW_DIR/evilclaw"
    echo "  Configure and start EvilClaw manually after its upstream release is stable."
    echo ""
}

if [[ "$EUID" -ne 0 ]]; then
    echo "Please run as root"
    exit 1
fi

INSTALL_EVILCLAW=false

ensure_prerequisites
setup_environment
configure_systemd_mode
install_malice_network
install_malefic
install_evilclaw
check_and_install_docker

if [ "$INSTALL_SYSTEMD" = "true" ]; then
    create_systemd_service
    print_evilclaw_manual_note
else
    print_manual_start_instructions
    print_evilclaw_manual_note
fi
