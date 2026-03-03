package livejournal

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/can3p/blg/pkg/types"
	"github.com/can3p/blg/pkg/util/pwd"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
)

const endpoint = "/interface/xmlrpc"

var PrivacyValues = []string{"public", "private", "friends"}

type client struct {
	cfg      types.Config
	rpc      *xmlrpcClient
	username string
	password string
}

func (c *client) PreparePost(fields map[string]string, body string) (*types.Post, []string, error) {
	p := &types.Post{
		Headers: types.PostHeaders{},
	}

	// title is optional for LJ
	if title, ok := fields["title"]; ok {
		p.Headers["title"] = title
	} else {
		p.Headers["title"] = ""
	}

	// privacy defaults to public
	privacy := "public"
	if v, ok := fields["privacy"]; ok {
		privacy = v
	}
	switch privacy {
	case "public":
		p.Headers["security"] = "public"
	case "private":
		p.Headers["security"] = "private"
	case "friends":
		p.Headers["security"] = "usemask"
		p.Headers["allowmask"] = 1
	default:
		return nil, nil, errors.Errorf("invalid privacy value: %s, must be one of: %s", privacy, strings.Join(PrivacyValues, ", "))
	}

	// optional fields
	if tags, ok := fields["tags"]; ok {
		p.Headers["props"] = map[string]any{"taglist": tags}
	}

	props := map[string]any{}
	if existing, ok := p.Headers["props"].(map[string]any); ok {
		props = existing
	}

	if music, ok := fields["music"]; ok {
		props["current_music"] = music
	}
	if mood, ok := fields["mood"]; ok {
		props["current_mood"] = mood
	}
	if location, ok := fields["location"]; ok {
		props["current_location"] = location
	}

	if len(props) > 0 {
		p.Headers["props"] = props
	}

	// journal (for posting to communities)
	if journal, ok := fields["journal"]; ok {
		p.Headers["usejournal"] = journal
	}

	// check for draft
	if _, ok := fields["draft"]; ok {
		return nil, nil, errors.Errorf("post is marked as draft, skipping")
	}

	// parse body for images and links
	parser, err := parseBody(body)
	if err != nil {
		return nil, nil, err
	}

	p.Body = parser

	extractedImages, err := parser.ExtractImages()
	if err != nil {
		return nil, nil, err
	}

	return p, extractedImages, nil
}

func (c *client) UploadImage(fname string) (string, error) {
	// LiveJournal doesn't have native image hosting
	// Images are typically hosted externally (e.g., on fotki.yandex.ru or imgur)
	// For now, we return an error indicating this limitation
	return "", errors.Errorf("LiveJournal does not support native image uploads. Please host images externally and reference them by URL.")
}

func (c *client) DownloadImage(url string) ([]byte, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, errors.Errorf("invalid image URL: %s", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "downloading image from %s", url)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "reading image data from %s", url)
	}

	return data, nil
}

func (c *client) FetchPosts(updatedSince int64) ([]*types.RemotePost, []string, error) {
	// First, get sync items to find what changed
	itemIDs, err := c.getSyncItems(updatedSince)
	if err != nil {
		return nil, nil, err
	}

	if len(itemIDs) == 0 {
		return []*types.RemotePost{}, []string{}, nil
	}

	// Fetch the actual posts
	posts, imageURLs, err := c.getEvents(itemIDs)
	if err != nil {
		return nil, nil, err
	}

	return posts, imageURLs, nil
}

func (c *client) getSyncItems(updatedSince int64) ([]int, error) {
	params := map[string]any{
		"ver": 1,
	}

	if updatedSince > 0 {
		t := time.Unix(updatedSince, 0).UTC()
		params["lastsync"] = t.Format("2006-01-02 15:04:05")
	}

	params, err := c.rpc.addAuth(params, c.username, c.password)
	if err != nil {
		return nil, err
	}

	result, err := c.rpc.call("LJ.XMLRPC.syncitems", params)
	if err != nil {
		return nil, err
	}

	syncItems, ok := result["syncitems"].([]any)
	if !ok {
		return []int{}, nil
	}

	itemIDs := []int{}
	for _, item := range syncItems {
		if m, ok := item.(map[string]any); ok {
			itemStr := getString(m, "item")
			// Format is "L-123" for log entries
			if strings.HasPrefix(itemStr, "L-") {
				if id, err := strconv.Atoi(itemStr[2:]); err == nil {
					itemIDs = append(itemIDs, id)
				}
			}
		}
	}

	return itemIDs, nil
}

func (c *client) getEvents(itemIDs []int) ([]*types.RemotePost, []string, error) {
	if len(itemIDs) == 0 {
		return []*types.RemotePost{}, []string{}, nil
	}

	// Convert to comma-separated string
	idStrs := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		idStrs[i] = strconv.Itoa(id)
	}

	params := map[string]any{
		"ver":         1,
		"selecttype":  "multiple",
		"itemids":     strings.Join(idStrs, ","),
		"lineendings": "unix",
	}

	params, err := c.rpc.addAuth(params, c.username, c.password)
	if err != nil {
		return nil, nil, err
	}

	result, err := c.rpc.call("LJ.XMLRPC.getevents", params)
	if err != nil {
		return nil, nil, err
	}

	events, ok := result["events"].([]any)
	if !ok {
		return []*types.RemotePost{}, []string{}, nil
	}

	posts := []*types.RemotePost{}
	imageURLs := []string{}

	for _, event := range events {
		if m, ok := event.(map[string]any); ok {
			itemID := getInt(m, "itemid")
			eventText := getString(m, "event")
			eventTime := getString(m, "eventtime")

			// Parse event time to get updated timestamp
			var updatedAt int64
			if t, err := time.Parse("2006-01-02 15:04:05", eventTime); err == nil {
				updatedAt = t.Unix()
			}

			// Create hash from content
			b, _ := json.Marshal(m)
			hash := fmt.Sprintf("%x", sha256.Sum256(b))

			posts = append(posts, &types.RemotePost{
				ID:        strconv.Itoa(itemID),
				Hash:      hash,
				Data:      m,
				UpdatedAt: updatedAt,
			})

			// Extract images from event text
			if parser, err := parseBody(eventText); err == nil {
				if imgs, err := parser.ExtractImages(); err == nil {
					imageURLs = append(imageURLs, imgs...)
				}
			}
		}
	}

	return posts, imageURLs, nil
}

func (c *client) FormatRemotePost(remote *types.RemotePost) (string, []byte, error) {
	m, ok := remote.Data.(map[string]any)
	if !ok {
		return "", nil, errors.Errorf("invalid remote post data")
	}

	subject := getString(m, "subject")
	event := getString(m, "event")
	security := getString(m, "security")
	eventTime := getString(m, "eventtime")

	// Build header
	var sb strings.Builder

	if subject != "" {
		fmt.Fprintf(&sb, "title: %s\n", subject)
	}

	// Map security to privacy
	privacy := "public"
	switch security {
	case "private":
		privacy = "private"
	case "usemask":
		privacy = "friends"
	}
	if privacy != "public" {
		fmt.Fprintf(&sb, "privacy: %s\n", privacy)
	}

	// Extract props
	if props, ok := m["props"].(map[string]any); ok {
		if tags := getString(props, "taglist"); tags != "" {
			fmt.Fprintf(&sb, "tags: %s\n", tags)
		}
		if music := getString(props, "current_music"); music != "" {
			fmt.Fprintf(&sb, "music: %s\n", music)
		}
		if mood := getString(props, "current_mood"); mood != "" {
			fmt.Fprintf(&sb, "mood: %s\n", mood)
		}
		if location := getString(props, "current_location"); location != "" {
			fmt.Fprintf(&sb, "location: %s\n", location)
		}
	}

	sb.WriteString("\n")
	sb.WriteString(event)

	// Generate filename
	slugTitle := "no-title"
	if subject != "" {
		slugTitle = slug.Make(subject)
	}

	var t time.Time
	if parsed, err := time.Parse("2006-01-02 15:04:05", eventTime); err == nil {
		t = parsed
	} else {
		t = time.Unix(remote.UpdatedAt, 0)
	}

	fname := fmt.Sprintf("%s-%s.md", t.Format("2006-01-02"), slugTitle)

	return fname, []byte(sb.String()), nil
}

func (c *client) Create(p *types.Post) (string, error) {
	body, err := p.Body.MaybeString()
	if err != nil {
		return "", err
	}

	now := time.Now()

	params := map[string]any{
		"ver":         1,
		"event":       body,
		"subject":     p.Headers["title"],
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
	}

	if security, ok := p.Headers["security"].(string); ok {
		params["security"] = security
	}

	if allowmask, ok := p.Headers["allowmask"].(int); ok {
		params["allowmask"] = allowmask
	}

	if props, ok := p.Headers["props"].(map[string]any); ok {
		params["props"] = props
	}

	if usejournal, ok := p.Headers["usejournal"].(string); ok {
		params["usejournal"] = usejournal
	}

	params, err = c.rpc.addAuth(params, c.username, c.password)
	if err != nil {
		return "", err
	}

	result, err := c.rpc.call("LJ.XMLRPC.postevent", params)
	if err != nil {
		return "", err
	}

	// Prefer the URL from response as it contains the correct ditemid
	if url := getString(result, "url"); url != "" {
		return url, nil
	}

	// Fallback to itemid if URL not provided
	itemID := getInt(result, "itemid")
	if itemID == 0 {
		return "", errors.Errorf("no itemid in response")
	}

	return strconv.Itoa(itemID), nil
}

// extractItemID extracts the itemid from a remoteID which can be either a URL or numeric ID
func extractItemID(remoteID string) (int, error) {
	// If it's a URL like https://user.livejournal.com/12345.html, extract the number
	if strings.Contains(remoteID, "/") {
		// Extract filename from URL
		parts := strings.Split(remoteID, "/")
		lastPart := parts[len(parts)-1]
		// Remove .html suffix
		lastPart = strings.TrimSuffix(lastPart, ".html")
		if id, err := strconv.Atoi(lastPart); err == nil {
			return id, nil
		}
		return 0, errors.Errorf("cannot extract itemid from URL: %s", remoteID)
	}

	// Otherwise it's a numeric ID
	id, err := strconv.Atoi(remoteID)
	if err != nil {
		return 0, errors.Errorf("invalid remote ID: %s", remoteID)
	}
	return id, nil
}

func (c *client) Update(remoteID string, p *types.Post) error {
	itemID, err := extractItemID(remoteID)
	if err != nil {
		return err
	}

	body, err := p.Body.MaybeString()
	if err != nil {
		return err
	}

	now := time.Now()

	params := map[string]any{
		"ver":         1,
		"itemid":      itemID,
		"event":       body,
		"subject":     p.Headers["title"],
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
	}

	if security, ok := p.Headers["security"].(string); ok {
		params["security"] = security
	}

	if allowmask, ok := p.Headers["allowmask"].(int); ok {
		params["allowmask"] = allowmask
	}

	if props, ok := p.Headers["props"].(map[string]any); ok {
		params["props"] = props
	}

	if usejournal, ok := p.Headers["usejournal"].(string); ok {
		params["usejournal"] = usejournal
	}

	params, err = c.rpc.addAuth(params, c.username, c.password)
	if err != nil {
		return err
	}

	_, err = c.rpc.call("LJ.XMLRPC.editevent", params)
	return err
}

func (c *client) Delete(remoteID string) error {
	itemID, err := extractItemID(remoteID)
	if err != nil {
		return err
	}

	now := time.Now()

	// To delete, send empty event and subject
	params := map[string]any{
		"ver":         1,
		"itemid":      itemID,
		"event":       "",
		"subject":     "",
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
	}

	params, err = c.rpc.addAuth(params, c.username, c.password)
	if err != nil {
		return err
	}

	_, err = c.rpc.call("LJ.XMLRPC.editevent", params)
	return err
}

func (c *client) NewPostTemplate(name string) string {
	return fmt.Sprintf(`title: %s
privacy: public
tags: 

Write your post here in markdown!
`, name)
}

func (c *client) PostURL(remoteID string) string {
	// If remoteID is already a URL, return it directly
	if strings.HasPrefix(remoteID, "http://") || strings.HasPrefix(remoteID, "https://") {
		return remoteID
	}
	// Otherwise construct URL from itemid
	return fmt.Sprintf("https://%s.%s/%s.html", c.username, strings.TrimPrefix(c.cfg.Host, "https://"), remoteID)
}

func createClient(cfg types.Config) (types.Service, error) {
	password, err := pwd.GetAndSetPassword(cfg.Stored.Login, cfg.Host)
	if err != nil {
		return nil, err
	}

	host := cfg.Host
	if !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}

	return &client{
		cfg:      cfg,
		rpc:      newXMLRPCClient(host),
		username: cfg.Stored.Login,
		password: password,
	}, nil
}

func init() {
	types.DefaultServiceRepo.Register(
		types.NewServiceDefinition(
			"livejournal",
			"www.livejournal.com",
			createClient))

	types.DefaultServiceRepo.Register(
		types.NewServiceDefinition(
			"dreamwidth",
			"www.dreamwidth.org",
			createClient))
}
