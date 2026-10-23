---
description: Use after backend or frontend edits to confirm the Go code vets and compiles and the web app type-checks, lints and passes its unit tests
---
Run `go vet ./...`, `go build -o /dev/null ./...` and `make web-check`.
Only output errors if a step fails. If all succeed, return strictly: "BUILD OK"
