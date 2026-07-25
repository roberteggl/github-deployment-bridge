// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"errors"
	"net/http"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v89/github"

	"github.com/roberteggl/github-deployment-bridge/pkg/retry"
)

func TestClassifyGitHubErrorRetryAfterHeader(t *testing.T) {
	t.Parallel()

	resp := &gogithub.Response{
		Response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Retry-After": []string{"12"}},
		},
	}
	err := classifyGitHubError(errors.New("429"), resp)
	d, ok := retry.SuggestedDelay(err)
	if !ok || d != 12*time.Second {
		t.Fatalf("SuggestedDelay = (%v, %v), want (12s, true)", d, ok)
	}
	if retry.IsPermanent(err) {
		t.Fatal("429 must not be permanent")
	}
}

func TestClassifyGitHubErrorAbuseRateLimit(t *testing.T) {
	t.Parallel()

	wait := 45 * time.Second
	abuse := &gogithub.AbuseRateLimitError{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Retry-After": []string{"45"}},
		},
		Message:    "secondary rate limit",
		RetryAfter: &wait,
	}
	resp := &gogithub.Response{Response: abuse.Response}
	err := classifyGitHubError(abuse, resp)
	d, ok := retry.SuggestedDelay(err)
	if !ok || d != wait {
		t.Fatalf("SuggestedDelay = (%v, %v), want (%v, true)", d, ok, wait)
	}
	if retry.IsPermanent(err) {
		t.Fatal("secondary rate limit 403 must not be permanent")
	}
}

func TestClassifyGitHubErrorRateLimitReset(t *testing.T) {
	t.Parallel()

	reset := time.Now().Add(90 * time.Second)
	rateErr := &gogithub.RateLimitError{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Message:  "API rate limit exceeded",
		Rate: gogithub.Rate{
			Limit:     5000,
			Remaining: 0,
			Reset:     gogithub.Timestamp{Time: reset},
		},
	}
	err := classifyGitHubError(rateErr, &gogithub.Response{Response: rateErr.Response})
	d, ok := retry.SuggestedDelay(err)
	if !ok {
		t.Fatal("expected suggested delay from rate reset")
	}
	if d < 60*time.Second || d > 95*time.Second {
		t.Fatalf("SuggestedDelay = %v, want ~90s", d)
	}
}

func TestClassifyGitHubErrorClientErrorPermanent(t *testing.T) {
	t.Parallel()

	resp := &gogithub.Response{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	err := classifyGitHubError(errors.New("not found"), resp)
	if !retry.IsPermanent(err) {
		t.Fatalf("expected permanent 404, got %v", err)
	}
}

func TestParseRetryAfterHeader(t *testing.T) {
	t.Parallel()

	d, ok := parseRetryAfterHeader("30")
	if !ok || d != 30*time.Second {
		t.Fatalf("seconds: got (%v, %v)", d, ok)
	}

	future := time.Now().UTC().Add(2 * time.Second).Format(http.TimeFormat)
	d, ok = parseRetryAfterHeader(future)
	if !ok || d <= 0 || d > 3*time.Second {
		t.Fatalf("http-date: got (%v, %v)", d, ok)
	}

	if _, ok := parseRetryAfterHeader(""); ok {
		t.Fatal("empty should fail")
	}
	if _, ok := parseRetryAfterHeader("-1"); ok {
		t.Fatal("negative should fail")
	}
}
