# mip & gip

默认 registry 拉不动的时候，换一条真正可用的路。
GitHub 访问慢的时候，自动走最快的镜像。

[English](README.md)

## mip — 容器镜像加速

`mip` 是一个很小的 CLI，专门处理 `docker pull` 卡住、超时、慢到不可用的问题。你不用改项目里的镜像名。把镜像交给 `mip`，它会找候选镜像源、探测谁真的可用、通过可用路径拉取，再把镜像标回原来的名字。

```bash
mip pull nginx:1.27
```

可以临时在终端里救急，也可以放进 CI。

### 先跑起来

```bash
mip probe nginx:1.27 --timeout 8s
mip rewrite nginx:1.27 --all
mip pull hello-world:latest --timeout 8s
mip pull hello-world:latest --platform linux/amd64 --retries 2
mip pull hello-world:latest --engine podman --dry-run
```

构建前预拉 Dockerfile 里所有 FROM 镜像：

```bash
mip prefetch
mip prefetch -f path/to/Dockerfile --dry-run
docker build -t myapp .
```

## gip — GitHub 加速

`gip` 对 GitHub 做了 `mip` 对容器镜像做的事。并发探测 GitHub 镜像站，走最快的路 clone、下载文件。

### 通过镜像 clone 仓库

```bash
gip clone https://github.com/user/repo
gip clone https://github.com/user/repo --dry-run
```

### 下载 Release 和 Raw 文件

```bash
gip get https://github.com/user/repo/releases/download/v1.0/binary.tar.gz
gip get https://raw.githubusercontent.com/user/repo/main/file.txt --output myfile.txt
```

### 透明代理（git insteadOf）

自动配置 git 通过最快镜像访问 GitHub：

```bash
gip install
# 之后所有 git clone/fetch/pull 到 github.com 都自动走镜像
gip uninstall
```

### 诊断

```bash
gip probe https://github.com/user/repo --timeout 8s
gip rewrite https://github.com/user/repo --all
gip mirrors list --host github.com
gip config show
```

## 安装

### Homebrew

```bash
brew install vlln/tap/mip
mip version
gip version
```

### GitHub Release

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | sh
mip version
gip version
```

## 配置

刚开始不需要配置文件。默认 mirror 规则已嵌入二进制，保存在 [configs/mip.yaml](configs/mip.yaml) 和 [configs/gip.yaml](configs/gip.yaml)。

需要本地策略时，创建：

- `$XDG_CONFIG_HOME/mip/config.yaml` / `~/.config/mip/config.yaml`
- `$XDG_CONFIG_HOME/gip/config.yaml` / `~/.config/gip/config.yaml`

```yaml
# mip 示例
prefer:
  - company-cache
exclude:
  - dockerproxy.cool
registries:
  docker.io:
    mirrors:
      - registry.example.com/docker.io
```

```yaml
# gip 示例
prefer:
  - my-mirror
mirrors:
  github.com:
    mirrors:
      - mirror.example.com:host-replace
```

健康状态文件：

- `$XDG_STATE_HOME/mip/state.json`
- `$XDG_STATE_HOME/gip/state.json`

## Shell 补全

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
gip completion bash > ~/.local/share/bash-completion/completions/gip
gip completion zsh > ~/.zfunc/_gip
```

## 开发

```bash
make test
make build
./bin/mip version
./bin/gip version
```

创建本地 release 压缩包：

```bash
make release VERSION=0.1.0
ls dist/
```

## 要求

- `mip`：Docker、Podman 或 nerdctl
- `gip`：git
- Go 1.22+，仅开发构建需要

## 许可证

MIT