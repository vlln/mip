package probe

import "testing"

func TestParseBearerChallenge(t *testing.T) {
	got, ok := parseBearerChallenge(`Bearer realm="https://auth.example/token",service="registry.example",scope="repository:library/nginx:pull"`)
	if !ok {
		t.Fatal("expected bearer challenge")
	}

	if got["realm"] != "https://auth.example/token" {
		t.Fatalf("unexpected realm: %q", got["realm"])
	}
	if got["service"] != "registry.example" {
		t.Fatalf("unexpected service: %q", got["service"])
	}
	if got["scope"] != "repository:library/nginx:pull" {
		t.Fatalf("unexpected scope: %q", got["scope"])
	}
}

func TestExtractJSONToken(t *testing.T) {
	if got := extractJSONToken(`{"token":"abc"}`); got != "abc" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := extractJSONToken(`{"access_token":"xyz"}`); got != "xyz" {
		t.Fatalf("unexpected access token: %q", got)
	}
}

func TestSelectPlatformDigest(t *testing.T) {
	body := []byte(`{
	  "schemaVersion": 2,
	  "mediaType": "application/vnd.oci.image.index.v1+json",
	  "manifests": [
	    {
	      "mediaType": "application/vnd.oci.image.manifest.v1+json",
	      "digest": "sha256:amd64",
	      "platform": {"os": "linux", "architecture": "amd64"}
	    },
	    {
	      "mediaType": "application/vnd.oci.image.manifest.v1+json",
	      "digest": "sha256:arm64",
	      "platform": {"os": "linux", "architecture": "arm64"}
	    }
	  ]
	}`)

	got, ok := selectPlatformDigest(body, "linux/amd64")
	if !ok {
		t.Fatal("expected platform digest")
	}
	if got != "sha256:amd64" {
		t.Fatalf("digest = %q", got)
	}
}

func TestSelectPlatformDigestVariant(t *testing.T) {
	body := []byte(`{
	  "manifests": [
	    {"digest": "sha256:armv6", "platform": {"os": "linux", "architecture": "arm", "variant": "v6"}},
	    {"digest": "sha256:armv7", "platform": {"os": "linux", "architecture": "arm", "variant": "v7"}}
	  ]
	}`)

	got, ok := selectPlatformDigest(body, "linux/arm/v7")
	if !ok {
		t.Fatal("expected variant digest")
	}
	if got != "sha256:armv7" {
		t.Fatalf("digest = %q", got)
	}
}

func TestSelectPlatformDigestMissing(t *testing.T) {
	body := []byte(`{"manifests":[{"digest":"sha256:amd64","platform":{"os":"linux","architecture":"amd64"}}]}`)

	if got, ok := selectPlatformDigest(body, "linux/arm64"); ok {
		t.Fatalf("unexpected digest: %q", got)
	}
}

func TestParsePlatform(t *testing.T) {
	got, ok := parsePlatform("linux/x86_64")
	if !ok {
		t.Fatal("expected platform")
	}
	if got.OS != "linux" || got.Architecture != "amd64" {
		t.Fatalf("unexpected platform: %+v", got)
	}
}
