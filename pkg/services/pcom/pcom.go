package pcom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/can3p/blg/pkg/types"
	"github.com/can3p/blg/pkg/util/pwd"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

const endpoint = "/api/v1"

type client struct {
	httpClient http.Client
	cfg        types.Config
	auth       string
}

type ApiPost struct {
	Subject     string `json:"subject"`
	MdBody      string `json:"md_body"`
	Visibility  string `json:"visibility"`
	IsPublished bool   `json:"is_published"`
}

func toApiPost(p *types.Post) (*ApiPost, error) {
	body, err := p.Body.MaybeString()

	if err != nil {
		return nil, err
	}

	return &ApiPost{
		Subject:     p.Headers["subject"].(string),
		MdBody:      body,
		Visibility:  p.Headers["visibility"].(string),
		IsPublished: p.Headers["published"].(bool),
	}, nil
}

var VisibilityValues = []string{"direct_only", "second_degree"}
var PublishedValues = []string{"yes", "no"}

func (c *client) NewPostTemplate(name string) string {
	return fmt.Sprintf(`subject: New post %s
visibility: %s
published: %s

Write yout new post there!`, name, strings.Join(VisibilityValues, " or "), strings.Join(PublishedValues, " or "))
}

func (c *client) PostURL(remoteID string) string {
	return c.cfg.Host + "/posts/" + remoteID
}

func (c *client) PreparePost(fields map[string]string, body string) (*types.Post, []string, error) {
	p := &types.Post{
		Headers: types.PostHeaders{},
	}

	if subject, ok := fields["subject"]; !ok {
		return nil, nil, errors.Errorf("`subject` field should be present")
	} else {
		p.Headers["subject"] = subject
	}

	if visibility, ok := fields["visibility"]; !ok {
		return nil, nil, errors.Errorf("`visibility` field should be present, possible values are %s", strings.Join(VisibilityValues, ", "))
	} else if !lo.Contains(VisibilityValues, visibility) {
		return nil, nil, errors.Errorf("`visibility` has value [%s], but only possible values are %s", visibility, strings.Join(VisibilityValues, ", "))
	} else {
		p.Headers["visibility"] = visibility
	}

	if published, ok := fields["published"]; !ok {
		return nil, nil, errors.Errorf("`published` field should be present, possible values are %s", strings.Join(PublishedValues, ", "))
	} else if !lo.Contains(PublishedValues, published) {
		return nil, nil, errors.Errorf("`published` has value [%s], but only possible values are %s", published, strings.Join(PublishedValues, ", "))
	} else {
		p.Headers["published"] = published == "yes"
	}

	if len(body) < 10 {
		return nil, nil, errors.Errorf("Post body should have at least 10 characters")
	}

	// parse the post and replace local image filenames with remote ids
	parser, err := parseBody(body)

	if err != nil {
		return nil, nil, err
	}

	p.Body = parser

	extractedLinks, err := parser.ExtractImages()

	if err != nil {
		return nil, nil, err
	}

	return p, extractedLinks, nil
}

// ```
// curl -v -H'Authorization: Bearer <api-key>' -XPUT -F 'file=@path/to/image.png' http://localhost:8080/api/v1/image
// {"data":{"ImageID":"0190478c-5592-74ab-9d1a-5cdab598f2dd.png"}}%
// ```
func (c *client) UploadImage(fname string) (string, error) {
	url := fmt.Sprintf("%s%s/image", c.cfg.Host, endpoint)

	file, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(fname))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)

	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.auth))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.Errorf("Failed to upload an image, return code should be 200, got %d instead", res.StatusCode)
	}

	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	var respParsed struct {
		Data struct {
			ImageID string `json:"image_id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &respParsed); err != nil {
		return "", err
	} else if respParsed.Data.ImageID == "" {
		return "", errors.Errorf("image_id is missing in upload image response")
	}

	return respParsed.Data.ImageID, nil
}

// ```
// curl -v -H'Authorization: Bearer <api-key>' -XPOST -d'{ "subject": "test post", "md_body": "is saved\n\n![trololo](0190478c-5592-74ab-9d1a-5cdab598f2dd.png)", "visibility": "direct_only" }' http://localhost:8080/api/v1/posts
// {"data":{"id":"01904796-62f7-7a9a-a7bd-1595ed6d1663","public_url":"http://localhost:8080/posts/01904796-62f7-7a9a-a7bd-1595ed6d1663"}}%
// ```
func (c *client) Create(p *types.Post) (string, error) {
	url := fmt.Sprintf("%s%s/posts", c.cfg.Host, endpoint)

	return c._sendPost(url, p)
}

func (c *client) Update(remoteID string, p *types.Post) error {
	url := fmt.Sprintf("%s%s/posts/%s", c.cfg.Host, endpoint, remoteID)

	_, err := c._sendPost(url, p)
	return err
}

func (c *client) _sendPost(url string, p *types.Post) (string, error) {
	payload, err := toApiPost(p)

	if err != nil {
		return "", err
	}

	b, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.auth))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.Errorf("Failed to save a post, return code should be 200, got %d instead", res.StatusCode)
	}

	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	var respParsed struct {
		Data struct {
			ID        string `json:"id"`
			PublicUrl string `json:"public_url"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &respParsed); err != nil {
		return "", err
	} else if respParsed.Data.ID == "" {
		return "", errors.Errorf("ID is missing in the response: %s", string(respBody))
	}

	return respParsed.Data.ID, nil
}

// ```
// curl -v -H'Authorization: Bearer <api-key>' -XDELETE http://localhost:8080/api/v1/posts/01904796-62f7-7a9a-a7bd-1595ed6d1663
// {"data":null}
// ```
func (c *client) Delete(remoteID string) error {
	url := fmt.Sprintf("%s%s/posts/%s", c.cfg.Host, endpoint, remoteID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)

	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.auth))

	res, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}

	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}

	return errors.Errorf("Failed to delete a post, return code should be 200, costs %d instead", res.StatusCode)
}

func createClient(cfg types.Config) (types.Service, error) {
	auth, err := pwd.GetAndSetPassword(cfg.Stored.Login, cfg.Host)

	if err != nil {
		return nil, err
	}

	return &client{
		cfg:  cfg,
		auth: auth,
		httpClient: http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func init() {
	types.DefaultServiceRepo.Register(
		types.NewServiceDefinition(
			"pcom",
			"https://www.pcom.com",
			createClient))
}
