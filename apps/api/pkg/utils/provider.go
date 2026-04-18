package utils

import "strings"

// providerPatterns maps a provider key to the substrings that identify it
// within a URL. Patterns are checked in iteration order; first match wins
// per URL. The "other" bucket captures anything that matches no pattern.
var providerPatterns = []struct {
	key      string
	patterns []string
}{
	{"baidu", []string{"pan.baidu.com", "tieba.baidu.com", "pan.baidu.", "baidu.com"}},
	{"aliyun", []string{"alipan.com", "aliyun", "aliyundrive", "aliyuncs", "aliyunpan"}},
	{"quark", []string{"pan.quark.cn", "quark.cn", "quark"}},
	{"pan123", []string{
		"123pan", "123684", "123865", "123912", "123912.com",
		"123684.com", "123865.com", "123pan.cn", "vip.123pan",
	}},
	{"tianyiyun", []string{"cloud.189.cn", "189.cn", "ecloud.189.cn"}},
	{"caiyun", []string{"caiyun.139.com", "yun.139.com", "139.com"}},
	{"xunlei", []string{"pan.xunlei.com", "xunlei.com"}},
	{"uc", []string{"drive.uc.cn", "uc.cn"}},
	{"lanzou", []string{
		"lanzou.com", "lanzous.com", "lanzoux.com", "lanzoui.com",
		"lanzouw.com", "lanzouj.com", "lanzouu.com", "lanzouq.com",
	}},
}

// DetectProviderFromURL classifies a single URL into one of the known
// provider keys, falling back to "other".
func DetectProviderFromURL(url string) string {
	if url == "" {
		return "other"
	}
	s := strings.ToLower(url)
	for _, entry := range providerPatterns {
		for _, p := range entry.patterns {
			if strings.Contains(s, p) {
				return entry.key
			}
		}
	}
	return "other"
}

// DetectProvidersFromURLs returns the deduped set of providers spanning all
// URLs. Preserves the classifier order for stable output.
func DetectProvidersFromURLs(urls []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		p := DetectProviderFromURL(u)
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
