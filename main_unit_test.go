package main

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func TestMainServesProvider(t *testing.T) {
	originalServe, originalFatal := serveProvider, fatal
	t.Cleanup(func() {
		serveProvider, fatal = originalServe, originalFatal
	})
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			want := errors.New("serve failed")
			called, reported := false, false
			serveProvider = func(_ context.Context, factory func() provider.Provider, opts providerserver.ServeOpts) error {
				called = true
				if factory() == nil || opts.Address != "registry.terraform.io/geo-gaussian/pfsense-nexus" {
					t.Errorf("unexpected provider server configuration: %+v", opts)
				}
				if fail {
					return want
				}
				return nil
			}
			fatal = func(args ...interface{}) {
				if len(args) == 1 {
					err, ok := args[0].(error)
					reported = ok && errors.Is(err, want)
				}
			}
			main()
			if !called || reported != fail {
				t.Fatalf("served = %v, failure reported = %v", called, reported)
			}
		})
	}
}
