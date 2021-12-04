# gg (go-graft)
gg is a portable tool to redirect the traffic of a given program to your modern proxy without installing any other programs.

gg was inspired by [graftcp](https://github.com/hmgle/graftcp), and is a pure golang implementation with more useful features.

**Why I created go-graft?**

I am so tired of the poor network condition in my research and development. So I need a light and portable command-line tool to help me download and install dependencies and software on various servers.

**Advantages**

Compared to proxychains or graftcp, we have the following advantages:

1. Use it independently without any other proxy utils.
2. UDP support.
3. Support golang programs. See [applications built by Go can not be hook by proxychains-ng](https://github.com/rofl0r/proxychains-ng/issues/199).

## [WIP] Installation

1. Run this command to download the current stable release of Docker Compose:

    ```bash
    sudo curl -L "https://github.com/mzz2017/gg/releases/download/0.1.0/gg-$(uname -s)-$(uname -m)" -o /usr/local/bin/gg
    sudo chmod +x /usr/local/bin/gg
    ```

    If the command gg fails after installation, check your path. You can also create a symbolic link to /usr/bin or any other directory in your path.

    For example:

    ```bash
    sudo ln -s /usr/local/bin/gg /usr/bin/gg
    ```
2. Test the installation.
   ```bash
   $ gg --version
   gg version 0.1.0
   ```

## Usage

### Temporarily use

**Use share-link**

```bash
$ gg git clone https://github.com/mzz2017/gg.git
Enter the share-link of your proxy: ********
Cloning into 'gg'...
...
Receiving objects: 100% (100/100), 72.10 KiB | 403.00 KiB/s, done.
Resolving deltas: 100% (36/36), done.
```

Or use `--link`: 

```bash
$ gg --link ss://your_share_link_of_a_node git clone https://github.com/mzz2017/gg.git
Cloning into 'gg'...
...
Receiving objects: 100% (100/100), 72.10 KiB | 403.00 KiB/s, done.
Resolving deltas: 100% (36/36), done.
```

**[WIP] Use subscription**

Use the first available node:
```bash
$ gg --subscription https://your_subscription_link curl ipv4.appspot.com
13.141.150.163
```

Select node manually:
```bash
$ gg --subscription https://your_subscription_link --select curl ipv4.appspot.com
Select to connect:
[ ] 253ms - Azure US West
[x] 51ms - Azure HK
[ ] 70ms - xTom IIJ JP  
13.141.150.163
```

Automatically select the fastest node:
```bash
$ gg --subscription https://your_subscription_link --fast curl ipv4.appspot.com
13.141.150.163
```

### [WIP] Long-term use

Write a config variable with `-w`:
```bash
$ gg config -w subscription=https://your_subscription_link
$ gg curl ipv4.appspot.com
13.141.150.163
```
```bash
$ gg config -w link=vmess://your_share_link_of_a_node
$ gg curl ipv4.appspot.com
53.141.112.10
```

Read a config variable:
```bash
$ gg config link
ss://your_share_link_of_a_node
```

List config variables:
```bash
$ gg config
subscription=https://your_subscription_link
subscription.select=fast
cache.subscription.lastlink=trojan://the_link_of_a_node
```

## Support List

### OS/Arch

- [x] Linux/amd64
- [ ] Linux/386
- [ ] Linux/arm64
- [ ] Linux/arm

### Protocol

- [ ] VMess
  - [x] vmess + tcp (AEAD)
- [x] Shadowsocks
  - [x] AEAD Ciphers
  - [ ] Stream Ciphers
- [ ] Trojan
- [ ] Trojan-go
- [ ] HTTP(S)
- [ ] Socks5

### [WIP] Subscription

- [ ] base64 (v2rayN, etc.)
- [ ] SIP008
- [ ] clash
- [ ] surge
- [ ] Quantumult
- [ ] Quantumult X

## TODO
1. Use system DNS as the fallback.
2. Restore the IP of connect that family is AF_LINKLAYER and others.