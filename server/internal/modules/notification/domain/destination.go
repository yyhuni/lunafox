package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Provider identifies one installation-owned outbound destination protocol.
type Provider string

const (
	ProviderDiscord Provider = "discord"
	ProviderWeCom   Provider = "wecom"
	ProviderFeishu  Provider = "feishu"
)

var fixedProviders = [...]Provider{
	ProviderDiscord,
	ProviderWeCom,
	ProviderFeishu,
}

// FixedProviders returns the closed installation-owned destination registry in
// its stable display and startup materialization order.
func FixedProviders() []Provider {
	return append([]Provider(nil), fixedProviders[:]...)
}

// Destination contains complete credential material only at the authorized
// settings and delivery boundaries. It must never be logged or copied to an
// attempt record.
type Destination struct {
	ID            int64
	Provider      Provider
	Credential    string
	Enabled       bool
	Subscriptions []Kind
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ValidateProvider(provider Provider) error {
	switch provider {
	case ProviderDiscord, ProviderWeCom, ProviderFeishu:
		return nil
	default:
		return fmt.Errorf("unsupported notification provider %q", provider)
	}
}

// ValidateDestinationCredential enforces the provider-owned outbound endpoint
// boundary before a credential can be persisted or used for a test delivery.
// The allowlist prevents this setting from becoming an arbitrary SSRF proxy.
func ValidateDestinationCredential(provider Provider, credential string) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	parsedCredential := strings.TrimSpace(credential)
	if provider == ProviderFeishu {
		// Feishu accepts one exact endpoint spelling. Unlike the legacy
		// providers, trimming would silently turn an unsafe stored value into a
		// different outbound target.
		if credential != parsedCredential {
			return fmt.Errorf("Feishu webhook URL must not contain leading or trailing whitespace")
		}
		parsedCredential = credential
	}
	parsed, err := url.Parse(parsedCredential)
	if err != nil {
		return fmt.Errorf("destination credential URL is invalid: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("destination credential URL must use https")
	}
	if parsed.User != nil || parsed.Port() != "" || parsed.Fragment != "" {
		return fmt.Errorf("destination credential URL contains unsupported authority or fragment data")
	}
	// RawPath is non-empty only when URL parsing retained an escaped alternate
	// path. Reject it instead of normalizing an encoded path into a trusted one.
	if parsed.RawPath != "" {
		return fmt.Errorf("destination credential URL must not use an encoded path")
	}

	switch provider {
	case ProviderDiscord:
		return validateDiscordWebhookURL(parsed)
	case ProviderWeCom:
		return validateWeComWebhookURL(parsed)
	case ProviderFeishu:
		return validateFeishuWebhookURL(parsed)
	default:
		return fmt.Errorf("unsupported notification provider %q", provider)
	}
}

func validateDiscordWebhookURL(parsed *url.URL) error {
	if parsed.Host != "discord.com" {
		return fmt.Errorf("Discord webhook URL must use the official discord.com host")
	}
	if parsed.RawQuery != "" {
		return fmt.Errorf("Discord webhook URL must not contain query parameters")
	}
	segments := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(segments) != 4 || segments[0] != "api" || segments[1] != "webhooks" || segments[2] == "" || segments[3] == "" {
		return fmt.Errorf("Discord webhook URL must use /api/webhooks/{id}/{token}")
	}
	return nil
}

func validateWeComWebhookURL(parsed *url.URL) error {
	if parsed.Host != "qyapi.weixin.qq.com" {
		return fmt.Errorf("WeCom webhook URL must use the official qyapi.weixin.qq.com host")
	}
	if parsed.Path != "/cgi-bin/webhook/send" {
		return fmt.Errorf("WeCom webhook URL must use /cgi-bin/webhook/send")
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 1 || len(query["key"]) != 1 || strings.TrimSpace(query.Get("key")) == "" {
		return fmt.Errorf("WeCom webhook URL must contain exactly one non-empty key query parameter")
	}
	return nil
}

func validateFeishuWebhookURL(parsed *url.URL) error {
	if parsed.Host != "open.feishu.cn" {
		return fmt.Errorf("Feishu webhook URL must use the official open.feishu.cn host")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return fmt.Errorf("Feishu webhook URL must not contain query parameters")
	}
	segments := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(segments) != 5 ||
		segments[0] != "open-apis" ||
		segments[1] != "bot" ||
		segments[2] != "v2" ||
		segments[3] != "hook" ||
		segments[4] == "" {
		return fmt.Errorf("Feishu webhook URL must use /open-apis/bot/v2/hook/{token}")
	}
	return nil
}

// RequiresWebhookUpdate identifies persisted non-empty credentials that no
// longer satisfy the current delivery boundary. It is derived server-side so
// browser guidance cannot drift from persistence and delivery enforcement.
func RequiresWebhookUpdate(destination Destination) bool {
	credential := destination.Credential
	if destination.Provider != ProviderFeishu {
		credential = strings.TrimSpace(credential)
	}
	return credential != "" && ValidateDestinationCredential(destination.Provider, credential) != nil
}

func ValidateDestinationEnabled(enabled bool, subscriptions []Kind) error {
	seen := make(map[Kind]struct{}, len(subscriptions))
	for _, kind := range subscriptions {
		if !IsExternallyDeliverableKind(kind) {
			return fmt.Errorf("unsupported destination subscription kind %q", kind)
		}
		if _, duplicate := seen[kind]; duplicate {
			return fmt.Errorf("duplicate destination subscription kind %q", kind)
		}
		seen[kind] = struct{}{}
	}
	if enabled && len(subscriptions) == 0 {
		return fmt.Errorf("enabled notification destination requires at least one kind subscription")
	}
	return nil
}

// ValidateDestination permits an untouched disabled default while requiring a
// complete parseable credential before a destination can become externally
// reachable.
func ValidateDestination(destination Destination) error {
	if err := ValidateProvider(destination.Provider); err != nil {
		return err
	}
	if err := ValidateDestinationEnabled(destination.Enabled, destination.Subscriptions); err != nil {
		return err
	}
	credential := destination.Credential
	if destination.Provider != ProviderFeishu {
		credential = strings.TrimSpace(credential)
	}
	if credential == "" {
		if destination.Enabled {
			return fmt.Errorf("enabled notification destination requires a credential")
		}
		return nil
	}
	return ValidateDestinationCredential(destination.Provider, credential)
}
