#!/bin/sh
# Creates a Go project from go-template: devbox run new <module-path> <dir>
# Devbox runs scripts from this repo's root, so <dir> must be absolute.
set -eu

if [ $# -ne 2 ]; then
	echo "usage: devbox run new <module-path> <absolute-dir>" >&2
	exit 2
fi
module=$1
dir=$2
case $dir in
/*) ;;
*)
	echo "new: $dir must be an absolute path" >&2
	exit 2
	;;
esac
name=$(basename "$module")

go run golang.org/x/tools/cmd/gonew@v0.52.0 github.com/ianflanagan1/go-template "$module" "$dir"
cd "$dir"
mv cmd/app "cmd/$name"
sed -i "s/^\/\/ Command app /\/\/ Command $name /; s/\"app:\"/\"$name:\"/; s/usage: app /usage: $name /" "cmd/$name/main.go"
rm LICENSE
printf '# %s\n' "$name" > README.md
go tool go-baseline write
devbox run check
git init --quiet --initial-branch=main
git add --all
git commit --quiet --message "Initialise from go-template"
echo "Created $module in $dir"
