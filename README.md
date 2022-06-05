# SweetLisa

Center side infrastructure for RDA.

## Special Usage

**Subscription Flags**

Flags can be added to the subscription link.

```
# filter out ipv4
https://sweetlisa.tuta.cc/api/ticket/<your user ticket>/sub/4

# filter out ipv6, and force to show the quotas
https://sweetlisa.tuta.cc/api/ticket/<your user ticket>/sub/6,quota

# force to disable the quotas
https://sweetlisa.tuta.cc/api/ticket/<your user ticket>/sub/noquota

# filter out endpoint
https://sweetlisa.tuta.cc/api/ticket/<your user ticket>/sub/endpoint

# filter out relay
https://sweetlisa.tuta.cc/api/ticket/<your user ticket>/sub/relay
```

**Telegram Commands**

In the anonymous channel, you can send following commands:

1. `/sweetlisa`: show the link of management.
2. `/verify <verification code>`: verify qualification.
3. `/revoke <ticket>`: revoke your ticket immediately.

## Setup

You can set up your own SweetLisa at your own machines.

### Prepare

**Domain**

1. A domain like `sweetlisa.tuta.cc`.
2. Proxy your domain by the CDN (we only support Cloudflare now).
3. Set URL Rewrite rules to your SweetLisa domain in Transform Rules in Cloudflare:
   1. `(ip.geoip.country eq "CN" and http.user_agent ne "BitterJohn" and http.host eq "yourdomain")`, path rewrite to `block-cn`, and query rewrite to empty
   2. `(ip.geoip.country eq "CN" and http.user_agent ne "BitterJohn" and http.host eq "yourdomain" and not http.request.uri.path contains "/api/")`, path rewrite to `block-cn`, and query rewrite to empty
4. Set Page Rules to your SweetLisa domain in Cloudflare:
   1. `yourdomain/block-cn-html,Forwarding URL,https://e14914c0-6759-480d-be89-66b7b7676451.github.io/blocked-page/cn.html`
   2. `yourdomain/block-cn,Forwarding URL,https://e14914c0-6759-480d-be89-66b7b7676451.github.io/blocked-page/cn.txt`
5. Delete all your Firewall rules of your SweetLisa domain because the firewall can log the accesses.
6. Set the TXT record as a API Token of Cloudflare at `cdn-validate.yourdomain`, which should have privilege to access to the `Account - Account Rulesets:Read` and `Page Rules:Read, Firewall Services:Read` to validate your CDN Settings. 

**Telegram**

1. A bot token from @BotFather.
2. An anonymous channel with your bot. 

### Systemd

```unit file (systemd)
# /etc/systemd/system/SweetLisa.service
[Unit]
Description=SweetLisa Service
After=network.target

[Service]
Type=simple
User=root
Restart=always
ExecStart=/usr/bin/SweetLisa -a 127.0.0.1:14914 --bot-token <yourtoken> --host <yourdomain> --log-level info --cn-proxy <optional>

[Install]
WantedBy=multi-user.target
```
