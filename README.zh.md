# mip

默认 registry 拉不动的时候，换一条真正可用的路。

[English](README.md)

`mip` 是一个很小的 CLI，专门处理 `docker pull` 卡住、超时、慢到不可用，或者公共 registry 路由不稳定的问题。你不用改项目里的镜像名，也不用把 CI 脚本写成一堆临时 mirror 规则。把原来的镜像交给 `mip`，它会找候选镜像源、探测谁真的可用、通过可用路径拉取，再把镜像标回原来的名字。

```bash
mip pull nginx:1.27
```

可以临时在终端里救急，也可以放进 CI；想知道一个镜像为什么在你这里拉不下来，也可以先让它探测一遍。

## 它解决什么问题

容器镜像已经是构建流程的一部分，但拉镜像这件事依然很脆弱：

- Docker Hub 在当前网络里很慢，甚至不可达。
- 一个公共 mirror 对这个镜像可用，对下一个镜像就不一定。
- Docker Hub、GHCR、Quay、MCR、Kubernetes 镜像的 mirror 路径规则各不相同。
- CI 还没跑测试，就因为基础镜像没拉下来失败。
- 手动改写一次镜像地址能救急，但脚本里从此多了一段没人想维护的 URL。

`mip` 把这些细节挡在前面。它理解镜像引用，知道常见公共 registry 的 mirror 规则，会并发探测候选地址，最后把真正的拉取交给 Docker、Podman 或 nerdctl。

## 先跑起来

看看一个镜像有哪些可用路线：

```bash
mip probe nginx:1.27 --timeout 8s
```

预览它会被改写成哪些候选地址：

```bash
mip rewrite nginx:1.27 --all
```

通过最佳可用 mirror 拉取，并保留原始镜像标签：

```bash
mip pull hello-world:latest --timeout 8s
```

需要指定平台或运行时：

```bash
mip pull hello-world:latest --platform linux/amd64 --retries 2
mip pull hello-world:latest --engine podman --dry-run
```

## 它不只是字符串替换

`mip` 会检查候选地址是否真的能提供你要的 manifest，支持带平台的 manifest list，记录基础 mirror 健康状态，并在已知 manifest digest 时校验拉取结果。

它内置了 Docker Hub、GHCR、Quay、MCR、Kubernetes、GCR、Elastic、NVCR、DHI、Ollama 等常见公共 registry 的默认规则。开箱即用；需要更精细控制时，再加自己的偏好配置。

```bash
mip mirrors list --registry registry.k8s.io
mip config show
```

## 安装

### Homebrew

```bash
brew install vlln/tap/mip
mip version
```

### GitHub Release

通过安装脚本安装最新 GitHub Release：

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | sh
mip version
```

默认会优先安装到 `/usr/local/bin`，不可写时安装到 `$HOME/.local/bin`。
设置 `MIP_BINDIR` 可以指定其他安装目录。

## 配置

刚开始不需要配置文件。默认 mirror 规则已经嵌入二进制文件，也保存在 [configs/mip.yaml](configs/mip.yaml)。

需要本地策略时，创建其中一个文件：

- `$XDG_CONFIG_HOME/mip/config.yaml`
- `~/.config/mip/config.yaml`

示例：

```yaml
prefer:
  - company-cache
exclude:
  - dockerproxy.cool
registries:
  docker.io:
    mirrors:
      - registry.example.com/docker.io
```

`mip` 还会把轻量的 mirror 健康状态记录在：

- `$XDG_STATE_HOME/mip/state.json`
- `~/.local/state/mip/state.json`

状态文件读写失败时，`mip` 会提示警告，但不会中断当前操作。

## Shell 补全

Shell 补全的作用是：在终端里按 Tab 时，让 shell 自动提示 `mip` 的命令、参数和子命令。

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
mip completion fish > ~/.config/fish/completions/mip.fish
```

## Agent Skill

本仓库包含一个 Agent Skill，方便 AI agent 帮你诊断和修复容器镜像拉取问题。

```sh
skit install --global vlln/mip/skills/image-mirror-skill
```

安装本仓库里的全部 skills：

```sh
skit install --global vlln/mip --all
```

手动安装：将 [skills/image-mirror-skill/](skills/image-mirror-skill/) 复制到你的 agent skills 目录。

## 开发

```bash
make test
make build
./bin/mip version
```

创建本地 release 压缩包：

```bash
make release VERSION=0.1.0
ls dist/
```

## 要求

- Docker、Podman 或 nerdctl，用于真实镜像拉取。
- 可以访问选定 registry 和 mirror 的网络环境。
- Go 1.22+，仅开发构建需要。

## 许可证

`mip` 代码和 `skills/image-mirror-skill` 使用 MIT 许可证。
