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
    curl --retry 4 --silent -L -o "$dest" "$url"
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

# check and install docker
check_and_install_docker(){
    log_task_status in_progress "Malefic's build can use the following two methods at least one:"
    echo "  1. Docker (install docker and compile image)"
    echo "  2. Github Action (configure reference: https://chainreactors.github.io/wiki/IoM/manual/manual/deploy/#config)"
    yum_install_docker(){
        yum install -y yum-utils curl unzip git
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum makecache fast
        yum install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }
    apt_install_docker(){
        apt update && apt install -y ca-certificates curl unzip git
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL "https://download.docker.com/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
        chmod a+r /etc/apt/keyrings/docker.asc
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/$ID $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
        apt update -y && apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
    }
    if [ -f /etc/os-release ]; then
        . /etc/os-release
    else
        log_task_status ended "Unsupported OS"
        exit 1
    fi
    if ! command -v docker &> /dev/null; then
        log_task_status in_progress "Docker is not installed..."
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
        if [ "$ID" = "centos" ]; then
            yum_install_docker
        elif [ "$ID" = "ubuntu" ] || [ "$ID" = "debian" ]; then
            apt_install_docker
        else
            log_task_status ended "Current operating system is not supported"
            exit 1
        fi
        log_task_status completed "Docker installation complete, version: $(docker --version)"
    else
        log_task_status completed "Docker is already installed, version: $(docker --version)"
    fi
    # pull images for compilation
    docker_pull_image(){
        log_task_status in_progress "Pulling the Docker image for Malefic compilation..."
        SOURCE_IMAGE=${SOURCE_IMAGE:="ghcr.io/chainreactors/malefic-builder:v0.1.0"}
        FINAL_IMAGE=${FINAL_IMAGE:="ghcr.io/chainreactors/malefic-builder:v0.1.0"}
        docker pull $SOURCE_IMAGE
        docker tag $SOURCE_IMAGE $FINAL_IMAGE
        if [ "$SOURCE_IMAGE" != "$FINAL_IMAGE" ]; then
            docker rmi $SOURCE_IMAGE
        fi
        log_task_status completed "Docker image pulled successfully!"
    }
    docker_pull_image
}

# install malice-network's artifacts
install_malice_network() {
    local MALICE_NETWORK=${MALICE_NETWORK:="v0.1.0"}
    local md="${IoM_ROOT_DIR}/malice-network"
    local MALICE_NETWORK_RELEASES_URL=${MALICE_NETWORK_RELEASES_URL:="https://github.com/chainreactors/malice-network/releases/download/$MALICE_NETWORK"}
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
    download_file "https://raw.githubusercontent.com/chainreactors/malice-network/$MALICE_NETWORK/server/config.yaml" "config.yaml"

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
    local MALEFIC_VERSION=${MALEFIC_VERSION:="v0.1.0"}
    local MALEFIC_ROOT_DIR="$IoM_ROOT_DIR/malefic"
    
    install_source_code(){
        local MALEFIC_REPO_URL="https://github.com/chainreactors/malefic"
        local SRC_DIR="${MALEFIC_ROOT_DIR}/build/src"
        if [ -d "${SRC_DIR}" ]; then
            BACKUP_DIR="${MALEFIC_ROOT_DIR}/build/src_backup_$(date +%Y%m%d_%H%M%S)"
            mv "$SRC_DIR" "$BACKUP_DIR"
            log_task_status in_progress "${SRC_DIR} exists, backed up to ${BACKUP_DIR}. You may delete this directory if it is no longer needed."
        fi
        git clone --branch $MALEFIC_VERSION --recurse-submodules --depth=1 "${MALEFIC_REPO_URL}" "${SRC_DIR}"
        log_task_status "completed" "Source code downloaded successfully!"
    }
    
    install_resources(){
        # install win kit lib
        local MALEFIC_RELEASES_URL=${MALEFIC_RELEASES_URL:="https://github.com/chainreactors/malefic/releases/download/$MALEFIC_VERSION"}
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
    log_task_status "in_progress" "Starting the Malice Network service..."
    systemctl daemon-reload
    systemctl enable malice-network
    systemctl start malice-network
    systemctl status malice-network
    # --- Show the final status ---
    log_task_status "in_progress" "Your ROOT DIR : $IoM_ROOT_DIR"
    log_task_status "in_progress" "Server log : $LOG_DIR/debug.log"
    log_task_status "completed" "Malice Network service started successfully!"

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