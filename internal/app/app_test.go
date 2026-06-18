package app

import "testing"

func TestStaticUIRouteServesNestedAssets(t *testing.T) {
	t.Parallel()

	route := staticUIRoute()
	if route.Path != "/" {
		t.Fatalf("expected static UI route to match nested assets with /, got %q", route.Path)
	}
}
