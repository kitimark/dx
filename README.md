# DX
Improve development experience that working in my cooperation development

## Installation
```bash
go install github.com/kitimark/dx/cmd/dx@latest
```

## Features

### Help to deploy dev/beta branch
```bash
## Create new feature branch
git checkout -b feature

## Make some change for example; file.go

## Add files with git
git add file.go

## Commit with dx on feature branch
dx commit -m "message"

## Sync change from feature into dev branch
dx sync dev

## Push change into origin/dev branch
git push origin dev
```

### Auto Resolve conflict

File types is supported to auto resolve conflict
- go.sum
- yarn.lock

```bash
## Try to rebase feature above main branch and then got code conflict
## Or another actions that can got a code conflict
git rebase origin/main

## Then got a conflict that 
## - go.sum is in these conflict
##   - Resolve go.mod and *.go code first
##   - Run dx resolve-conflict
## - yarn.lock is in these conflict
##   - Resolve package.json files first
##   - Run dx resolve-conflict
dx resolve-conflict
```

## Next features improvement
- Sync mirror file to another repo
  - Example: some protobuf files 
  - TBD
