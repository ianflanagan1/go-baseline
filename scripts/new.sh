#!/bin/sh
# Creates a Go project from go-template:
#   devbox run new command|pure-library|adapters <module-path> <absolute-dir>
# Devbox runs scripts from this repo's root, so <dir> must be absolute.
set -eu

usage="usage: devbox run new command|pure-library|adapters <module-path> <absolute-dir>"
if [ $# -ne 3 ]; then
	echo "$usage" >&2
	exit 2
fi
kind=$1
module=$2
dir=$3
case $kind in
command | pure-library | adapters) ;;
*)
	echo "$usage" >&2
	exit 2
	;;
esac
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
case $kind in
command)
	mv cmd/app "cmd/$name"
	main="cmd/$name/main.go"
	# -i.bak works with both GNU and BSD (macOS) sed.
	sed -i.bak "s/^\/\/ Command app /\/\/ Command $name /; s/\"app:\"/\"$name:\"/" "$main"
	rm "$main.bak"
	if ! grep -q "^// Command $name " "$main" || ! grep -q "\"$name:\"" "$main"; then
		echo "new: couldn't rename the command in $main; go-template's main.go has changed" >&2
		exit 1
	fi
	;;
pure-library | adapters)
	rm -r cmd
	# The package is named for the module, without a go- prefix, hyphens or dots.
	pkg=$(printf '%s' "${name#go-}" | tr -d -- '-.' | tr '[:upper:]' '[:lower:]')
	case $pkg in
	'' | [0-9]* | *[!a-z0-9_]*)
		echo "new: can't make a package name from $name" >&2
		exit 1
		;;
	esac
	printf '// Package %s TODO: say what it provides.\npackage %s\n' "$pkg" "$pkg" > doc.go
	;;
esac
rm LICENSE
case $kind in
command)
	cat > README.md <<EOF
# $name

TODO: what it does, and for whom.

## Running

\`\`\`sh
devbox run build
./bin/$name
\`\`\`

## Configuration

Read only in \`cmd/$name\`.

| Environment variable or flag | Meaning | Default |
|---|---|---|

## Architecture

\`go tool go-baseline map\` lists each package's role, the ports the shell declares and the adapters that implement them. The rules are in \`BASELINE.md\`.
EOF
	;;
*)
	cat > README.md <<EOF
# $name

TODO: what it provides, and for whom.

## Usage

\`\`\`go
import "$module"
\`\`\`

## Architecture

This is a \`$kind\` module. \`go tool go-baseline map\` lists each package's role. The rules are in \`BASELINE.md\`.
EOF
	;;
esac
go tool go-baseline write -kind "$kind"
devbox run check
git init --quiet --initial-branch=main
git add --all
git commit --quiet --message "Initialise from go-template"
echo "Created $module in $dir"
