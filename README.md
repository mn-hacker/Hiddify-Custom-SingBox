# Hiddify Custom SingBox

Custom sing-box builds with `with_v2ray_api` tag for Hiddify Panel.

## Why this repository?

Official sing-box releases don't include `with_v2ray_api` tag which is **required** for Hiddify panel to communicate with sing-box core.

## Build Tags

All builds include:
- `with_v2ray_api` ⭐ (Required for Hiddify)
- `with_quic`
- `with_gvisor`
- `with_dhcp`
- `with_wireguard`
- `with_utls`
- `with_acme`
- `with_clash_api`

## Download

Go to [Releases](../../releases) page and download:
- `sing-box-linux-amd64.zip` - for x86_64 servers
- `sing-box-linux-arm64.zip` - for ARM64 servers

## Build New Version

1. Go to **Actions** tab
2. Select **Build Singbox with V2Ray API**
3. Click **Run workflow**
4. Enter:
   - `singbox_version`: Leave empty for latest, or enter specific tag (e.g., `v1.10.0`)
   - `release_tag`: Your release tag (e.g., `v1.10.0.h1`)
5. Click **Run workflow**

The build will automatically create a new release with both amd64 and arm64 binaries.

## Usage in Hiddify Panel

After creating a release, update `packages.lock` in Hiddify Manager:

```
singbox|VERSION|amd64|https://github.com/mn-hacker/Hiddify-Custom-SingBox/releases/download/VERSION/sing-box-linux-amd64.zip|HASH
singbox|VERSION|arm64|https://github.com/mn-hacker/Hiddify-Custom-SingBox/releases/download/VERSION/sing-box-linux-arm64.zip|HASH
```

## License

Same as [sing-box](https://github.com/SagerNet/sing-box) - GPL-3.0
