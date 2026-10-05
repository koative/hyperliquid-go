#!/bin/sh
# Generates the API reference page from the Go doc comments into the Hugo
# content tree (docs/content/docs/reference, git-ignored).
set -eu
cd "$(dirname "$0")/.."
out=docs/content/docs/reference
rm -rf "$out"
go run go.abhg.dev/doc2go@v0.12.2 \
	-embed -basename _index.html \
	-home github.com/koative/hyperliquid-go \
	-highlight classes:github \
	-frontmatter docs/frontmatter.tmpl \
	-out "$out" .
# Hugo parses "{{<" and "{{%" as shortcodes, which Go code can contain.
sed -i.bak 's/{{/\&#123;\&#123;/g' "$out/_index.html"
rm "$out/_index.html.bak"
