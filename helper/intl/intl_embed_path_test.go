package intl

import "testing"

func TestReadEmbedResourceCanonicalPath(t *testing.T) {
	requireCommunityFixture(t, "community/resources/bof/ipconfig/ipconfig.x64.o")
	path := "embed://community/community/resources/bof/ipconfig/ipconfig.x64.o"
	content, err := ReadEmbedResource(path)
	if err != nil {
		t.Fatalf("canonical path should be readable: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("canonical path returned empty content")
	}
}

func TestReadEmbedResourceLegacyPathFallback(t *testing.T) {
	requireCommunityFixture(t, "community/resources/bof/ipconfig/ipconfig.x64.o")
	path := "embed://community/resources/bof/ipconfig/ipconfig.x64.o"
	content, err := ReadEmbedResource(path)
	if err != nil {
		t.Fatalf("legacy path should fallback correctly: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("legacy fallback returned empty content")
	}
}

func TestReadEmbedResourceInvalidPath(t *testing.T) {
	_, err := ReadEmbedResource("embed://invalid")
	if err == nil {
		t.Fatal("expected parse error for invalid embed path")
	}
}
