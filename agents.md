# BLG - Blogging Client Architecture

## Overview

`blg` is a command-line blogging client written in Go that supports multiple blogging services. It manages local markdown files and syncs them with remote services.

## Project Structure

```
blg/
├── cmd/                    # CLI commands (cobra)
│   ├── root.go            # Root command setup
│   ├── init.go            # Initialize a blog folder
│   ├── new.go             # Create new post
│   ├── push.go            # Push local changes to remote
│   ├── fetch.go           # Fetch posts from remote
│   ├── merge.go           # Merge fetched posts to local files
│   ├── status.go          # Show sync status
│   ├── last.go            # Show last post
│   └── url.go             # Get URL for a post
├── pkg/
│   ├── fileformat/        # Post file parsing
│   │   └── parse.go       # Parse markdown with headers
│   ├── ops/               # Business logic operations
│   │   ├── push.go        # Push operation (create/update/delete)
│   │   ├── fetch.go       # Fetch remote posts
│   │   ├── merge.go       # Merge remote to local
│   │   ├── status.go      # Calculate folder state
│   │   └── ...
│   ├── services/          # Service implementations
│   │   ├── pcom/          # pcom.com service (reference impl)
│   │   │   ├── pcom.go    # API client
│   │   │   └── markdown.go # Markdown parsing with goldmark
│   │   └── livejournal/   # LiveJournal/Dreamwidth
│   │       ├── livejournal.go  # Service implementation
│   │       ├── xmlrpc.go       # XML-RPC client
│   │       └── markdown.go     # Markdown parsing
│   ├── store/             # Local storage management
│   │   ├── store.go       # Config loading/saving (posts.json)
│   │   ├── remote.go      # Remote posts cache (remote.json)
│   │   └── examine.go     # Folder state examination
│   ├── types/             # Core types and interfaces
│   │   ├── service.go     # Service interface definition
│   │   ├── post.go        # Post types
│   │   ├── remote_posts.go # Remote post types
│   │   ├── storage.go     # Config types
│   │   └── errors.go
│   └── util/
│       └── pwd/           # Password management (keyring)
└── main.go
```

## Key Types

### Service Interface (`pkg/types/service.go`)

```go
type Service interface {
    PreparePost(headers map[string]string, body string) (*Post, []string, error)
    UploadImage(string) (string, error)
    Create(p *Post) (string, error)
    Update(remoteID string, p *Post) error
    Delete(remoteID string) error
    PostURL(remoteID string) string
    NewPostTemplate(name string) string
    FetchPosts(updatedSince int64) ([]*RemotePost, []string, error)
    DownloadImage(string) ([]byte, error)
    FormatRemotePost(*RemotePost) (string, []byte, error)
}
```

### Post (`pkg/types/post.go`)

```go
type Post struct {
    Headers PostHeaders  // map[string]any - service-specific fields
    Body    PostBody     // Interface with ReplaceImages and MaybeString
}
```

### Config (`pkg/types/storage.go`)

Stored in `posts.json`:
- `login` - username
- `service_name` - service identifier
- `custom_host` - optional custom host
- `remote_posts` - list of synced posts with file->remote mapping
- `remote_images` - list of synced images

## File Format

Posts are markdown files with YAML-like headers:

```markdown
subject: Post Title
visibility: direct_only
published: yes

Post body in markdown...
```

Headers are parsed by `pkg/fileformat/parse.go` - simple `key: value` format until first non-header line.

## Service Registration

Services register themselves in `init()`:

```go
func init() {
    types.DefaultServiceRepo.Register(
        types.NewServiceDefinition(
            "livejournal",
            "www.livejournal.com",
            createClient))
}
```

## LiveJournal API

Uses XML-RPC at `/interface/xmlrpc`. Key methods:
- `LJ.XMLRPC.getchallenge` - Get auth challenge
- `LJ.XMLRPC.postevent` - Create post
- `LJ.XMLRPC.editevent` - Update/delete post
- `LJ.XMLRPC.getevents` - Fetch posts
- `LJ.XMLRPC.syncitems` - Get list of changed items

Authentication: Challenge-response with MD5(challenge + MD5(password))

## LiveJournal Post Fields

From cl-journal reference:
- `title` - Post subject
- `privacy` - public/private/friends (maps to security + allowmask)
- `tags` - Comma-separated tags (prop: taglist)
- `music` - Current music (prop: current_music)
- `mood` - Current mood (prop: current_mood)
- `location` - Current location (prop: current_location)
- `journal` - Post to community instead of user journal (usejournal)
- `draft` - If present, skip this file

## Migration Considerations

For migrating between LJ-like services:
1. Posts reference each other by local filename, not remote URL
2. When pushing, resolve `[link](other-post.md)` to actual remote URLs
3. When fetching, convert remote URLs back to local filenames
4. Images need to be downloaded and re-uploaded to new service
5. LJ doesn't have native image hosting - uses external services or fotki

## Current Limitations

1. **Image hosting**: LiveJournal doesn't have native image uploads. Images must be hosted externally and referenced by URL.

2. **Link resolution timing**: Links to other posts are only resolved if the target post has already been pushed. Push posts in dependency order for correct link resolution.

## Development Commands

Use the Makefile for common development tasks:

```bash
make test    # Run all tests with race detection and coverage
make lint    # Run golangci-lint
make build   # Build all packages
make check   # Run all checks (build, test, lint)
make fix     # Run go fix and go mod tidy
```

CI runs `make fix` first and fails if it produces uncommitted changes.

## Reference Implementation

See `/Users/dima/code/cl-journal` for Common Lisp implementation:
- `src/lj-api.lisp` - XML-RPC calls
- `src/db.lisp` - Data structures and serialization
- `src/file-api.lisp` - File parsing and formatting
- `src/markdown.lisp` - Link resolution (file.md -> post URL)
