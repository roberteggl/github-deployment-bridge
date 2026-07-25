// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

// Package registry reads OCI image config labels without pulling layers.
package registry

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/roberteggl/github-deployment-bridge/internal/metrics"
	"github.com/roberteggl/github-deployment-bridge/pkg/ocilabels"
	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

// Inspector fetches OCI metadata for container images.
type Inspector interface {
	Inspect(ctx context.Context, image string) (ocilabels.Metadata, error)
}

// Client inspects images via go-containerregistry.
type Client struct {
	keychain authn.Keychain
	metrics  *metrics.Metrics
	retry    retry.Config
}

// Options configures a registry Client.
type Options struct {
	Keychain authn.Keychain
	Metrics  *metrics.Metrics
	Retry    retry.Config
}

// New creates a registry Client.
func New(opts Options) *Client {
	kc := opts.Keychain
	if kc == nil {
		kc = authn.DefaultKeychain
	}
	cfg := opts.Retry
	if cfg.MaxAttempts == 0 {
		cfg = retry.Default()
	}
	return &Client{
		keychain: kc,
		metrics:  opts.Metrics,
		retry:    cfg,
	}
}

// Inspect fetches the image manifest and config blob, then extracts OCI labels.
// Image layers are never pulled.
func (c *Client) Inspect(ctx context.Context, image string) (ocilabels.Metadata, error) {
	var meta ocilabels.Metadata
	err := retry.Do(ctx, c.retry, func(ctx context.Context) error {
		ref, err := name.ParseReference(image)
		if err != nil {
			c.inc("error")
			return retry.Permanent(fmt.Errorf("parse image reference %q: %w", image, err))
		}

		desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(c.keychain))
		if err != nil {
			c.inc("error")
			return fmt.Errorf("fetch image manifest %q: %w", image, err)
		}

		img, err := desc.Image()
		if err != nil {
			c.inc("error")
			return fmt.Errorf("resolve image %q: %w", image, err)
		}

		cfg, err := img.ConfigFile()
		if err != nil {
			c.inc("error")
			return fmt.Errorf("fetch image config %q: %w", image, err)
		}
		if cfg == nil {
			c.inc("error")
			return retry.Permanent(fmt.Errorf("image %q has empty config", image))
		}

		meta = ocilabels.Extract(cfg.Config.Labels)
		if d := desc.Digest; d.String() != "" {
			meta.Digest = d.String()
		}
		c.inc("success")
		return nil
	})
	return meta, err
}

func (c *Client) inc(result string) {
	if c.metrics == nil {
		return
	}
	c.metrics.OCIRequestsTotal.WithLabelValues(result).Inc()
}
