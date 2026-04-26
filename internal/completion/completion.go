package completion

import "fmt"

func Script(shell string) (string, error) {
	switch shell {
	case "bash":
		return bash, nil
	case "zsh":
		return zsh, nil
	case "fish":
		return fish, nil
	default:
		return "", fmt.Errorf("unsupported shell %q; supported shells: bash, zsh, fish", shell)
	}
}

const bash = `# bash completion for mip
_mip_completion() {
  local cur prev commands
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  commands="version pull rewrite probe mirrors config completion help"

  case "$prev" in
    --engine)
      COMPREPLY=( $(compgen -W "docker podman nerdctl" -- "$cur") )
      return 0
      ;;
    --platform)
      COMPREPLY=( $(compgen -W "linux/amd64 linux/arm64 linux/arm/v7 linux/arm/v6 linux/386 linux/ppc64le linux/s390x" -- "$cur") )
      return 0
      ;;
    --registry)
      COMPREPLY=( $(compgen -W "docker.io ghcr.io quay.io mcr.microsoft.com registry.k8s.io gcr.io docker.elastic.co nvcr.io" -- "$cur") )
      return 0
      ;;
  esac

  if [[ "$cur" == -* ]]; then
    COMPREPLY=( $(compgen -W "--config --json --dry-run --no-retag --no-verify-digest --engine --platform --timeout --pull-timeout --concurrency --all --plain --registry" -- "$cur") )
    return 0
  fi

  if [[ "$COMP_CWORD" -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
    return 0
  fi
}
complete -F _mip_completion mip
`

const zsh = `#compdef mip
_mip() {
  local -a commands
  commands=(
    'version:show version'
    'pull:pull image'
    'rewrite:rewrite image'
    'probe:probe image'
    'mirrors:list mirrors'
    'config:show config'
    'completion:generate completion'
    'help:show help'
  )
  _arguments \
    '--config[config file]:file:_files' \
    '--json[emit JSON]' \
    '--dry-run[show plan]' \
    '--no-retag[keep mirror tag]' \
    '--no-verify-digest[skip digest verification]' \
    '--engine[engine]:(docker podman nerdctl)' \
    '--platform[platform]:(linux/amd64 linux/arm64 linux/arm/v7 linux/arm/v6 linux/386 linux/ppc64le linux/s390x)' \
    '--timeout[probe timeout]' \
    '--pull-timeout[pull timeout]' \
    '--concurrency[probe concurrency]' \
    '--all[all candidates]' \
    '--plain[plain output]' \
    '--registry[registry]:(docker.io ghcr.io quay.io mcr.microsoft.com registry.k8s.io gcr.io docker.elastic.co nvcr.io)' \
    '1:command:->command' \
    '*::arg:->args'
  case $state in
    command) _describe 'commands' commands ;;
  esac
}
_mip "$@"
`

const fish = `# fish completion for mip
complete -c mip -f
complete -c mip -n '__fish_use_subcommand' -a 'version pull rewrite probe mirrors config completion help'
complete -c mip -l config -r -d 'config file'
complete -c mip -l json -d 'emit JSON'
complete -c mip -l dry-run -d 'show plan'
complete -c mip -l no-retag -d 'keep mirror tag'
complete -c mip -l no-verify-digest -d 'skip digest verification'
complete -c mip -l engine -r -a 'docker podman nerdctl'
complete -c mip -l platform -r -a 'linux/amd64 linux/arm64 linux/arm/v7 linux/arm/v6 linux/386 linux/ppc64le linux/s390x'
complete -c mip -l timeout -r -d 'probe timeout'
complete -c mip -l pull-timeout -r -d 'pull timeout'
complete -c mip -l concurrency -r -d 'probe concurrency'
complete -c mip -l all -d 'all candidates'
complete -c mip -l plain -d 'plain output'
complete -c mip -l registry -r -a 'docker.io ghcr.io quay.io mcr.microsoft.com registry.k8s.io gcr.io docker.elastic.co nvcr.io'
`
