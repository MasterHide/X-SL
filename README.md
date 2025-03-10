[English](/README.md)
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/user-attachments/assets/4ae35e02-c849-4645-8012-d18ebb2e02d3">
    <img alt="x-sl" src="https://github.com/user-attachments/assets/4ae35e02-c849-4645-8012-d18ebb2e02d3">
  </picture>
</p>

**An Advanced Web Panel • Built on Xray Core**

[![](https://img.shields.io/github/v/release/mhsanaei/3x-ui.svg)](https://github.com/MHSanaei/3x-ui/releases)
[![](https://img.shields.io/github/actions/workflow/status/mhsanaei/3x-ui/release.yml.svg)](#)
[![GO Version](https://img.shields.io/github/go-mod/go-version/mhsanaei/3x-ui.svg)](#)
[![Downloads](https://img.shields.io/github/downloads/mhsanaei/3x-ui/total.svg)](#)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> **Disclaimer:** This project is only for personal learning and communication, please do not use it for illegal purposes, please do not use it in a production environment

**If this project is helpful to you, you may wish to give it a**:star2:


- USDT (TRC20): `TXncxkvhkDWGts487Pjqq1qT9JmwRUz8CC`


## Install & Upgrade

```
bash <(curl -Ls https://raw.githubusercontent.com/MasterHide/X-SL/main/install.sh)

```

## SSL Certificate  (Enable SSL before running the web panel.)

<details>
  <summary>Click for SSL Certificate details</summary>

### ACME

To manage SSL certificates using ACME:

1. Ensure your domain is correctly resolved to the server.
2. Run the `x-ui` command in the terminal, then choose `Manage SSL Certificates`.
3. You will be presented with the following options:

   - **Get SSL:** Obtain SSL certificates.
   - **Revoke:** Revoke existing SSL certificates.
   - **Force Renew:** Force renewal of SSL certificates.
   - **Show Existing Domains:** Display all domain certificates available on the server.  
   - **Set Certificate Paths for the Panel:** Specify the certificate for your domain to be used by the panel. 

 </details>

## Features 🚀

| Category                | Details                                                                 |
|-------------------------|-------------------------------------------------------------------------|
| **Monitoring & Security** |                                                                         |
| 📊 System Status         | Real-time monitoring of system performance and resource usage          |
| 🚫 Torrent Blocking      | Block public torrent traffic via iptables (reduce the risk)           |
| 📉 Usage Analytics       | Traffic statistics, traffic limits, and client expiration management    |
|                         |                                                                         |
| **Protocol Support**     |                                                                         |
| 🌐 Multi-Protocol         | Supports VMESS, VLESS, Trojan, Shadowsocks, Dokodemo-door, Socks, HTTP  |
| 🔒 Advanced Protocols    | XTLS native support (RPRX-Direct, Vision, REALITY) and WireGuard        |
|                         |                                                                         |
| **Automation**           |                                                                         |
| ⚙️ Server Management     | Automated server boot system and API route fixes                        |
| 🔒 SSL Management        | One-click SSL certificate issuance + automatic renewal                 |
| 🔄 Data Control          | Export/import database functionality                                   |
|                         |                                                                         |
| **Customization**        |                                                                         |
| 🎨 Theme Support         | Dark/Light mode toggle                                                  |
| 🛠️ Configuration         | Customizable Xray templates and panel-driven config adjustments         |
| 🔍 Search                | Full search capability across inbounds and clients                      |
|                         |                                                                         |
| **User Management**      |                                                                         |
| 👥 Multi-User System      | Robust multi-user support with traffic monitoring (Traffic-X)           |
| 🔧 Admin Tools           | Create user settings via API and advanced panel configurations          |


## Nginx Settings
<details>
  <summary>Click for Reverse Proxy Configuration</summary>

#### Nginx Reverse Proxy
```nginx
location / {
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header Range $http_range;
    proxy_set_header If-Range $http_if_range; 
    proxy_redirect off;
    proxy_pass http://127.0.0.1:2053;
}
```

#### Nginx sub-path
- Ensure that the "URI Path" in the `/sub` panel settings is the same.
- The `url` in the panel settings needs to end with `/`.   

```nginx
location /sub {
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header Range $http_range;
    proxy_set_header If-Range $http_if_range; 
    proxy_redirect off;
    proxy_pass http://127.0.0.1:2053;
}
```
</details>

## Recommended OS

- Ubuntu 20.04+
- Debian 11+
- CentOS 8+
- OpenEuler 22.03+
- Fedora 36+
- Arch Linux
- Parch Linux
- Manjaro
- Armbian
- AlmaLinux 8.0+
- Rocky Linux 8+
- Oracle Linux 8+
- OpenSUSE Tubleweed
- Amazon Linux 2023


## Default Panel Settings

<details>
  <summary>Click for default settings details</summary>

### Username, Password, Port, and Web Base Path

If you choose not to modify these settings, they will be generated randomly .

**Default Settings for Docker:**
- **Username:** admin
- **Password:** admin
- **Port:** 2053

### Database Management:

  You can conveniently perform database Backups and Restores directly from the panel.

- **Database Path:**
  - `/etc/x-ui/x-ui.db`


### Web Base Path

1. **Reset Web Base Path:**
   - Open your terminal.
   - Run the `x-ui` command.
   - Select the option to `Reset Web Base Path`.

2. **Generate or Customize Path:**
   - The path will be randomly generated, or you can enter a custom path.

3. **View Current Settings:**
   - To view your current settings, use the `x-ui settings` command in the terminal or `View Current Panel Info` in `x-ui`

### Security Recommendation:
- For enhanced security, use a long, random word in your URL structure.

**Examples:**
- `http://ip:port/*webbasepath*/panel`
- `https://domain:port/*webbasepath*/panel`

</details>

## WARP Configuration

<details>
  <summary>Click for WARP configuration details</summary>

#### Usage

**For versions `v2.1.0` and later:**

WARP is built-in, and no additional installation is required. Simply turn on the necessary configuration in the panel.

</details>



































## API Routes

<details>
  <summary>Click for API routes details</summary>

#### Usage

- `/login` with `POST` user data: `{username: '', password: ''}` for login
- `/panel/api/inbounds` base for following actions:

| Method | Path                               | Action                                      |
| :----: | ---------------------------------- | ------------------------------------------- |
| `GET`  | `"/list"`                          | Get all inbounds                            |
| `GET`  | `"/get/:id"`                       | Get inbound with inbound.id                 |
| `GET`  | `"/getClientTraffics/:email"`      | Get Client Traffics with email              |
| `GET`  | `"/getClientTrafficsById/:id"`     | Get client's traffic By ID |
| `GET`  | `"/createbackup"`                  | Telegram bot sends backup to admins         |
| `POST` | `"/add"`                           | Add inbound                                 |
| `POST` | `"/del/:id"`                       | Delete Inbound                              |
| `POST` | `"/update/:id"`                    | Update Inbound                              |
| `POST` | `"/clientIps/:email"`              | Client Ip address                           |
| `POST` | `"/clearClientIps/:email"`         | Clear Client Ip address                     |
| `POST` | `"/addClient"`                     | Add Client to inbound                       |
| `POST` | `"/:id/delClient/:clientId"`       | Delete Client by clientId\*                 |
| `POST` | `"/updateClient/:clientId"`        | Update Client by clientId\*                 |
| `POST` | `"/:id/resetClientTraffic/:email"` | Reset Client's Traffic                      |
| `POST` | `"/resetAllTraffics"`              | Reset traffics of all inbounds              |
| `POST` | `"/resetAllClientTraffics/:id"`    | Reset traffics of all clients in an inbound |
| `POST` | `"/delDepletedClients/:id"`        | Delete inbound depleted clients (-1: all)   |
| `POST` | `"/onlines"`                       | Get Online users ( list of emails )         |

\*- The field `clientId` should be filled by:

- `client.id` for VMESS and VLESS
- `client.password` for TROJAN
- `client.email` for Shadowsocks

- [<img src="https://run.pstmn.io/button.svg" alt="Run In Postman" style="width: 128px; height: 32px;">](https://app.getpostman.com/run-collection/5146551-dda3cab3-0e33-485f-96f9-d4262f437ac5?action=collection%2Ffork&source=rip_markdown&collection-url=entityId%3D5146551-dda3cab3-0e33-485f-96f9-d4262f437ac5%26entityType%3Dcollection%26workspaceId%3Dd64f609f-485a-4951-9b8f-876b3f917124)
</details>

## Environment Variables

<details>
  <summary>Click for environment variables details</summary>

#### Usage

| Variable       |                      Type                      | Default       |
| -------------- | :--------------------------------------------: | :------------ |
| XUI_LOG_LEVEL  | `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"`      |
| XUI_DEBUG      |                   `boolean`                    | `false`       |
| XUI_BIN_FOLDER |                    `string`                    | `"bin"`       |
| XUI_DB_FOLDER  |                    `string`                    | `"/etc/x-ui"` |
| XUI_LOG_FOLDER |                    `string`                    | `"/var/log"`  |

Example:

```sh
XUI_BIN_FOLDER="bin" XUI_DB_FOLDER="/etc/x-ui" go build main.go
```

</details>

## Preview
![home](https://github.com/user-attachments/assets/1a0187ba-90fa-4518-98d9-7033c475a440)
![inbound](https://github.com/user-attachments/assets/20837943-25d7-4b40-ad9a-0f0d93611b7e)
![usage](https://github.com/user-attachments/assets/98b1dc01-6897-4150-8391-12db8b1beadf)
![result](https://github.com/user-attachments/assets/21c73af1-fc43-4575-be94-4f5df4ed361c)




## A Special Thanks to

- [Project X](https://github.com/XTLS)
- [Inspired by 3x-ui](https://github.com/MHSanaei)


## Acknowledgment

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (License: **GPL-3.0**): _Enhanced v2ray/xray and v2ray/xray-clients routing rules with built-in Iranian domains and a focus on security and adblocking._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (License: **GPL-3.0**): _This repository contains automatically updated V2Ray routing rules based on data on blocked domains and addresses in Russia._


## Stargazers over Time

[![Stargazers over time](https://starchart.cc/MasterHide/X-SL.svg?variant=adaptive)](https://starchart.cc/MasterHide/X-SL)

