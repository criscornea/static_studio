---
name: build-check
description: Use after code edits to confirm the project vets, builds, lints and passes tests. Auto-detects Go, Node (Vue/React/TS) and PHP (Symfony) projects. Prints only errors or BUILD OK.
---
Run `.claude/skills/build-check/check.sh`.

- `BUILD OK`: done, say nothing more about it.
- Failure: the script prints the failing command and the last 40 lines. Fix the cause, then run it again.
- `NO CHECKS`: no stack detected. Write `scripts/check.sh` for this project; the script then uses it.
