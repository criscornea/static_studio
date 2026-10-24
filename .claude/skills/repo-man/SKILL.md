---
name: repo-man
description: Use before exploring unfamiliar code or when unsure which files hold a symbol; prints Aider's tree-sitter repo map (files, types, function signatures) so you open only the files you need
---
Run `.claude/skills/repo-man/repo-map.sh` (default budget 2048 tokens).
For a large repo, raise the budget: `.claude/skills/repo-man/repo-map.sh 4096`.

Use the map to pick files and symbols. Then open only those files, by line range where possible.
Do not run it again in the same task unless files were added.
