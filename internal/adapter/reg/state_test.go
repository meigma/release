package reg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/release/internal/rel"
	"github.com/meigma/release/internal/stage/puboci"
)

const (
	// testRepo is the repository path used by the in-process registry.
	testRepo = "owner/image"
	// testTag is the fixture tag written by successful cases.
	testTag = "1.2.3"
	// testVersion is the canonical version annotation written by fixtures.
	testVersion = "1.2.3"
	// testToken is a credential that must never appear in errors or formats.
	testToken = "ghs_this_must_never_appear_in_errors"
	// ociSchemaVersion is the OCI image-spec schema version.
	ociSchemaVersion = 2
)

func TestResolveReturnsManifestDigest(t *testing.T) {
	t.Parallel()

	server := newRegistryServer(t)
	body := indexJSON(t, map[string]string{ocispec.AnnotationVersion: testVersion})
	putManifest(t, server, testRepo, testTag, ocispec.MediaTypeImageIndex, body)

	got, err := newPlainClient(server).Resolve(context.Background(), mustRef(t, server))
	require.NoError(t, err)
	assert.Equal(t, digestOf(body), got.String())
}

func TestResolveMissingTagWrapsErrTagAbsent(t *testing.T) {
	t.Parallel()

	server := newRegistryServer(t)
	_, err := newPlainClient(server).Resolve(context.Background(), mustRef(t, server))
	require.Error(t, err)
	require.ErrorIs(t, err, puboci.ErrTagAbsent)
	assert.NotContains(t, err.Error(), testToken)
}

func TestVersionReadsAnnotation(t *testing.T) {
	t.Parallel()

	server := newRegistryServer(t)
	body := indexJSON(t, map[string]string{ocispec.AnnotationVersion: testVersion})
	putManifest(t, server, testRepo, testTag, ocispec.MediaTypeImageIndex, body)

	got, err := newPlainClient(server).Version(context.Background(), mustRef(t, server))
	require.NoError(t, err)
	assert.Equal(t, testVersion, got.String())
}

func TestVersionWrapsCorruptManifests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		body        []byte
	}{
		{
			name:        "no annotations",
			contentType: ocispec.MediaTypeImageIndex,
			body:        indexJSON(t, nil),
		},
		{
			name:        "empty version annotation",
			contentType: ocispec.MediaTypeImageIndex,
			body:        indexJSON(t, map[string]string{ocispec.AnnotationVersion: ""}),
		},
		{
			name:        "minor-only version",
			contentType: ocispec.MediaTypeImageIndex,
			body:        indexJSON(t, map[string]string{ocispec.AnnotationVersion: "1.2"}),
		},
		{
			name:        "v-prefixed version",
			contentType: ocispec.MediaTypeImageIndex,
			body:        indexJSON(t, map[string]string{ocispec.AnnotationVersion: "v1.2.3"}),
		},
		{
			name:        "prerelease version",
			contentType: ocispec.MediaTypeImageIndex,
			body:        indexJSON(t, map[string]string{ocispec.AnnotationVersion: "1.2.3-rc.1"}),
		},
		{
			name:        "non-JSON body",
			contentType: ocispec.MediaTypeImageManifest,
			body:        []byte("not-json"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := newRegistryServer(t)
			putManifest(t, server, testRepo, testTag, test.contentType, test.body)

			_, err := newPlainClient(server).Version(context.Background(), mustRef(t, server))
			require.Error(t, err)
			require.ErrorIs(t, err, puboci.ErrCorruptState)
		})
	}
}

func TestRegistryStatusClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   int
		wantSent error
		wantText string
	}{
		{
			name:     "service unavailable",
			status:   http.StatusServiceUnavailable,
			wantSent: puboci.ErrRetryable,
			wantText: "retryable",
		},
		{
			name:     "too many requests",
			status:   http.StatusTooManyRequests,
			wantSent: puboci.ErrRetryable,
			wantText: "retryable",
		},
		{
			name:     "unauthorized",
			status:   http.StatusUnauthorized,
			wantText: "registry authentication failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
			}))
			t.Cleanup(server.Close)

			client := New(Options{
				Credentials: Credentials{
					Username: "octocat",
					Password: rel.NewSecret(testToken),
				},
				PlainHTTP:  true,
				HTTPClient: server.Client(),
			})
			_, err := client.Resolve(context.Background(), mustRef(t, server))
			require.Error(t, err)
			if test.wantSent != nil {
				require.ErrorIs(t, err, test.wantSent)
			} else {
				require.NotErrorIs(t, err, puboci.ErrRetryable)
				require.NotErrorIs(t, err, puboci.ErrTagAbsent)
			}
			assert.Contains(t, err.Error(), test.wantText)
			assert.NotContains(t, err.Error(), testToken)
			assert.NotContains(t, err.Error(), "Authorization")
			assert.NotContains(t, err.Error(), server.URL)
		})
	}
}

func TestCanceledContextDoesNotSucceed(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newPlainClient(server).Resolve(ctx, mustRef(t, server))
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, hits.Load())
	assert.NotContains(t, err.Error(), server.URL)
	assert.NotContains(t, err.Error(), testToken)
}

func TestClientFormatOmitsToken(t *testing.T) {
	t.Parallel()

	client := New(Options{Credentials: Credentials{
		Username: "octocat",
		Password: rel.NewSecret(testToken),
	}})
	assert.NotContains(t, fmt.Sprintf("%v", client), testToken)
	assert.NotContains(t, fmt.Sprintf("%+v", client), testToken)
}

// newRegistryServer starts an in-process OCI registry.
func newRegistryServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(registry.New())
	t.Cleanup(server.Close)

	return server
}

// newPlainClient returns an anonymous client pointed at server over HTTP.
func newPlainClient(server *httptest.Server) *Client {
	return New(Options{
		PlainHTTP:  true,
		HTTPClient: server.Client(),
	})
}

// putManifest writes body as a tagged manifest on the fake registry.
func putManifest(t *testing.T, server *httptest.Server, repo, tag, contentType string, body []byte) {
	t.Helper()

	req, err := http.NewRequest(
		http.MethodPut,
		server.URL+"/v2/"+repo+"/manifests/"+tag,
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)

	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// indexJSON encodes a minimal OCI image index with the given annotations.
func indexJSON(t *testing.T, annotations map[string]string) []byte {
	t.Helper()

	payload := struct {
		SchemaVersion int               `json:"schemaVersion"`
		MediaType     string            `json:"mediaType"`
		Manifests     []struct{}        `json:"manifests"`
		Annotations   map[string]string `json:"annotations,omitempty"`
	}{
		SchemaVersion: ociSchemaVersion,
		MediaType:     ocispec.MediaTypeImageIndex,
		Manifests:     []struct{}{},
		Annotations:   annotations,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	return body
}

// digestOf returns the sha256:<hex> digest of body.
func digestOf(body []byte) string {
	sum := sha256.Sum256(body)

	return "sha256:" + hex.EncodeToString(sum[:])
}

// mustRef builds the fixture reference for testRepo and testTag on server.
func mustRef(t *testing.T, server *httptest.Server) puboci.Reference {
	t.Helper()

	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	image, err := puboci.ParseImage(parsed.Host + "/" + testRepo)
	require.NoError(t, err)
	tag, err := rel.ParseTag(testTag)
	require.NoError(t, err)

	return image.Reference(tag)
}
