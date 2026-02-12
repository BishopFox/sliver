# Sliver

Sliver 是一个开源、跨平台的 adversary emulation / red team 框架，可供各种规模的组织用于安全测试。

Sliver 的 implants 支持通过 Mutual TLS (mTLS)、WireGuard、HTTP(S) 和 DNS 进行 C2 通信，并且在编译时会为每个二进制文件动态生成独立的非对称加密密钥。

Server 和 Client 支持 MacOS、Windows 和 Linux。Implants 支持 MacOS、Windows 和 Linux（理论上支持所有 Golang 编译目标平台，但尚未全部测试）。

# v1.6.0 / `master`

**注意：** 你当前查看的是 Sliver v1.6.0 的最新 master 分支；新的 PR 应提交到该分支。

但该分支 **目前不推荐用于生产环境**。为了获得最佳使用体验，请使用带有 release tag 的版本。

如果 PR 包含针对 Sliver v1.5 的 bug 修复，请提交到 [`v1.5.x/master` 分支](https://github.com/BishopFox/sliver/tree/v1.5.x/master)。

## 功能特性

- 动态代码生成（Dynamic code generation）
- 编译期混淆（Compile-time obfuscation）
- Multiplayer 模式
- Staged 与 Stageless payloads
- 基于 HTTP(S) 的 Procedurally generated C2
- DNS canary 蓝队检测机制
- 通过 mTLS、WireGuard、HTTP(S) 和 DNS 实现的 Secure C2
- 可通过 JavaScript/TypeScript 或 Python 完全脚本化控制
- Windows 进程迁移（process migration）、process injection、用户 token 操作等
- Let's Encrypt 集成
- 内存中执行 .NET assembly
- COFF/BOF 内存加载器
- TCP 和 named pipe 横向移动（pivot）
- 以及更多功能！

## 快速开始（Getting Started）

下载最新的 release 版本，并查看 Sliver wiki 了解基础安装与使用的快速教程。

如果希望使用最新版本，可以从源码自行编译。

### Linux 一行命令安装

```bash
curl https://sliver.sh/install | sudo bash
```

然后运行：

```bash
sliver
```

## 帮助

请查看官方 wiki，或在 GitHub 上发起 discussion。

我们通常也会出现在 Bloodhound Gang Slack 服务器的 `#golang` 频道。

## 从源码编译

请参考 wiki 中的 Compile from Source 说明。

## 反馈

欢迎填写官方 survey 问卷以提供反馈。

## 许可证 - GPLv3

Sliver 采用 GPLv3 许可证发布。部分子组件可能使用不同的许可证，请查看对应子目录中的具体说明。
