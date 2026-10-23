---
description: Prompts code review via Opus subagent after multi-file or complex changes
---
Check the current git status and diff. 

IF the changes touch critical backend logic (e.g., path confinement in `internal/project`, API handlers in `internal/server`, frontmatter parsing, file writes, or >3 files changed):
1. Ask the user: "I notice significant changes to core systems. Should I launch an isolated Opus code review subagent? (y/n)"
2. If the user says yes, spawn the `code-reviewer` subagent.
3. If the user says no, skip review and summarize changes concisely.

IF the change is small or purely visual (styling, copy, layout tweaks):
Do not ask. Skip review and return "Done".
