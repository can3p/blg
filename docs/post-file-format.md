# Post File Format

This document describes the markdown file format used by blg for blog posts.

## Basic Structure

Posts are markdown files with YAML-like headers at the top:

```markdown
title: Post Title
privacy: public
tags: tag1, tag2

Post body in markdown...
```

## Header Fields

### Common Fields (all services)

| Field | Description | Values |
|-------|-------------|--------|
| `title` / `subject` | Post title | Any string (max 255 chars for LJ) |

### pcom Service Fields

| Field | Required | Values |
|-------|----------|--------|
| `subject` | Yes | Post title |
| `visibility` | Yes | `direct_only`, `second_degree` |
| `published` | Yes | `yes`, `no` |

### LiveJournal/Dreamwidth Fields

| Field | Required | Values |
|-------|----------|--------|
| `title` | No | Post title |
| `privacy` | No | `public`, `private`, `friends`, or group number |
| `tags` | No | Comma-separated tag list |
| `music` | No | Current music |
| `mood` | No | Current mood |
| `location` | No | Current location |
| `journal` | No | Community name (to post to community) |
| `draft` | No | If present, skip this file during push |

## Privacy Mapping

| File value | LJ security | allowmask |
|------------|-------------|-----------|
| `public` | public | - |
| `private` | private | - |
| `friends` | usemask | 1 (all friends) |
| `N` (number) | usemask | N (specific group mask) |

## Body Content

The body starts after the first blank line following headers.

### Markdown Support

Standard markdown is supported:
- Headers, paragraphs, lists
- Links: `[text](url)`
- Images: `![alt](path)`
- Code blocks, emphasis, etc.

### Local File References

Posts can reference other local posts by filename:

```markdown
See my [previous post](2024-01-15-other-post.md) for details.
```

When pushing, these are resolved to actual remote URLs.

### YouTube Embeds (LiveJournal/Dreamwidth)

YouTube videos can be embedded by simply including the URL:

**Standalone embed** - URL on its own line becomes an embed:
```markdown
Check out this video:

https://www.youtube.com/watch?v=dQw4w9WgXcQ

What do you think?
```

**Inline with text** - URL in text gets an embed appended after the paragraph:
```markdown
I found this great video https://www.youtube.com/watch?v=dQw4w9WgXcQ today!
```

Supported URL formats:
- `https://www.youtube.com/watch?v=VIDEO_ID`
- `https://youtu.be/VIDEO_ID`
- `https://www.youtube.com/embed/VIDEO_ID`

When pushing, YouTube URLs are converted to `<lj-embed>` tags. When fetching, embeds are converted back to URLs.

### User References (LiveJournal/Dreamwidth)

Reference other users with `@username`:

```markdown
Hey @alice, what do you think about this?

I was chatting with @bob_smith yesterday.
```

When pushing:
- For LiveJournal: `@username` → `<a href="https://username.livejournal.com/">@username</a>`
- For Dreamwidth: `@username` → `<a href="https://username.dreamwidth.org/">@username</a>`

When fetching, user links are converted back to `@username` format.

**Username rules:**
- Must start with a letter
- Can contain lowercase letters, numbers, and underscores
- Minimum 3 characters
- Email addresses (user@domain.com) are NOT matched

### Images

Images are referenced by local path:

```markdown
![Photo](images/photo.jpg)
```

When pushing:
1. Images are uploaded to the service
2. Local paths are replaced with remote IDs/URLs
3. Image mappings are stored in `posts.json`

## File Naming Convention

Recommended format: `YYYY-MM-DD-slug.md`

Example: `2024-05-15-my-first-post.md`

When fetching posts, filenames are generated from:
1. Post date
2. Slugified title
3. Counter suffix if needed (e.g., `-2.md`)

## Example Files

### pcom Post

```markdown
subject: My New Post
visibility: direct_only
published: yes

This is my post content with **markdown**.

![Photo](photo.jpg)
```

### LiveJournal Post

```markdown
title: Thoughts on Programming
privacy: friends
tags: coding, thoughts
music: Pink Floyd - Time
mood: contemplative

Today I've been thinking about code quality...

See also [my previous post](2024-01-10-code-review.md).
```

### Draft Post (skipped during push)

```markdown
title: Work in Progress
privacy: private
draft: yes

This post won't be pushed until draft is removed.
```
