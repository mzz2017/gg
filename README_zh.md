# gg (go-graft)

| [English](README.md)     |
| ------------------------ |
| [中文简体](README_zh.md) |

gg 是一个命令行工具，可在 Linux 环境下对任意命令进行一键代理，而无需安装 v2ray 等其他工具。

你只需要在想代理的命令之前添加 `gg` 即可，例如: `gg python -m pip install torch`.

感谢 [graftcp](https://github.com/hmgle/graftcp) 带来的灵感，gg 是它的一个纯 Go 语言实现，并且拥有更多的有用特性。

**我为什么编写 go-graft？**

我已经厌倦了我在科研和开发中所遇到的糟糕的网络状况。但我并不希望在我的几台工作服务器上安装 v2ray，因为它太笨重了，且配置麻烦。

因此，我需要一个轻巧便携的命令行工具来帮助我在各种服务器上下载和安装依赖项和软件。

**优势**

相比较于 proxychains 或 graftcp，go-graft 拥有以下优势:

1. gg 下载即用，不需要安装任何额外的工具。
2. 支持 UDP，从而有效应对 DNS 污染。
3. 支持 Go 语言编写的程序。见 [applications built by Go can not be hook by proxychains-ng](https://github.com/rofl0r/proxychains-ng/issues/199) 。

## 安装

1. 运行如下命令下载安装 go-graft 最新的版本：

    ```bash
    # curl -L https://github.com/mzz2017/gg/raw/main/release/go.sh | sudo bash
    # 使用镜像以加速：
    curl -L https://github.com/mzz2017/gg/raw/main/release/go.sh | sudo bash
    ```

   > 如果安装完毕后 gg 命令运行 `失败`，请检查 `$PATH`.
   >
   > 你也可以创建一个到 /usr/bin 的软链接。
   >
   > 例如：
   >
   > ```bash
   > sudo ln -s /usr/local/bin/gg /usr/bin/gg
   > ```
2. 测试安装是否成功:
   ```bash
   $ gg --version
   gg version 0.1.1
   ```

## 使用方法

**例如：**

配置你的订阅地址:

```bash
gg config -w subscription=https://example.com/path/to/sub
```

克隆 linux 仓库来试试效果：

```bash
gg git clone --depth=1 https://github.com/torvalds/linux.git
```

输出:

> ```
> Cloning into 'linux'...
> ...
> Receiving objects: 100% (78822/78822), 212.19 MiB | 7.04 MiB/s, done.
> Resolving deltas: 100% (7155/7155), done.
> ```

### 临时使用

**使用节点的分享链接**

```bash
# 如果你之前没有写过配置项，将会提示你输入节点链接。
gg wget -O frp.tar.gz https://github.com/fatedier/frp/releases/download/v0.38.0/frp_0.38.0_linux_amd64.tar.gz
```

> ```
> Enter the share-link of your proxy: ********
> ...
> Saving to: ‘frp.tar.gz’
> frp.tar.gz 100%[=====================================================>] 8.44M 12.2MB/s in 0.7s    
> 2021-12-06 09:21:08 (12.2 MB/s) - ‘frp.tar.gz’ saved [8848900/8848900]
> ```

或者显式地使用 `--node`:

```bash
gg --node ss://YWVzLTEyOC1nY206MQ@example.com:17247 speedtest
```

> ```
> Retrieving speedtest.net configuration...
> Testing from Microsoft (13.xx.xx.xx)...
> ...
> Hosted by xxx: 55.518 ms
> Testing download speed................................................................................
> Download: 104.83 Mbit/s
> Testing upload speed......................................................................................................
> Upload: 96.35 Mbit/s
> ```

**使用订阅地址**

默认情况下 gg 会从你的订阅中自动挑选第一个可用的节点：

```bash
gg --subscription https://example.com/path/to/sub docker pull caddy
```

> ```
> Using default tag: latest
> latest: Pulling from library/caddy
> 97518928ae5f: Pull complete
> 23ccae726125: Pull complete
> 3de6a61c89ac: Pull complete
> 39ed957bdc00: Pull complete
> 0ae44c2d42dd: Pull complete
> Digest: sha256:46f11f4601ecb4c5a37d6014ad51f5cbfeb92b70f5c9ec6c2ac39c4c1a325588
> Status: Downloaded newer image for caddy:latest
> docker.io/library/caddy:latest
> ```

你也可以手动选择节点：

```bash
gg --subscription https://example.com/path/to/sub --select curl ipv4.appspot.com
```

> ```
> Select to connect:
> [ ] 253ms - Azure US West
> [x] 51ms - Azure HK
> [ ] 70ms - xTom IIJ JP
> 13.141.150.163
> ```

### 长期使用

你可以使用 `-w` 来写入配置项：

设置订阅地址：

```bash
gg config -w subscription=https://example.com/path/to/sub
gg curl ipv4.appspot.com
```

> ```
> 13.141.150.163
> ```

设置节点链接:

```bash
gg config -w node=vmess://MY_VMESS_SERVER_SHARE_LINK
gg curl ipv4.appspot.com
```

> ```
> 53.141.112.10
> ```

列出所有配置项:

```bash
gg config
```

> ```
> node=
> subscription.link=https://example.com/path/to/sub
> subscription.select=first
> subscription.cache_last_node=true
> cache.subscription.last_node=trojan-go://MY_TROJAN_GO_SERVER_SHARE_LINK
> no_udp=false
> test_node_before_use=true
> ```

读取某个特定配置项:

```bash
gg config node
```

> ```
> vmess://MY_VMESS_SERVER_SHARE_LINK
> ```

## 问与答

1. Q: 当我使用 `sudo gg xxx` 时，尽管我已经设置了配置项，它还是每次都要求我都输入分享链接。怎样解决这个问题？

   A: 使用 `sudo -E gg xxx` 即可。

## 支持列表

### 操作系统/架构

- [x] Linux/amd64
- [x] Linux/arm（但未测试）
- [x] Linux/arm64（但未测试）
- [ ] Linux/386

### 协议类型

- [x] HTTP(S)
- [x] Socks5
- [x] VMess(AEAD, alterID=0) / VLESS
    - [x] tcp
    - [x] ws
    - [x] tls
    - [ ] grpc
- [x] Shadowsocks
    - [x] AEAD Ciphers
    - [x] Simple-obfs (not tested)
    - [ ] Stream Ciphers
- [x] ShadowsocksR
- [x] Trojan
    - [x] Trojan-gfw
    - [x] Trojan-go

### 订阅类型

- [x] base64 (v2rayN, etc.)
- [ ] SIP008
- [ ] clash
- [ ] surge
- [ ] Quantumult
- [ ] Quantumult X
