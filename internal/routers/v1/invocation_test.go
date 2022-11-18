package v1

import (
	"testing"
)

func TestPaorseHost(t *testing.T) {
	cases := []struct {
		Name           string
		A, B, Expected string
	}{
		{"case1", "src-ns", "demo", "src-ns/demo.src-ns.svc.cluster.local"},
		{"case2", "src-ns", "demo.svc.cluster.local", "src-ns/demo.src-ns.svc.cluster.local"},
		{"case3", "src-ns", "demo.default.svc.cluster.local", "default/demo.default.svc.cluster.local"},
		{"case4", "src-ns", "demo:80", "src-ns/demo.src-ns.svc.cluster.local"},
		{"case5", "src-ns", "demo.default:80", "default/demo.default.svc.cluster.local"},
		{"case6", "src-ns", "demo.default.svc.cluster.local:80", "default/demo.default.svc.cluster.local"},
		{"case7", "src-ns", "demo/ping", "src-ns/demo.src-ns.svc.cluster.local"},
		{"case8", "src-ns", "demo.default.svc.cluster.local/ping", "default/demo.default.svc.cluster.local"},
		{"case9", "src-ns", "demo:80/ping", "src-ns/demo.src-ns.svc.cluster.local"},
		{"case10", "src-ns", "demo.default.svc.cluster.local:80/ping", "default/demo.default.svc.cluster.local"},
		{"case11", "src-ns", ".demo.default.svc.cluster.local:80/ping", "default/demo.default.svc.cluster.local"},
		{"case12", "src-ns", "demo..default.svc.cluster.local:80/ping", "default/demo..default.svc.cluster.local"},
		{"case13", "src-ns", "demo./ping", "src-ns/demo.src-ns.svc.cluster.local"},
	}

	invocation := NewInvocation(nil)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			host, _ := invocation.ParseHost(c.A, c.B)
			if host != c.Expected {
				t.Errorf("src namespace: %s , destination: %s , expected: %s, but %s got", c.A, c.B, c.Expected, host)
			}
		})
	}
}
