#!/bin/sh
set -e

repo="wungjyan/aicommit"
binary="aicommit"
install_dir="${AICOMMIT_INSTALL_DIR:-$HOME/.local/bin}"

detect_platform() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin) os="darwin" ;;
    linux)  os="linux" ;;
    *) echo "Unsupported OS: $os"; exit 1 ;;
  esac
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "Unsupported architecture: $arch"; exit 1 ;;
  esac
}

prepare_install_dir() {
  if ! mkdir -p "$install_dir"; then
    echo "Could not create installation directory: $install_dir" >&2
    exit 1
  fi
  if [ ! -w "$install_dir" ]; then
    echo "Installation directory is not writable: $install_dir" >&2
    echo "Choose a writable user directory with AICOMMIT_INSTALL_DIR and run the installer again." >&2
    exit 1
  fi
}

path_contains_install_dir() {
  old_ifs=$IFS
  IFS=:
  for path_entry in $PATH; do
    if [ "${path_entry%/}" = "${install_dir%/}" ]; then
      IFS=$old_ifs
      return 0
    fi
  done
  IFS=$old_ifs
  return 1
}

shell_config_file() {
  case "${SHELL:-}" in
    */zsh) printf '%s\n' "$HOME/.zshrc" ;;
    */bash)
      if [ "$os" = "darwin" ]; then
        printf '%s\n' "$HOME/.bash_profile"
      else
        printf '%s\n' "$HOME/.bashrc"
      fi
      ;;
    *) printf '%s\n' "$HOME/.profile" ;;
  esac
}

print_path_hint() {
  if path_contains_install_dir; then
    return
  fi

  config_file="$(shell_config_file)"
  echo ""
  echo "${install_dir} is not in your PATH."
  echo "Add this line to ${config_file}, then open a new terminal:"
  echo "  export PATH=\"${install_dir}:\$PATH\""
}

get_latest_version() {
  version=$(curl -fsSL -H "User-Agent: ${repo}-installer" \
    "https://api.github.com/repos/${repo}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  if [ -z "$version" ]; then
    echo "Failed to fetch latest version" >&2
    exit 1
  fi
}

main() {
  detect_platform
  prepare_install_dir
  get_latest_version

  filename="${binary}-${os}-${arch}"
  url="https://github.com/${repo}/releases/download/v${version}/${filename}"

  echo "Installing aicommit v${version}..."
  echo "  Platform: ${os}/${arch}"
  echo "  Install to: ${install_dir}/${binary}"

  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' 0 HUP INT TERM

  curl -fsSL "$url" -o "${tmpdir}/${binary}"
  chmod +x "${tmpdir}/${binary}"

  if ! "${tmpdir}/${binary}" version >/dev/null 2>&1; then
    echo "Downloaded binary is invalid. Aborting." >&2
    exit 1
  fi

  mv "${tmpdir}/${binary}" "${install_dir}/${binary}"

  echo "Installed: $("${install_dir}/${binary}" version)"
  print_path_hint
}

main
