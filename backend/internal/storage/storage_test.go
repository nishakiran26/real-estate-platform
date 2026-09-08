package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestValidateUploadAcceptsAllowedImages(t *testing.T) {
	for mime, wantExt := range AllowedImageTypes {
		ext, err := ValidateUpload(mime, 1024, MaxImageSize, AllowedImageTypes)
		if err != nil {
			t.Errorf("%s was rejected: %v", mime, err)
		}
		if ext != wantExt {
			t.Errorf("%s extension = %q, want %q", mime, ext, wantExt)
		}
	}
}

func TestValidateUploadRejectsDisallowedTypes(t *testing.T) {
	// A PDF is a valid document but must never enter the image gallery, and an
	// executable must never be accepted anywhere.
	rejected := []string{"application/pdf", "text/html", "application/x-msdownload", "image/svg+xml"}
	for _, mime := range rejected {
		if _, err := ValidateUpload(mime, 1024, MaxImageSize, AllowedImageTypes); !errors.Is(err, ErrTypeNotAllowed) {
			t.Errorf("%s: error = %v, want ErrTypeNotAllowed", mime, err)
		}
	}
}

func TestValidateUploadEnforcesSizeLimit(t *testing.T) {
	if _, err := ValidateUpload("image/png", MaxImageSize+1, MaxImageSize, AllowedImageTypes); !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("error = %v, want ErrFileTooLarge", err)
	}
	if _, err := ValidateUpload("image/png", 0, MaxImageSize, AllowedImageTypes); err == nil {
		t.Error("an empty file was accepted")
	}
}

func TestValidateUploadIgnoresCharsetParameters(t *testing.T) {
	// Browsers send parameters on the content type; matching must not break.
	if _, err := ValidateUpload("image/jpeg; charset=binary", 100, MaxImageSize, AllowedImageTypes); err != nil {
		t.Errorf("a parameterised content type was rejected: %v", err)
	}
}

func TestBuildKeyIsUniqueAndSanitised(t *testing.T) {
	first := BuildKey("properties/1/images", "My Photo (Front).jpg", ".jpg")
	second := BuildKey("properties/1/images", "My Photo (Front).jpg", ".jpg")

	if first == second {
		t.Error("two uploads of the same filename produced the same key")
	}
	if !strings.HasPrefix(first, "properties/1/images/") {
		t.Errorf("key %q is not inside the requested folder", first)
	}
	if !strings.HasSuffix(first, ".jpg") {
		t.Errorf("key %q lost its extension", first)
	}
	if strings.ContainsAny(first, " ()") {
		t.Errorf("key %q was not sanitised", first)
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir(), "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	// A key that climbs out of the upload directory must not resolve.
	if _, err := store.Open(context.Background(), "../../../etc/passwd"); err == nil {
		t.Error("a traversal key was allowed to open a file outside the storage root")
	}
}

func TestLocalStorageSaveAndOpenRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := NewLocal(t.TempDir(), "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	content := "brochure-bytes"
	obj, err := store.Save(ctx, "properties/9/documents", "brochure.pdf",
		strings.NewReader(content), int64(len(content)), "application/pdf")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if obj.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", obj.Size, len(content))
	}
	if !strings.HasPrefix(obj.URL, "http://localhost:8080/uploads/") {
		t.Errorf("URL = %q, which is not under the public base URL", obj.URL)
	}

	reader, err := store.Open(ctx, obj.Key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, readErr := io.ReadAll(reader)
	// Close before deleting: Windows refuses to remove a file that is still open.
	reader.Close()

	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != content {
		t.Errorf("read %q, want %q", got, content)
	}

	if err := store.Delete(ctx, obj.Key); err != nil {
		t.Errorf("Delete: %v", err)
	}
	// Deleting a key that is already gone is not an error.
	if err := store.Delete(ctx, obj.Key); err != nil {
		t.Errorf("second Delete: %v", err)
	}
}
