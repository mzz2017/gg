# gg (go-graft)
gg is a portable tool to redirect the traffic of a given program to your modern proxy without installing any other programs.

gg was inspired by [graftcp](https://github.com/hmgle/graftcp), and is a pure golang implementation with more useful features.

**Why I created go-graft?**

I am so tired of the poor network condition in my research and development. So I need a light and portable tool to help me download and install dependencies on various servers.

**Advantages**

Compared to proxychains or graftcp, we have the following advantages:

1. Use it independently without any other proxy utils.
2. UDP support.
3. Support golang programs.

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

**Use share-link:**

```bash
$ gg --link vmess://your_share_link_of_a_node curl ipv4.appspot.com
13.141.150.163
```

**[WIP] Use subscription:**

```bash
$ gg --subscription https://your_subscription_link curl ipv4.appspot.com
13.141.150.163
```

**[WIP] Persistent config variables**

Write a config variable with `-w`:
```bash
$ gg config -w link=ss://your_share_link_of_a_node
$ gg curl ipv4.appspot.com
13.141.150.163
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
cache.subscription.lastlink=trojan://the_link_of_a_node
```

## Support List

### Protocol

- [ ] vmess(v2rayN)
  - [x] vmess + tcp (AEAD)
- [x] SS(SIP002)
  - [x] AEAD Ciphers
  - [ ] Stream Ciphers
- [ ] trojan
- [ ] trojan-go
- [ ] http
- [ ] https
- [ ] socks5

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