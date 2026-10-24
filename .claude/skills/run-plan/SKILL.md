---
name: run-plan
description: Execute PLAN.md one phase at a time - Sonnet implements, optional Opus review, then stop so the user can test before the local squash merge into main. `/run-plan auto` runs all phases without stopping. Use for /run-plan or "start building", "continue the plan".
---
You are the coordinator. Stay lean: never read source files or diffs yourself. You only read `PLAN.md`, run git commands, and delegate.

Modes:
- **Manual (default)**: `/run-plan`. Run one phase up to `Status: ready`, then stop so the user can test the branch. On the next `/run-plan`, merge the ready phase and continue with the next phase, stopping again at `ready`.
- **Auto**: only when the user says `/run-plan auto` (or explicitly asks for autopilot). Run every phase without stopping for tests: treat `ready` as tested and merge right away.

Setup:
1. No `PLAN.md`? Stop and tell the user to run `/kickoff`.
2. `git switch main`. Auto mode only: set `autopilot: on` in PLAN.md and commit `chore: start autopilot`. Manual mode: leave `autopilot: off`.

Loop over phases in order, skipping `Status: merged`:

1. **Implement** (Status `todo` or `in-progress`): spawn the `implementer` agent with prompt `Implement Phase <n> of PLAN.md.`
   - Report `blocked`: stop the loop and show the user the BLOCKED line.
2. **Review gate** (Status `review`): run `git diff --shortstat main...<branch>`.
   - Ask the user (AskUserQuestion, one question: "Review Phase <n> with Opus before merge?") only if `Risk: high`.
   - Any other risk (low, medium) is an automatic pass: skip the review without asking, whatever the file or line count.
3. **Review** (user said yes): spawn `code-reviewer` with prompt `Review branch <branch> for Phase <n>.`
   - `NO ISSUES FOUND`: go to step 4.
   - Findings: spawn `implementer` with `On branch <branch>, fix these review findings, one commit, then run build-check:` plus the findings. One fix round only. Show remaining critical findings to the user.
4. **Ready**: on the branch, set the phase `Status: ready` in PLAN.md and commit `chore: phase <n> ready for testing`.
   - Manual mode: stop. Tell the user the branch name, what to test (from the phase's `Done when`), and to run `/run-plan` again to merge, or to describe problems to fix first.
   - Auto mode: go to merge.
5. **Merge** (Status `ready`), in this order:
   ```
   git switch main
   git merge --squash <branch>
   ```
   Set the phase `Status: merged` in PLAN.md, then:
   ```
   git add -A
   git commit -m "feat: phase <n> <name>" -m "<milestone list>"
   git branch -D <branch>
   ```
   Do not push `main` here: every push to `main` runs CI and can mail the user. Merged phases stay local until the user asks for a push, or Finish.
   If the branch exists on the remote, `git push origin --delete <branch>` (branch pushes do not run CI).
   Merge conflict: write `BLOCKED: merge conflict phase <n>` and stop.

Push `main` only when the user says so (after testing the merged phases manually), or at Finish:
1. Run build-check on `main`. It must print `BUILD OK`; otherwise fix first (spawn `implementer`) and do not push.
2. Ask the user (AskUserQuestion): "Push main to origin (<k> unpushed commits)?" Push only on yes.

Finish: when every phase is merged, set `autopilot: off` (if on), commit `chore: plan complete`, then follow the push steps above.

Report to the user only when stopping (ready, blocked or finished): one line per phase handled (`P<n> ready|merged|blocked, reviewed yes|no`).
