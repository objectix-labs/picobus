#!/usr/bin/env bash
set -euo pipefail

echo "Running post-create steps for Go devcontainer..."

# Ensure Go bin is on PATH for the vscode user
export GOPATH=${GOPATH:-/go}
export PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

# Encsure user can access Go modules cache
sudo chown -R vscode:vscode "$GOPATH"

echo "Go version: $(go version)"
echo "GOMODCACHE: $(go env GOMODCACHE)"

echo "Checking installed tools..."
for tool in gopls dlv staticcheck goimports; do
  if ! command -v $tool >/dev/null 2>&1; then
    echo "Installing $tool"
    case $tool in
      gopls) go install golang.org/x/tools/gopls@latest ;;
      dlv) go install github.com/go-delve/delve/cmd/dlv@latest ;;
      staticcheck) go install honnef.co/go/tools/cmd/staticcheck@latest ;;
      goimports) go install golang.org/x/tools/cmd/goimports@latest ;;
    esac
  else
    echo "$tool already present"
  fi
done

echo "Post-create finished."

