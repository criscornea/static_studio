#!/usr/bin/env bash
# Create a small Hugo site for manually testing Static Studio.
# Usage: scripts/sample-site.sh [dir] [--force]   (default dir: ./sample-site)
# Needs no hugo binary; the site has its own minimal layouts, so no theme is required.
set -euo pipefail

dir=sample-site
force=false
for arg in "$@"; do
  case $arg in
    --force) force=true ;;
    -h|--help) sed -n '2,4p' "$0" | sed 's/^# //'; exit 0 ;;
    *) dir=$arg ;;
  esac
done

if [[ -e $dir ]]; then
  if ! $force; then
    echo "$dir already exists; pass --force to recreate it" >&2
    exit 1
  fi
  rm -rf -- "$dir"
fi

mkdir -p "$dir"/{archetypes,content/posts/2026,content/docs,layouts/_default,static/images}
cd "$dir"

cat > hugo.toml <<'EOF'
baseURL = "https://example.github.io/sample-site/"
languageCode = "en-us"
title = "Static Studio Sample"
EOF

cat > archetypes/default.md <<'EOF'
+++
title = '{{ replace .File.ContentBaseName "-" " " | title }}'
date = {{ .Date }}
draft = true
+++
EOF

cat > layouts/_default/baseof.html <<'EOF'
<!doctype html>
<html lang="{{ .Site.LanguageCode }}">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }} | {{ .Site.Title }}</title>
  <style>body{font-family:system-ui,sans-serif;max-width:42rem;margin:2rem auto;padding:0 1rem;line-height:1.6}img{max-width:100%}</style>
</head>
<body>
  <nav><a href="{{ "/" | relURL }}">{{ .Site.Title }}</a></nav>
  <main>{{ block "main" . }}{{ end }}</main>
</body>
</html>
EOF

cat > layouts/_default/single.html <<'EOF'
{{ define "main" }}
<article>
  <h1>{{ .Title }}</h1>
  {{ with .Date }}<p><time>{{ .Format "2006-01-02" }}</time>{{ if $.Draft }} · draft{{ end }}</p>{{ end }}
  {{ .Content }}
</article>
{{ end }}
EOF

cat > layouts/_default/list.html <<'EOF'
{{ define "main" }}
<h1>{{ .Title }}</h1>
{{ .Content }}
<ul>
  {{ range .Pages }}<li><a href="{{ .RelPermalink }}">{{ .Title }}</a></li>{{ end }}
</ul>
{{ end }}
EOF

cat > content/_index.md <<'EOF'
---
title: Home
---
Welcome to the sample site. Browse the [posts](posts/) or the [docs](docs/).
EOF

cat > content/posts/_index.md <<'EOF'
---
title: Posts
---
EOF

cat > content/posts/hello-world.md <<'EOF'
---
title: Hello World
date: 2026-10-01T09:00:00+02:00
draft: false
tags: [intro, sample]
author: Sample Author
---
This is a **YAML** frontmatter page with tags and an unknown `author` key.

![Sample image](/images/sample.png)
EOF

cat > content/posts/toml-page.md <<'EOF'
+++
# a TOML comment
title = "TOML Frontmatter"
date = 2026-10-05T12:00:00+02:00
draft = false
weight = 10
+++
This page uses **TOML** frontmatter and has a `weight` key.
EOF

cat > content/posts/work-in-progress.md <<'EOF'
---
title: Work in Progress
date: 2026-10-20T10:00:00+02:00
draft: true
---
A draft. Hugo only renders it with `--buildDrafts`.
EOF

cat > content/posts/future-post.md <<'EOF'
---
title: Future Post
date: 2030-01-01T00:00:00+01:00
draft: false
---
Dated in the future. Hugo only renders it with `--buildFuture`.
EOF

cat > "content/posts/Mixed Case Name.md" <<'EOF'
---
title: Mixed Case Name
date: 2026-10-10T08:00:00+02:00
---
The file name has capitals and spaces; Hugo serves it at `/posts/mixed-case-name/`.
EOF

printf -- '---\r\ntitle: Windows Line Endings\r\ndate: 2026-10-12T08:00:00+02:00\r\n---\r\nThis page uses CRLF line endings.\r\nSaving it must keep them.\r\n' \
  > content/posts/crlf-page.md

cat > content/posts/2026/nested-post.md <<'EOF'
---
title: Nested Post
date: 2026-09-15T08:00:00+02:00
---
A page one folder deeper.
EOF

cat > content/posts/no-frontmatter.md <<'EOF'
A page without any frontmatter. Saving fields from the form adds a YAML block.
EOF

cat > content/docs/_index.md <<'EOF'
---
title: Docs
---
EOF

cat > content/docs/getting-started.md <<'EOF'
---
title: Getting Started
date: 2026-08-01T08:00:00+02:00
---
## Install

Some text, a list and code:

- one
- two

```sh
hugo server
```
EOF

# 1x1 transparent PNG
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' \
  > static/images/sample.png

echo "Sample Hugo site created at $(pwd)"
command -v hugo >/dev/null || echo "Note: hugo is not installed; build and preview need it."
