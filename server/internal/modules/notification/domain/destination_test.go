package domain

import "testing"

func TestValidateDestinationCredentialAcceptsOnlyOfficialProviderWebhookShapes(t *testing.T) {
	tests := []struct {
		name       string
		provider   Provider
		credential string
		valid      bool
	}{
		{name: "Discord", provider: ProviderDiscord, credential: "https://discord.com/api/webhooks/123/token", valid: true},
		{name: "WeCom", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=token", valid: true},
		{name: "Feishu", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token", valid: true},
		{name: "Discord rejects HTTP", provider: ProviderDiscord, credential: "http://discord.com/api/webhooks/123/token"},
		{name: "Discord rejects alternate host", provider: ProviderDiscord, credential: "https://discord.example.com/api/webhooks/123/token"},
		{name: "Discord rejects port", provider: ProviderDiscord, credential: "https://discord.com:443/api/webhooks/123/token"},
		{name: "Discord rejects userinfo", provider: ProviderDiscord, credential: "https://user@discord.com/api/webhooks/123/token"},
		{name: "Discord rejects query", provider: ProviderDiscord, credential: "https://discord.com/api/webhooks/123/token?wait=true"},
		{name: "Discord rejects fragment", provider: ProviderDiscord, credential: "https://discord.com/api/webhooks/123/token#fragment"},
		{name: "Discord rejects encoded path", provider: ProviderDiscord, credential: "https://discord.com/api%2Fwebhooks/123/token"},
		{name: "Discord rejects missing token", provider: ProviderDiscord, credential: "https://discord.com/api/webhooks/123"},
		{name: "WeCom rejects extra query", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=token&debug=true"},
		{name: "WeCom rejects duplicate key", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=one&key=two"},
		{name: "WeCom rejects empty key", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="},
		{name: "WeCom rejects missing key", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"},
		{name: "WeCom rejects alternate host", provider: ProviderWeCom, credential: "https://wecom.example.test/cgi-bin/webhook/send?key=token"},
		{name: "WeCom rejects alternate path", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/receive?key=token"},
		{name: "WeCom rejects encoded path", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin%2Fwebhook/send?key=token"},
		{name: "WeCom rejects fragment", provider: ProviderWeCom, credential: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=token#fragment"},
		{name: "Feishu rejects HTTP", provider: ProviderFeishu, credential: "http://open.feishu.cn/open-apis/bot/v2/hook/token"},
		{name: "Feishu rejects alternate host", provider: ProviderFeishu, credential: "https://feishu.example.test/open-apis/bot/v2/hook/token"},
		{name: "Feishu rejects custom port", provider: ProviderFeishu, credential: "https://open.feishu.cn:443/open-apis/bot/v2/hook/token"},
		{name: "Feishu rejects userinfo", provider: ProviderFeishu, credential: "https://user@open.feishu.cn/open-apis/bot/v2/hook/token"},
		{name: "Feishu rejects query", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token?debug=true"},
		{name: "Feishu rejects empty query", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token?"},
		{name: "Feishu rejects fragment", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token#fragment"},
		{name: "Feishu rejects encoded path", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook%2Ftoken"},
		{name: "Feishu rejects missing token", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook"},
		{name: "Feishu rejects trailing slash", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token/"},
		{name: "Feishu rejects extra path segment", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token/extra"},
		{name: "Feishu rejects leading whitespace", provider: ProviderFeishu, credential: " https://open.feishu.cn/open-apis/bot/v2/hook/token"},
		{name: "Feishu rejects trailing whitespace", provider: ProviderFeishu, credential: "https://open.feishu.cn/open-apis/bot/v2/hook/token "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDestinationCredential(test.provider, test.credential)
			if test.valid && err != nil {
				t.Fatalf("ValidateDestinationCredential() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("ValidateDestinationCredential() error = nil, want rejection")
			}
		})
	}
}

func TestFixedProvidersHasOneStableClosedOrder(t *testing.T) {
	providers := FixedProviders()
	want := []Provider{ProviderDiscord, ProviderWeCom, ProviderFeishu}
	if len(providers) != len(want) {
		t.Fatalf("FixedProviders() = %#v, want %#v", providers, want)
	}
	for index, provider := range want {
		if providers[index] != provider {
			t.Fatalf("FixedProviders()[%d] = %q, want %q", index, providers[index], provider)
		}
	}
	providers[0] = ProviderFeishu
	if FixedProviders()[0] != ProviderDiscord {
		t.Fatal("FixedProviders() did not return a defensive copy")
	}
}

func TestRequiresWebhookUpdateUsesStrictProviderValidation(t *testing.T) {
	if !RequiresWebhookUpdate(Destination{Provider: ProviderDiscord, Credential: "https://discord.example.test/api/webhooks/1/token"}) {
		t.Fatal("RequiresWebhookUpdate() = false, want true for legacy credential")
	}
	if RequiresWebhookUpdate(Destination{Provider: ProviderDiscord, Credential: "https://discord.com/api/webhooks/1/token"}) {
		t.Fatal("RequiresWebhookUpdate() = true, want false for compliant credential")
	}
	if !RequiresWebhookUpdate(Destination{Provider: ProviderFeishu, Credential: " https://open.feishu.cn/open-apis/bot/v2/hook/token"}) {
		t.Fatal("RequiresWebhookUpdate() = false, want true for non-normalized Feishu credential")
	}
}
