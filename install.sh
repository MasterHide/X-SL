#!/bin/bash

# Color codes
red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

# Configurable paths
INSTALL_DIR="${INSTALL_DIR:-/usr/local/x-sl}"
BIN_DIR="${BIN_DIR:-/usr/bin}"
LOG_FILE="/var/log/x-sl-install.log"

# Logging functions
LOGE() {
    echo -e "${red}[ERROR] $* ${plain}" | tee -a "$LOG_FILE"
}

LOGI() {
    echo -e "${green}[INFO] $* ${plain}" | tee -a "$LOG_FILE"
}

# Ensure script is run as root
[[ $EUID -ne 0 ]] && LOGE "Fatal error: Please run this script with root privileges." && exit 1

# Create log file
touch "$LOG_FILE" || { LOGE "Failed to create log file: $LOG_FILE"; exit 1; }

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    LOGE "Failed to check the system OS. Please contact the author!"
    exit 1
fi
LOGI "The OS release is: $release"

# Function to detect CPU architecture
arch() {
    case "$(uname -m)" in
    x86_64 | x64 | amd64) echo 'amd64' ;;
    i*86 | x86) echo '386' ;;
    armv8* | armv8 | arm64 | aarch64) echo 'arm64' ;;
    armv7* | armv7 | arm) echo 'armv7' ;;
    armv6* | armv6) echo 'armv6' ;;
    armv5* | armv5) echo 'armv5' ;;
    s390x) echo 's390x' ;;
    *) LOGE "Unsupported CPU architecture: $(uname -m)" && exit 1 ;;
    esac
}

LOGI "Detected architecture: $(arch)"

# Function to generate a random string
gen_random_string() {
    local length="$1"
    local random_string=$(LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w "$length" | head -n 1)
    if [[ -z "$random_string" ]]; then
        LOGE "Failed to generate a random string. Please check /dev/urandom."
        exit 1
    fi
    echo "$random_string"
}

# Function to install base dependencies
install_base() {
    LOGI "Installing base dependencies..."
    case "${release}" in
    ubuntu | debian | armbian)
        apt-get update && apt-get install -y -q wget curl tar tzdata socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    centos | almalinux | rocky | ol)
        yum -y update && yum install -y -q wget curl tar tzdata socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    fedora | amzn)
        dnf -y update && dnf install -y -q wget curl tar tzdata socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    arch | manjaro | parch)
        pacman -Syu --noconfirm wget curl tar tzdata socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    opensuse-tumbleweed)
        zypper refresh && zypper -q install -y wget curl tar timezone socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    *)
        apt-get update && apt install -y -q wget curl tar tzdata socat fail2ban || { LOGE "Failed to install dependencies."; exit 1; }
        ;;
    esac
    LOGI "Base dependencies installed successfully."
}

# Function to configure X-SL after installation
config_after_install() {
    LOGI "Configuring X-SL after installation..."
    local settings_output=$(/usr/local/x-sl/x-sl setting -show true)
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to retrieve X-SL settings. Please check if X-SL is installed correctly."
        exit 1
    fi

    local existing_username=$(echo "$settings_output" | grep -Eo 'Owner username: .+' | awk '{print $2}')
    local existing_password=$(echo "$settings_output" | grep -Eo 'Owner password: .+' | awk '{print $2}')
    local existing_webBasePath=$(echo "$settings_output" | grep -Eo 'webBasePath: .+' | awk '{print $2}')
    local existing_port=$(echo "$settings_output" | grep -Eo 'port: .+' | awk '{print $2}')
    local server_ip=$(curl -s https://api.ipify.org)

    if [[ ${#existing_webBasePath} -lt 4 ]]; then
        if [[ "$existing_username" == "admin" && "$existing_password" == "admin" ]]; then
            local config_webBasePath=$(gen_random_string 15)
            local config_username=$(gen_random_string 10)
            local config_password=$(gen_random_string 10)

            read -p "Would you like to customize the Panel Port settings? (If not, a random port will be applied) [y/n]: " config_confirm
            if [[ "${config_confirm}" == "y" || "${config_confirm}" == "Y" ]]; then
                read -p "Please set up the X-SL panel port: " config_port
                LOGI "Your X-SL Panel Port is: ${config_port}"
            else
                local config_port=$(shuf -i 1024-62000 -n 1)
                LOGI "Generated random port: ${config_port}"
            fi

            /usr/local/x-sl/x-sl setting -username "${config_username}" -password "${config_password}" -port "${config_port}" -webBasePath "${config_webBasePath}"
            LOGI "This is a fresh installation, generating random login info for security concerns:"
            echo -e "╭───────────────────────────────────────────────────────────────────────────────────────╮"
            LOGI "Username: ${config_username}"
            LOGI "Password: ${config_password}"
            LOGI "Port: ${config_port}"
            LOGI "WebBasePath: ${config_webBasePath}"
            LOGI "Access URL: http://${server_ip}:${config_port}/${config_webBasePath}"
            echo -e "╰───────────────────────────────────────────────────────────────────────────────────────╯"
            LOGI "If you forgot your login info, you can type 'x-sl settings' to check"
        else
            local config_webBasePath=$(gen_random_string 15)
            LOGI "WebBasePath is missing or too short. Generating a new one..."
            /usr/local/x-sl/x-sl setting -webBasePath "${config_webBasePath}"
            LOGI "New WebBasePath: ${config_webBasePath}"
            LOGI "Your Access URL: http://${server_ip}:${existing_port}/${config_webBasePath}"
        fi
    else
        if [[ "$existing_username" == "admin" && "$existing_password" == "admin" ]]; then
            local config_username=$(gen_random_string 10)
            local config_password=$(gen_random_string 10)

            LOGI "Default credentials detected. Security update required..."
            /usr/local/x-sl/x-sl setting -username "${config_username}" -password "${config_password}"
            LOGI "Generated new random login credentials:"
            echo -e "╭┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈╮"
            LOGI "Username: ${config_username}"
            LOGI "Password: ${config_password}"
            echo -e "╰┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈┈╯"
            LOGI "If you forgot your login info, you can type 'x-sl settings' to check"
        else
            LOGI "Username, Password, and WebBasePath are properly set. Exiting..."
        fi
    fi

    /usr/local/x-sl/x-sl migrate
}

# Function to install X-SL
install_x-sl() {
    LOGI "Starting X-SL installation..."
    cd /usr/local/ || { LOGE "Failed to change directory to /usr/local/"; exit 1; }

    if [[ $# == 0 ]]; then
        tag_version=$(curl -Ls "https://api.github.com/repos/MasterHide/X-SL/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ -z "$tag_version" ]]; then
            LOGE "Failed to fetch X-SL version. Please check your internet connection or GitHub API status."
            exit 1
        fi
        LOGI "Got X-SL latest version: ${tag_version}, beginning the installation..."
        wget -N --no-check-certificate -O /usr/local/x-sl-linux-$(arch).tar.gz "https://github.com/MasterHide/X-SL/releases/download/${tag_version}/x-sl-linux-$(arch).tar.gz"
        if [[ $? -ne 0 ]]; then
            LOGE "Downloading X-SL failed. Please ensure your server can access GitHub."
            exit 1
        fi
    else
        tag_version=$1
        tag_version_numeric=${tag_version#v}
        min_version="1.0.1"

        if [[ "$(printf '%s\n' "$min_version" "$tag_version_numeric" | sort -V | head -n1)" != "$min_version" ]]; then
            LOGE "Please use a newer version (at least v1.0.1). Exiting installation."
            exit 1
        fi

        url="https://github.com/MasterHide/X-SL/releases/download/${tag_version}/x-sl-linux-$(arch).tar.gz"
        if ! curl --output /dev/null --silent --head --fail "$url"; then
            LOGE "Invalid URL: $url. Please check if the version exists."
            exit 1
        fi
        LOGI "Beginning to install X-SL $1"
        wget -N --no-check-certificate -O /usr/local/x-sl-linux-$(arch).tar.gz "$url"
        if [[ $? -ne 0 ]]; then
            LOGE "Download X-SL $1 failed. Please check if the version exists."
            exit 1
        fi
    fi

    if [[ -e /usr/local/x-sl/ ]]; then
        systemctl stop x-sl
        rm -rf /usr/local/x-sl/
    fi

    tar zxvf x-sl-linux-$(arch).tar.gz -C "$INSTALL_DIR"
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to extract X-SL archive. Please check the file integrity."
        exit 1
    fi
    rm x-sl-linux-$(arch).tar.gz -f
    cd "$INSTALL_DIR" || { LOGE "Failed to change directory to $INSTALL_DIR"; exit 1; }

    # Check the system's architecture and rename the file accordingly
    if [[ $(arch) == "armv5" || $(arch) == "armv6" || $(arch) == "armv7" ]]; then
        mv bin/xray-linux-$(arch) bin/xray-linux-arm
        chmod +x bin/xray-linux-arm
    fi

    chmod +x x-sl bin/xray-linux-$(arch)
    cp -f x-sl.service /etc/systemd/system/
    wget --no-check-certificate -O /usr/bin/x-sl https://raw.githubusercontent.com/MasterHide/X-SL/main/x-sl.sh
    chmod +x /usr/local/x-sl/x-sl.sh
    chmod +x /usr/bin/x-sl
    config_after_install

    systemctl daemon-reload
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to reload systemd daemon. Please check the systemd configuration."
        exit 1
    fi

    systemctl enable x-sl
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to enable X-SL service. Please check the systemd configuration."
        exit 1
    fi

    systemctl start x-sl
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to start X-SL service. Please check the logs for more details."
        exit 1
    fi

    LOGI "X-SL ${tag_version} installation completed, running now..."
    echo -e ""
    echo -e "┌───────────────────────────────────────────────────────┐
│  ${blue}X-SL control menu usages (subcommands):${plain}              │
│                                                       │
│  ${blue}x-sl${plain}              - Admin Management Script          │
│  ${blue}x-sl start${plain}        - Start                            │
│  ${blue}x-sl stop${plain}         - Stop                             │
│  ${blue}x-sl restart${plain}      - Restart                          │
│  ${blue}x-sl status${plain}       - Current Status                   │
│  ${blue}x-sl settings${plain}     - Current Settings                 │
│  ${blue}x-sl enable${plain}       - Enable Autostart on OS Startup   │
│  ${blue}x-sl disable${plain}      - Disable Autostart on OS Startup  │
│  ${blue}x-sl log${plain}          - Check logs                       │
│  ${blue}x-sl banlog${plain}       - Check Fail2ban ban logs          │
│  ${blue}x-sl update${plain}       - Update                           │
│  ${blue}x-sl legacy${plain}       - legacy version                   │
│  ${blue}x-sl install${plain}      - Install                          │
│  ${blue}x-sl uninstall${plain}    - Uninstall                        │
└───────────────────────────────────────────────────────┘"
}

# Main script execution
LOGI "Running X-SL installation..."
install_base
install_x-sl "$1"
