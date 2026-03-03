# LiveJournal XML-RPC API Reference

This document summarizes the LiveJournal XML-RPC API used by the blg client.

## Endpoint

All XML-RPC calls go to `/interface/xmlrpc` on the service host.

- **LiveJournal**: `http://www.livejournal.com/interface/xmlrpc`
- **Dreamwidth**: `https://www.dreamwidth.org/interface/xmlrpc`

## Authentication

LiveJournal uses challenge-response authentication:

1. Call `LJ.XMLRPC.getchallenge` to get a challenge string
2. Compute response: `MD5(challenge + MD5(password))`
3. Include in requests:
   - `username`: user login
   - `auth_method`: "challenge"
   - `auth_challenge`: the challenge string
   - `auth_response`: the computed response

### getchallenge

Returns:
- `challenge`: opaque string to hash with password
- `server_time`: server timestamp (Unix epoch)
- `expire_time`: when challenge expires

## Core Methods

### LJ.XMLRPC.postevent

Create a new journal entry.

**Required fields:**
- `username`, auth fields
- `event`: post body (HTML or plain text)
- `subject`: post title (max 255 chars)
- `lineendings`: "unix", "pc", or "mac"
- `year`, `mon`, `day`, `hour`, `min`: post timestamp

**Optional fields:**
- `security`: "public" (default), "private", or "usemask"
- `allowmask`: 32-bit bitmask for friend groups (when security=usemask)
- `props`: metadata properties (see below)
- `usejournal`: post to community instead of user journal

**Returns:**
- `itemid`: unique post ID
- `anum`: authentication number
- `url`: permanent link to post

### LJ.XMLRPC.editevent

Edit or delete an existing entry.

**Required fields:**
- `itemid`: the post to edit
- `event`: new content (empty string = delete)
- `subject`: new subject
- `year`, `mon`, `day`, `hour`, `min`: timestamp

Same optional fields as postevent.

**Returns:**
- `itemid`, `anum`, `url`

### LJ.XMLRPC.getevents

Download journal entries.

**Key fields:**
- `selecttype`: "day", "lastn", "one", or "syncitems"
- `lastsync`: for syncitems, date in "yyyy-mm-dd hh:mm:ss" format
- `itemid`: for selecttype=one
- `howmany`: for selecttype=lastn (max 50)

**Returns array of events with:**
- `itemid`: unique ID
- `eventtime`: post timestamp
- `event`: post body
- `subject`: post title
- `security`: visibility level
- `allowmask`: friend group mask
- `url`: permanent link
- `props`: metadata

### LJ.XMLRPC.syncitems

Get list of changed items since a timestamp.

**Fields:**
- `lastsync`: optional, "yyyy-mm-dd hh:mm:ss" format

**Returns:**
- `syncitems`: array of `{item, action, time}`
  - `item`: "L-123" format (L=log entry, number=itemid)
  - `action`: "create" or "update"
  - `time`: server timestamp
- `count`: items in response
- `total`: total items available

## Post Properties (props)

Common metadata fields:
- `taglist`: comma-separated tags
- `current_music`: current music
- `current_mood`: current mood
- `current_location`: current location

## Security Mapping

| Local value | LJ security | allowmask |
|-------------|-------------|-----------|
| public      | public      | -         |
| private     | private     | -         |
| friends     | usemask     | 1         |
| (group N)   | usemask     | 2^N       |

## Sync Algorithm

From cl-journal reference implementation:

1. Call `syncitems` with timestamp of last synced item (or nothing for first sync)
2. Filter to "L-" items (log entries) not yet downloaded or with newer server timestamp
3. Repeat until we have ~100 items or no more available
4. Call `getevents` with `selecttype=multiple` and `itemids=comma-separated-list`
5. Store sync timestamp with each downloaded entry
6. Repeat from step 1 until no new items

## Testing API Connectivity

Before running the validation script, verify the API is accessible:

```bash
curl -s -X POST https://www.livejournal.com/interface/xmlrpc \
  -H "Content-Type: text/xml" \
  -d '<?xml version="1.0"?><methodCall><methodName>LJ.XMLRPC.getchallenge</methodName><params><param><value><struct></struct></value></param></params></methodCall>'
```

A successful response contains a `challenge` string.

## Important Notes

1. **Protocol version**: Set `ver=1` for Unicode support
2. **Line endings**: LJ converts newlines to `<BR>` when displaying
3. **Images**: LJ doesn't have native image hosting - use external services
4. **Rate limiting**: API allows ~1 request per second. The client has built-in rate limiting and retry with exponential backoff.
5. **User-Agent header**: Required. LiveJournal rejects requests without a User-Agent header with "connection reset by peer". The client sends `blg/1.0 (https://github.com/can3p/blg)`.
6. **Unicode/Base64 encoding**: When `ver=1` is set, LiveJournal returns Unicode content (like Cyrillic text) as base64-encoded strings using the `<base64>` XML-RPC type instead of `<string>`. The client automatically decodes these values.
