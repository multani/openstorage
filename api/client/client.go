package client

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/libopenstorage/openstorage/cluster"
	"github.com/libopenstorage/openstorage/config"
	"github.com/libopenstorage/openstorage/volume"
)

var (
	httpCache = make(map[string]*http.Client)
	cacheLock sync.Mutex
)

// NewClient returns a new REST client for specified server.
func NewClient(host string, version string) (*Client, error) {
	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if baseURL.Path == "" {
		baseURL.Path = "/"
	}
	unix2HTTP(baseURL)
	c := &Client{
		base:       baseURL,
		version:    version,
		httpClient: getHttpClient(host),
	}
	return c, nil
}

// NewClusterClient returns a new REST client of the supplied version for cluster management.
func NewClusterClient(version string) (*Client, error) {
	sockPath := "unix://" + config.ClusterAPIBase + "osd.sock"
	if version == "" {
		// Set the default version
		version = config.Version
	}

	return NewClient(sockPath, version)
}

// GetSupportedClusterVersions returns a list of supported versions
// of the OSD cluster api
func GetSupportedClusterVersions(serverPath string) ([]string, error) {
	if serverPath == "" {
		serverPath = "unix://" + config.ClusterAPIBase + "osd.sock"
	}
	client, err := NewClient(serverPath, "")
	if err != nil {
		return []string{}, err
	}
	versions, err := client.Versions("cluster")
	if err != nil {
		return []string{}, err
	}
	return versions, nil
}

// NewDriver returns a new REST client of the supplied version for specified driver.
func NewDriverClient(driverName, version string) (*Client, error) {
	sockPath := "unix://" + config.DriverAPIBase + driverName + ".sock"
	if version == "" {
		// Set the default version
		version = config.Version
	}
	return NewClient(sockPath, version)
}

// GetSupportedDriverVersions returns a list of supported versions
// for the provided driver. It uses the given server endpoint or the
// standard unix domain socket
func GetSupportedDriverVersions(driverName, serverPath string) ([]string, error) {
	// Get a client handler
	if serverPath == "" {
		serverPath = "unix://" + config.DriverAPIBase + driverName + ".sock"
	}

	client, err := NewClient(serverPath, "")
	if err != nil {
		return []string{}, err
	}
	versions, err := client.Versions("osd-volumes")
	if err != nil {
		return []string{}, err
	}
	return versions, nil
}

// Client is an HTTP REST wrapper. Use one of Get/Post/Put/Delete to get a request
// object.
type Client struct {
	base       *url.URL
	version    string
	httpClient *http.Client
}

// VolumeDriver returns a REST wrapper for the VolumeDriver interface.
func (c *Client) VolumeDriver() volume.VolumeDriver {
	return newVolumeClient(c)
}

// ClusterManager returns a REST wrapper for the Cluster interface.
func (c *Client) ClusterManager() cluster.Cluster {
	return newClusterClient(c)
}

// Status sends a Status request at the /status REST endpoint.
func (c *Client) Status() (*Status, error) {
	status := &Status{}
	err := c.Get().UsePath("/status").Do().Unmarshal(status)
	return status, err
}

// Version send a request at the /versions REST endpoint.
func (c *Client) Versions(endpoint string) ([]string, error) {
	versions := []string{}
	err := c.Get().Resource(endpoint + "/versions").Do().Unmarshal(&versions)
	return versions, err
}

// Get returns a Request object setup for GET call.
func (c *Client) Get() *Request {
	return NewRequest(c.httpClient, c.base, "GET", c.version)
}

// Post returns a Request object setup for POST call.
func (c *Client) Post() *Request {
	return NewRequest(c.httpClient, c.base, "POST", c.version)
}

// Put returns a Request object setup for PUT call.
func (c *Client) Put() *Request {
	return NewRequest(c.httpClient, c.base, "PUT", c.version)
}

// Put returns a Request object setup for DELETE call.
func (c *Client) Delete() *Request {
	return NewRequest(c.httpClient, c.base, "DELETE", c.version)
}

func unix2HTTP(u *url.URL) {
	if u.Scheme == "unix" {
		// Override the main URL object so the HTTP lib won't complain
		u.Scheme = "http"
		u.Host = "unix.sock"
		u.Path = ""
	}
}

func newHTTPClient(u *url.URL, tlsConfig *tls.Config, timeout time.Duration) *http.Client {
	httpTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	switch u.Scheme {
	case "unix":
		socketPath := u.Path
		unixDial := func(proto, addr string) (net.Conn, error) {
			ret, err := net.DialTimeout("unix", socketPath, timeout)
			return ret, err
		}
		httpTransport.Dial = unixDial
		unix2HTTP(u)
	default:
		httpTransport.Dial = func(proto, addr string) (net.Conn, error) {
			return net.DialTimeout(proto, addr, timeout)
		}
	}

	return &http.Client{Transport: httpTransport}
}

func getHttpClient(host string) *http.Client {
	c, ok := httpCache[host]
	if !ok {
		cacheLock.Lock()
		defer cacheLock.Unlock()
		c, ok = httpCache[host]
		if !ok {
			u, err := url.Parse(host)
			if err != nil {
				// TODO(pedge): clean up
				fmt.Println("Failed to parse into url", host)
				return nil
			}
			if u.Path == "" {
				u.Path = "/"
			}
			c = newHTTPClient(u, nil, 10*time.Second)
			httpCache[host] = c
		}
	}
	return c
}
