# scan-projects

Small cross-platform CLI to scan folders for projects (with or without Git) and output a JSON inventory.

It detects:
- Whether a folder is a "project" (Git repo or marker files)
- Language stacks (python, node, dotnet, kotlin, swift)
- Git metadata (remotes, branches, dirty state, etc.)

## Setup

```bash
go mod tidy
````

## Build (native)

```bash
go build -o scan-projects ./cmd/scan-projects
```

## Build for all platforms

```bash
./build.sh
```

Binaries will appear in `dist/`.

## Usage

```bash
./scan-projects \
  --root ~/dev \
  --root /Volumes/ext-ssd/dev-archive \
  --output machines/laptop.json
```

Example JSON entry:

```json
{
  "name": "carrio-api",
  "path": "/Users/arman/dev/github/org-carrio/carrio-api",
  "stacks": ["dotnet", "python"],
  "primary_stack": "dotnet",
  "is_git": true,
  "has_origin": true,
  "remotes": [
    { "name": "origin", "url": "git@github.com:carrio-org/carrio-api.git" }
  ],
  "host": "github.com",
  "owner": "carrio-org",
  "repo_slug": "carrio-api",
  "current_branch": "main",
  "local_branches": ["main", "develop"],
  "tracking_branch": "origin/main",
  "dirty": false
}
```

````

---

## 11. How you’ll actually use this

On your **dev machine**:

```bash
cd scan-projects
go mod tidy
./build.sh
````

Then copy the right binary to each machine / SSD:

* Windows desktop: `dist/scan-projects-windows-amd64.exe`
* MacBook: `dist/scan-projects-macos-arm64`
* Linux VM: `dist/scan-projects-linux-amd64`

Run on any machine, example:

```bash
./scan-projects-macos-arm64 \
  --root ~/dev \
  --root /Volumes/SSD1/dev-archive \
  --output machines/macos.json