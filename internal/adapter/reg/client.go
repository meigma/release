package reg

import (
	"fmt"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/meigma/release/internal/rel"
	"github.com/meigma/release/internal/stage/puboci"
)

// Credentials is a registry username and password.
//
// A zero value is an anonymous read. Password is a [rel.Secret] so token
// text is not printed, logged, or encoded.
type Credentials struct {
	// Username is the registry user. An empty username with a password is
	// valid for token auth.
	Username string

	// Password is the registry password or token. Reveal it only when
	// composing the oras credential.
	Password rel.Secret
}

// Options configures a read-only registry [Client].
type Options struct {
	// Credentials authenticates registry reads. The zero value is anonymous.
	Credentials Credentials

	// PlainHTTP forces HTTP instead of HTTPS. Tests use this against a
	// local registry.
	PlainHTTP bool

	// HTTPClient is the optional transport. Nil selects a default client.
	HTTPClient *http.Client
}

// Client reads tag state from a GHCR-compatible registry.
//
// It implements [puboci.StateReader]. It never pushes, tags, or deletes.
type Client struct {
	// auth is the shared oras auth client. Credential is applied per
	// request so the registry host is known and token text is not stored
	// on this value.
	auth *auth.Client

	// options is the constructor configuration, including the redacted
	// secret used to build per-request credentials.
	options Options
}

// New constructs a [Client] from options.
//
// Token text stays inside [rel.Secret] until a request is built. The
// returned client is safe to format: password text is never a plain field.
func New(options Options) *Client {
	return &Client{
		auth: &auth.Client{
			Client: options.HTTPClient,
			Cache:  auth.NewCache(),
		},
		options: options,
	}
}

// String reports the client without credential material.
func (c *Client) String() string {
	if c == nil {
		return "<nil>"
	}

	return fmt.Sprintf("reg.Client{authenticated:%t plainHTTP:%t}", c.hasCredentials(), c.options.PlainHTTP)
}

// GoString reports the client without credential material.
func (c *Client) GoString() string {
	return c.String()
}

// repository builds a remote repository client for ref.
func (c *Client) repository(ref puboci.Reference) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref.Image.String())
	if err != nil {
		return nil, fmt.Errorf("parse repository: %w", err)
	}

	authClient := *c.auth
	if c.hasCredentials() {
		authClient.Credential = auth.StaticCredential(repo.Reference.Host(), auth.Credential{
			Username: c.options.Credentials.Username,
			Password: c.options.Credentials.Password.Reveal(),
		})
	}

	repo.Client = &authClient
	repo.PlainHTTP = c.options.PlainHTTP

	return repo, nil
}

// hasCredentials reports whether options include a username or password.
func (c *Client) hasCredentials() bool {
	return c.options.Credentials.Username != "" || !c.options.Credentials.Password.IsEmpty()
}
