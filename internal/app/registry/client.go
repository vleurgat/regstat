package registry

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/manifest/schema2"
)

// Client represents a HTTP client connection to a Docker registry.
type Client struct {
	client       *http.Client
	dockerConfig *configfile.ConfigFile
}

// CreateClient create a Client object.
func CreateClient(dockerConfig *configfile.ConfigFile) Client {
	return Client{
		client:       &http.Client{Timeout: 10 * time.Second},
		dockerConfig: dockerConfig,
	}
}

func parseBearer(suffix string) map[string]string {
	kv := make(map[string]string)
	for _, token := range strings.Split(suffix, ",") {
		token = strings.Trim(token, " ")
		if parts := strings.SplitN(token, "=", 2); len(parts) == 2 {
			kv[parts[0]] = strings.Trim(parts[1], `"`)
		} else {
			kv[token] = ""
		}
	}
	return kv
}

func (c *Client) getResponse(req *http.Request, auth string, target interface{}) (*http.Response, error) {
	addHeaders(req, "application/vnd.docker.distribution.manifest.v2+json", auth)
	r, err := c.client.Do(req)
	if err != nil {
		log.Println("failed to get response", err)
		return nil, err
	}
	return r, nil
}

func (c *Client) getJSONFromURL(queryURL string, target interface{}) error {
	request, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return err
	}
	basicAuth := c.getDockerBasicAuth(request)
	response, err := c.getResponse(request, basicAuth, target)
	if err != nil {
		return err
	}
	switch response.StatusCode {
	case 401:
		// try bearer
		bearerAuth, err := c.getDockerBearerAuth(response, basicAuth)
		if err != nil {
			return err
		}
		response, err = c.getResponse(request, bearerAuth, target)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			log.Println("failed to get good response with bearer auth", response)
			return errors.New("failed to get a good response with bearer auth- status code " + string(response.StatusCode))
		}
	case 200:
		// all good - nothing to do
	default:
		// oops
		log.Println("failed to get good response", response)
		return errors.New("failed to get a good response - status code " + string(response.StatusCode))
	}
	defer response.Body.Close()
	log.Println("got manifest response from URL", response)
	return json.NewDecoder(response.Body).Decode(target)
}

func getBrearerAuthURL(response *http.Response) (string, error) {
	header := response.Header.Get("Www-Authenticate")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errors.New("no bearer Www-Authenticate header")
	}
	bearer := header[7:]
	bearerKv := parseBearer(bearer)
	bearerURL, err := url.Parse(bearerKv["realm"])
	if err != nil {
		return "", err
	}
	bearerURL.RawQuery = url.Values{
		"service": []string{bearerKv["service"]},
		"scope":   []string{bearerKv["scope"]},
	}.Encode()
	return bearerURL.String(), nil
}

func extractBearerToken(response *http.Response) (string, error) {
	defer response.Body.Close()
	type tokenResponse struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	tr := new(tokenResponse)
	err := json.NewDecoder(response.Body).Decode(tr)
	if err != nil {
		return "", err
	}
	return "Bearer " + tr.Token, nil
}

func (c *Client) getDockerBearerAuth(response *http.Response, basicAuth string) (string, error) {
	bearerURL, err := getBrearerAuthURL(response)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", bearerURL, nil)
	if err != nil {
		return "", err
	}
	addHeaders(req, "application/json", basicAuth)
	response, err = c.client.Do(req)
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", errors.New("failed to determine the bearer token - status code is " + string(response.StatusCode))
	}
	return extractBearerToken(response)
}

func (c *Client) getDockerBasicAuth(req *http.Request) string {
	basicAuth := ""
	if c.dockerConfig != nil {
		config, exists := c.dockerConfig.AuthConfigs[req.Host]
		if exists && config.Auth != "" {
			basicAuth = "Basic " + config.Auth
		}
	}
	return basicAuth
}

func addHeaders(req *http.Request, accept string, auth string) {
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("User-Agent", "regstat")
	req.Header.Set("Accept", accept)
}

// GetV2Manifest returns the Docker V2 manifest object that corresponds with the provided registry URL.
func (c *Client) GetV2Manifest(url string) (schema2.Manifest, error) {
	v2Manifest := schema2.Manifest{}
	err := c.getJSONFromURL(url, &v2Manifest)
	if err != nil {
		log.Println("failed to get v2 manifest", url, err)
	} else {
		log.Println("v2 manifest", v2Manifest)
	}
	return v2Manifest, nil
}
