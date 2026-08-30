package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginRequest(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		referer string
		want   bool
	}{
		{name: "api client without origin", want: true},
		{name: "same origin", origin: "https://example.test", want: true},
		{name: "cross origin", origin: "https://evil.test", want: false},
		{name: "referer fallback", referer: "https://evil.test/page", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("DELETE", "https://example.test/api/v1/stats/clear", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				r.Header.Set("Referer", tc.referer)
			}
			if got := sameOriginRequest(r); got != tc.want {
				t.Fatalf("sameOriginRequest()=%v, want %v", got, tc.want)
			}
		})
	}
}
