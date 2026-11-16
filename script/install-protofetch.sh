#!/bin/bash
# Installs protofetch, which capabilities fetch their protos with: built with cargo, or taken from its releases
# on machines without a Rust toolchain to build it with.
set -x

VERSION=$1

if [ "$VERSION" == "" ]; then
    echo "version required"
    exit 1
fi

if protofetch --version | grep -qx "protofetch $VERSION"; then
    echo "protofetch up-to-date @ $VERSION"
    exit 0
fi

if command -v cargo >/dev/null 2>&1; then
    cargo install protofetch --version $VERSION --locked
    exit 0
fi

os=$(uname)
arch=$(uname -m)

case "$os-$arch" in
    Darwin-arm64) target=aarch64-apple-darwin ;;
    Darwin-x86_64) target=x86_64-apple-darwin ;;
    Linux-x86_64) target=x86_64-unknown-linux-musl ;;
    Linux-aarch64) target=aarch64-unknown-linux-musl ;;
    *)
        echo "unsupported os $os-$arch, and no cargo to build protofetch with. update $0"
        exit 1
        ;;
esac

install_dir=$HOME/.local
mkdir -p $install_dir

workdir=$(mktemp -d)
pushd $workdir
curl -LO https://github.com/coralogix/protofetch/releases/download/v$VERSION/protofetch_$target.tar.gz
tar -xzf protofetch_$target.tar.gz -C $install_dir
popd

echo "protofetch $VERSION installed in $install_dir/bin"
echo "Add $install_dir/bin to PATH"
