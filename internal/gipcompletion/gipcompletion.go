package gipcompletion

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

const bash = `# bash completion for gip
_gip_completion() {
  local cur prev commands
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  commands="version clone install uninstall get rewrite probe mirrors config completion help"

  case "$prev" in
    --host)
      COMPREPLY=( $(compgen -W "github.com gitlab.com" -- "$cur") )
      return 0
      ;;
  esac

  if [[ "$cur" == -* ]]; then
    COMPREPLY=( $(compgen -W "--config --json --dry-run --host --output --timeout --concurrency --all --plain --dir" -- "$cur") )
    return 0
  fi

  if [[ "$COMP_CWORD" -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
    return 0
  fi
}
complete -F _gip_completion gip
`

const zsh = `#compdef gip
_gip() {
  local -a commands
  commands=(
    'version:show version'
    'clone:clone repository'
    'install:install git insteadOf'
    'uninstall:remove git insteadOf'
    'get:download file'
    'rewrite:rewrite URL'
    'probe:probe URL'
    'mirrors:list mirrors'
    'config:show config'
    'completion:generate completion'
    'help:show help'
  )
  _arguments \
    '--config[config file]:file:_files' \
    '--json[emit JSON]' \
    '--dry-run[show plan]' \
    '--host[source host]:(github.com gitlab.com)' \
    '--output[output file]:file:_files' \
    '--timeout[probe timeout]' \
    '--concurrency[probe concurrency]' \
    '--all[all candidates]' \
    '--plain[plain output]' \
    '--dir[target directory]:directory:_files -/' \
    '1:command:->command' \
    '*::arg:->args'
  case $state in
    command) _describe 'commands' commands ;;
  esac
}
_gip "$@"
`

const fish = `# fish completion for gip
complete -c gip -f
complete -c gip -n '__fish_use_subcommand' -a 'version clone install uninstall get rewrite probe mirrors config completion help'
complete -c gip -l config -r -d 'config file'
complete -c gip -l json -d 'emit JSON'
complete -c gip -l dry-run -d 'show plan'
complete -c gip -l host -r -a 'github.com gitlab.com'
complete -c gip -l output -r -d 'output file'
complete -c gip -l timeout -r -d 'probe timeout'
complete -c gip -l concurrency -r -d 'probe concurrency'
complete -c gip -l all -d 'all candidates'
complete -c gip -l plain -d 'plain output'
complete -c gip -l dir -r -d 'target directory'
`