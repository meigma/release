package puboci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/meigma/release/internal/rel"
)

// PrepareInput is the candidate image, layout, and expected index digest.
type PrepareInput struct {
	// Image is the untagged repository that will receive the content.
	Image Image
	// Version is the candidate MAJOR.MINOR.PATCH release.
	Version rel.Version
	// IndexDigest is the expected index digest, cross-checked against the layout.
	IndexDigest rel.Digest
	// Layout is a filesystem rooted at the extracted oci-image/layout directory.
	Layout fs.FS
	// DryRun skips every write, verification, and signature.
	DryRun bool
}

// Prepare reads a local OCI layout, plans tags, and publishes digest-addressed content.
//
// It fails closed before any write when the layout index digest does not match
// [PrepareInput.IndexDigest], when [CollectState] fails, or when [rel.PlanTags]
// reports an immutable-tag or corrupt-channel conflict. A dry run returns
// [NewPrepareResult] with Authoritative false and never calls pusher or signer;
// those ports may be nil only in that mode. A real prepare pushes every layout
// blob, then each platform manifest, then the index, verifies the index and
// every platform digest (a deliberate strengthening of the workflow, which
// verifies only the index), and signs the index recursively. Errors name the
// failing step and descriptor digest and wrap the underlying error. Prepare
// does not retry transient failures.
func Prepare(
	ctx context.Context,
	input PrepareInput,
	state StateReader,
	pusher ContentPusher,
	signer Signer,
) (OCIPrepareResult, error) {
	if err := validatePrepare(ctx, input, state); err != nil {
		return OCIPrepareResult{}, err
	}

	layout, err := ReadLayout(input.Layout)
	if err != nil {
		return OCIPrepareResult{}, fmt.Errorf("read layout: %w", err)
	}
	if layout.Index.Digest != input.IndexDigest {
		return OCIPrepareResult{}, fmt.Errorf(
			"layout index digest %s does not match expected %s",
			layout.Index.Digest,
			input.IndexDigest,
		)
	}

	current, err := CollectState(ctx, state, input.Image, input.Version, input.IndexDigest)
	if err != nil {
		return OCIPrepareResult{}, fmt.Errorf("collect state: %w", err)
	}
	if _, err := rel.PlanTags(input.Version, input.IndexDigest, current); err != nil {
		return OCIPrepareResult{}, fmt.Errorf("plan tags: %w", err)
	}

	if input.DryRun {
		return NewPrepareResult(
			input.Image,
			input.Version,
			input.IndexDigest,
			layout.Platforms,
			current,
			false,
		), nil
	}
	if pusher == nil {
		return OCIPrepareResult{}, errors.New("content pusher is nil")
	}
	if signer == nil {
		return OCIPrepareResult{}, errors.New("signer is nil")
	}
	if err := pushContent(ctx, input.Image, input.Layout, layout, pusher); err != nil {
		return OCIPrepareResult{}, err
	}
	if err := verifyContent(ctx, input.Image, layout, pusher); err != nil {
		return OCIPrepareResult{}, err
	}

	indexRef := input.Image.Pin(input.IndexDigest)
	if err := signer.SignRecursive(ctx, indexRef); err != nil {
		return OCIPrepareResult{}, fmt.Errorf("sign %s: %w", indexRef, err)
	}

	return NewPrepareResult(
		input.Image,
		input.Version,
		input.IndexDigest,
		layout.Platforms,
		current,
		true,
	), nil
}

// validatePrepare rejects a nil context, a nil layout, a zero image, version,
// or digest, and a nil state reader.
func validatePrepare(ctx context.Context, input PrepareInput, state StateReader) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	if input.Layout == nil {
		return errors.New("layout is nil")
	}
	if input.Image == "" {
		return errors.New("image is empty")
	}
	if input.Version == (rel.Version{}) {
		return errors.New("version is zero")
	}
	if input.IndexDigest == "" {
		return errors.New("digest is empty")
	}
	if state == nil {
		return errors.New("state reader is nil")
	}

	return nil
}

// pushContent uploads blobs, platform manifests, then the index, in that order.
func pushContent(ctx context.Context, image Image, fsys fs.FS, layout Layout, pusher ContentPusher) error {
	for _, blob := range layout.Blobs {
		if err := pushBlob(ctx, image, fsys, blob, pusher); err != nil {
			return err
		}
	}
	for _, platform := range layout.Platforms {
		if err := pushManifest(ctx, image, fsys, platform.Descriptor, pusher); err != nil {
			return err
		}
	}

	return pushIndex(ctx, image, layout, pusher)
}

// pushBlob streams one config or layer blob from the layout filesystem.
func pushBlob(ctx context.Context, image Image, fsys fs.FS, desc Descriptor, pusher ContentPusher) error {
	file, err := openLayoutBlob(fsys, desc.Digest)
	if err != nil {
		return fmt.Errorf("push blob %s: %w", desc.Digest, err)
	}
	err = pusher.PushBlob(ctx, image, desc, file)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("push blob %s: %w", desc.Digest, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close blob %s: %w", desc.Digest, closeErr)
	}

	return nil
}

// pushManifest streams one platform manifest from the layout filesystem.
func pushManifest(ctx context.Context, image Image, fsys fs.FS, desc Descriptor, pusher ContentPusher) error {
	file, err := openLayoutBlob(fsys, desc.Digest)
	if err != nil {
		return fmt.Errorf("push manifest %s: %w", desc.Digest, err)
	}
	err = pusher.PushManifest(ctx, image, desc, file)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("push manifest %s: %w", desc.Digest, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close manifest %s: %w", desc.Digest, closeErr)
	}

	return nil
}

// pushIndex uploads the exact index.json bytes retained by [ReadLayout].
func pushIndex(ctx context.Context, image Image, layout Layout, pusher ContentPusher) error {
	if err := pusher.PushManifest(ctx, image, layout.Index, bytes.NewReader(layout.IndexBytes)); err != nil {
		return fmt.Errorf("push index %s: %w", layout.Index.Digest, err)
	}

	return nil
}

// verifyContent requires the published index, then each platform, to resolve.
//
// The publish workflow verifies only the index. Checking every platform
// manifest digest as well is a deliberate strengthening so a partial push
// cannot look successful.
func verifyContent(ctx context.Context, image Image, layout Layout, pusher ContentPusher) error {
	indexRef := image.Pin(layout.Index.Digest)
	if err := pusher.Verify(ctx, indexRef); err != nil {
		return fmt.Errorf("verify index %s: %w", layout.Index.Digest, err)
	}
	for _, platform := range layout.Platforms {
		if err := pusher.Verify(ctx, image.Pin(platform.Descriptor.Digest)); err != nil {
			return fmt.Errorf("verify manifest %s: %w", platform.Descriptor.Digest, err)
		}
	}

	return nil
}

// openLayoutBlob opens digest's blob file on fsys for streaming.
func openLayoutBlob(fsys fs.FS, digest rel.Digest) (fs.File, error) {
	name, err := BlobPath(digest)
	if err != nil {
		return nil, err
	}
	file, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}

	return file, nil
}
