package types

// LinkResolver is a function that takes a URL and returns the local filename
// if the URL corresponds to a known post, or false if not found.
type LinkResolver func(url string) (filename string, found bool)

// PostURLMapping represents a mapping from a remote URL to a local filename.
type PostURLMapping struct {
	URL      string
	Filename string
}

// BuildLinkResolver creates a LinkResolver from a list of URL-to-filename mappings.
func BuildLinkResolver(mappings []PostURLMapping) LinkResolver {
	urlMap := make(map[string]string, len(mappings))
	for _, m := range mappings {
		urlMap[m.URL] = m.Filename
	}

	return func(u string) (string, bool) {
		fname, ok := urlMap[u]
		return fname, ok
	}
}

// BuildLinkResolverFromMap creates a LinkResolver from a URL-to-filename map.
func BuildLinkResolverFromMap(urlMap map[string]string) LinkResolver {
	return func(u string) (string, bool) {
		fname, ok := urlMap[u]
		return fname, ok
	}
}
