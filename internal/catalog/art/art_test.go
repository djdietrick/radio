package art

import (
	"io"
	"testing"
)

func TestSaveAndOpenRoundTrip(t *testing.T) {
	c := New(t.TempDir())
	data := []byte("fake-jpeg-bytes")

	saved, err := c.Save("album1", "image/jpeg", data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !saved {
		t.Fatal("expected first save to write the file")
	}
	if !c.Has("album1") {
		t.Fatal("Has should report true after save")
	}

	f, mime, size, err := c.Open("album1")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", mime)
	}
	if size != int64(len(data)) {
		t.Fatalf("size = %d, want %d", size, len(data))
	}
	got, _ := io.ReadAll(f)
	if string(got) != string(data) {
		t.Fatalf("content mismatch: %q", got)
	}
}

func TestFirstArtWins(t *testing.T) {
	c := New(t.TempDir())
	if _, err := c.Save("album1", "image/jpeg", []byte("first")); err != nil {
		t.Fatal(err)
	}
	saved, err := c.Save("album1", "image/png", []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	if saved {
		t.Fatal("second save should be a no-op (first art wins)")
	}
	// Original bytes (and jpeg extension/mime) must be preserved.
	f, mime, _, err := c.Open("album1")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg (original)", mime)
	}
	got, _ := io.ReadAll(f)
	if string(got) != "first" {
		t.Fatalf("content = %q, want \"first\"", got)
	}
}

func TestPngExtensionAndMime(t *testing.T) {
	c := New(t.TempDir())
	if _, err := c.Save("album2", "image/png", []byte("png")); err != nil {
		t.Fatal(err)
	}
	_, mime, _, err := c.Open("album2")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" {
		t.Fatalf("mime = %q, want image/png", mime)
	}
}

func TestOpenMissingReturnsNotFound(t *testing.T) {
	c := New(t.TempDir())
	if _, _, _, err := c.Open("nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSaveEmptyDataIsNoOp(t *testing.T) {
	c := New(t.TempDir())
	saved, err := c.Save("album3", "image/jpeg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if saved || c.Has("album3") {
		t.Fatal("empty data should not create a file")
	}
}
