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

## Make some change for example, file.go

## Add files with git
git add file.go

## Commit with dx on feature branch
dx commit -m "message"

## Sync change from feature into dev branch
dx sync dev

## Push change into origin/dev branch
git push origin dev
```

### Auto Resolve confict
- go.sum file conflict
```bash
## Try to rebase feature above main branch
git rebase origin/main

## Then got a go.mod/go.sum conflict
## - Resolve go.mod conflict first
## - Run this command to resolve go.sum conflict
dx resolve-conflict
```

## Next features improvement
- Auto resolve yarn.lock conflict
- Sync mirror file to another repo
  - Example: some protobuf files 
  - TBD
