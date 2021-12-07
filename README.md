# gg (go-graft)

| [English](README.md)     |
| ------------------------ |
| [中文简体](README_zh.md) |

gg is a command-line tool for one-click proxy in your research and development.

You can just add `gg` before another command to redirect its traffic to your proxy without installing v2ray or anything else. Usage example: `gg python -m pip install torch`.

It was inspired by [graftcp](https://github.com/hmgle/graftcp), is a pure golang implementation with more useful
features.

**Why did I create go-graft?**

I am so tired of the poor network condition in my research and development. But I do not want to install v2ray in the
working servers because it is too heavy.

Thus, I need a light and portable command-line tool to help me download and install dependencies and software on various
servers.

**Advantages**

Compared to proxychains or graftcp, we have the following advantages:

1. Use it independently without any other proxy utils.
2. UDP support.
3. Support golang programs.
   See [applications built by Go can not be hook by proxychains-ng](https://github.com/rofl0r/proxychains-ng/issues/199)
   .

## Installation

1. Run this command to download the latest release of go-graft:

    ```bash
    # sudo curl -L "https://github.com/mzz2017/gg/releases/latest/download/gg-$(uname -s)-$(uname -m)" -o /usr/local/bin/gg
    # use the mirror:
    sudo curl -L "https://hubmirror.v2raya.org/mzz2017/gg/releases/latest/download/gg-$(uname -s)-$(uname -m)" -o /usr/local/bin/gg
    sudo chmod +x /usr/local/bin/gg
    ```

   > If the command gg `fails` after installation, check your path.
   >
   > You can also create a symbolic link to /usr/bin or any other directory in your path.
   >
   > For example:
   >
   > ```bash
   > sudo ln -s /usr/local/bin/gg /usr/bin/gg
   > ```
2. Test the installation.
   ```bash
   $ gg --version
   gg version 0.1.1
   ```

## Usage

**Examples:**

Configure the subscription:

```bash
gg config -w subscription=https://example.com/path/to/sub
```

Test with cloning linux repo:

```bash
gg git clone --depth=1 https://github.com/torvalds/linux.git
```

Output:

> ```
> Cloning into 'linux'...
> ...
> Receiving objects: 100% (78822/78822), 212.19 MiB | 7.04 MiB/s, done.
> Resolving deltas: 100% (7155/7155), done.
> ```

### Temporarily use

**Use share-link**

```bash
# if no configuration was written before, a share-link will be required to input.
gg wget -O frp.tar.gz https://github.com/fatedier/frp/releases/download/v0.38.0/frp_0.38.0_linux_amd64.tar.gz
```

> ```
> Enter the share-link of your proxy: ********
> ...
> Saving to: ‘frp.tar.gz’
> frp.tar.gz 100%[=====================================================>] 8.44M 12.2MB/s in 0.7s    
> 2021-12-06 09:21:08 (12.2 MB/s) - ‘frp.tar.gz’ saved [8848900/8848900]
> ```

Or use `--node`:

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

**Use subscription**

By default, gg will automatically select the first available node from the subscription:

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

Select the node manually:

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

### Long-term use

Write a config variable with `-w`:

Set subscription:

```bash
gg config -w subscription=https://example.com/path/to/sub
gg curl ipv4.appspot.com
```

> ```
> 13.141.150.163
> ```

Set node:

```bash
gg config -w node=vmess://MY_VMESS_SERVER_SHARE_LINK
gg curl ipv4.appspot.com
```

> ```
> 53.141.112.10
> ```

List config variables:

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

Read specific config variable:

```bash
gg config node
```

> ```
> vmess://MY_VMESS_SERVER_SHARE_LINK
> ```

## Support List

### OS/Arch

- [x] Linux/amd64
- [x] Linux/arm (not tested)
- [ ] Linux/386
- [ ] Linux/arm64

### Protocol

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

### Subscription

- [x] base64 (v2rayN, etc.)
- [ ] SIP008
- [ ] clash
- [ ] surge
- [ ] Quantumult
- [ ] Quantumult X

## TODO

1. Use system DNS as the fallback.
2. Restore the IP of connect that family is AF_LINKLAYER and others.
