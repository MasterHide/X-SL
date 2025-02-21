#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

# Logging functions
LOGE() {
    echo -e "${red}[ERROR] $* ${plain}"
}

LOGI() {
    echo -e "${green}[INFO] $* ${plain}"
}

log_file="/var/log/x-sl-install.log"
touch $log_file

# check root
[[ $EUID -ne 0 ]] && LOGE "Fatal error: Please run this script with root privilege \n" && exit 1

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    LOGE "Failed to check the system OS, please contact the author!"
    exit 1
fi
LOGI "The OS release is: $release"

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

LOGI "arch: $(arch)"

os_version=""
os_version=$(grep "^VERSION_ID" /etc/os-release | cut -d '=' -f2 | tr -d '"' | tr -d '.')

if [[ "${release}" == "arch" ]]; then
    LOGI "Your OS is Arch Linux"
elif [[ "${release}" == "parch" ]]; then
    LOGI "Your OS is Parch Linux"
elif [[ "${release}" == "manjaro" ]]; then
    LOGI "Your OS is Manjaro"
elif [[ "${release}" == "armbian" ]]; then
    LOGI "Your OS is Armbian"
elif [[ "${release}" == "alpine" ]]; then
    LOGI "Your OS is Alpine Linux"
elif [[ "${release}" == "opensuse-tumbleweed" ]]; then
    LOGI "Your OS is OpenSUSE Tumbleweed"
elif [[ "${release}" == "openEuler" ]]; then
    if [[ ${os_version} -lt 2203 ]]; then
        LOGE "Please use OpenEuler 22.03 or higher"
        exit 1
    fi
elif [[ "${release}" == "centos" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        LOGE "Please use CentOS 8 or higher"
        exit 1
    fi
elif [[ "${release}" == "ubuntu" ]]; then
    if [[ ${os_version} -lt 2004 ]]; then
        LOGE "Please use Ubuntu 20 or higher version!"
        exit 1
    fi
elif [[ "${release}" == "fedora" ]]; then
    if [[ ${os_version} -lt 36 ]]; then
        LOGE "Please use Fedora 36 or higher version!"
        exit 1
    fi
elif [[ "${release}" == "amzn" ]]; then
    if [[ ${os_version} != "2023" ]]; then
        LOGE "Please use Amazon Linux 2023!"
        exit 1
    fi
elif [[ "${release}" == "debian" ]]; then
    if [[ ${os_version} -lt 11 ]]; then
        LOGE "Please use Debian 11 or higher"
        exit 1
    fi
elif [[ "${release}" == "almalinux" ]]; then
    if [[ ${os_version} -lt 80 ]]; then
        LOGE "Please use AlmaLinux 8.0 or higher"
        exit 1
    fi
elif [[ "${release}" == "rocky" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        LOGE "Please use Rocky Linux 8 or higher"
        exit 1
    fi
elif [[ "${release}" == "ol" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        LOGE "Please use Oracle Linux 8 or higher"
        exit 1
    fi
else
    LOGE "Your operating system is not supported by this script."
    echo "Please ensure you are using one of the following supported operating systems:"
    echo "- Ubuntu 20.04+"
    echo "- Debian 11+"
    echo "- CentOS 8+"
    echo "- OpenEuler 22.03+"
    echo "- Fedora 36+"
    echo "- Arch Linux"
    echo "- Parch Linux"
    echo "- Manjaro"
    echo "- Armbian"
    echo "- AlmaLinux 8.0+"
    echo "- Rocky Linux 8+"
    echo "- Oracle Linux 8+"
    echo "- OpenSUSE Tumbleweed"
    echo "- Amazon Linux 2023"
    exit 1
fi

install_base() {
    case "${release}" in
    ubuntu | debian | armbian)
        apt-get update && apt-get install -y -q wget curl tar tzdata socat fail2ban
        ;;
    centos | almalinux | rocky | ol)
        yum -y update && yum install -y -q wget curl tar tzdata socat fail2ban
        ;;
    fedora | amzn)
        dnf -y update && dnf install -y -q wget curl tar tzdata socat fail2ban
        ;;
    arch | manjaro | parch)
        pacman -Syu && pacman -Syu --noconfirm wget curl tar tzdata socat fail2ban
        ;;
    opensuse-tumbleweed)
        zypper refresh && zypper -q install -y wget curl tar timezone socat fail2ban
        ;;
    *)
        apt-get update && apt install -y -q wget curl tar tzdata socat fail2ban
        ;;
    esac
}

gen_random_string() {
    local length="$1"
    local random_string=$(LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w "$length" | head -n 1)
    echo "$random_string"
}

config_after_install() {
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

install_x-sl() {
    cd /usr/local/

    if [ $# == 0 ]; then
        tag_version=$(curl -Ls "https://api.github.com/repos/MasterHide/X-SL/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$tag_version" ]]; then
            LOGE "Failed to fetch X-SL version. Please check your internet connection or GitHub API status."
            exit 1
        fi
        LOGI "Got x-sl latest version: ${tag_version}, beginning the installation..."
        wget -N --no-check-certificate -O /usr/local/x-sl-linux-$(arch).tar.gz https://github.com/MasterHide/X-SL/releases/download/${tag_version}/x-sl-linux-$(arch).tar.gz
        if [[ $? -ne 0 ]]; then
            LOGE "Downloading x-sl failed, please be sure that your server can access GitHub"
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
        LOGI "Beginning to install x-sl $1"
        wget -N --no-check-certificate -O /usr/local/x-sl-linux-$(arch).tar.gz ${url}
        if [[ $? -ne 0 ]]; then
            LOGE "Download x-sl $1 failed, please check if the version exists"
            exit 1
        fi
    fi

    if [[ -e /usr/local/x-sl/ ]]; then
        systemctl stop x-sl
        rm /usr/local/x-sl/ -rf
    fi

    tar zxvf x-sl-linux-$(arch).tar.gz
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to extract x-sl archive. Please check the file integrity."
        exit 1
    fi
    rm x-sl-linux-$(arch).tar.gz -f
    cd x-sl
    chmod +x x-sl

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
        LOGE "Failed to enable x-sl service. Please check the systemd configuration."
        exit 1
    fi

    systemctl start x-sl
    if [[ $? -ne 0 ]]; then
        LOGE "Failed to start x-sl service. Please check the logs for more details."
        exit 1
    fi

    LOGI "x-sl ${tag_version} installation completed, running now..."
    echo -e ""
    echo -e "┌───────────────────────────────────────────────────────┐
│  ${blue}x-sl control menu usages (subcommands):${plain}              │
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

cleanup() {
    rm -rf /usr/local/x-sl-linux-$(arch).tar.gz
    rm -rf /usr/local/x-sl/
    systemctl stop x-sl
    systemctl disable x-sl
    rm -f /etc/systemd/system/x-sl.service
}

trap cleanup EXIT

LOGI "Running..."
install_base
install_x-sl $1
