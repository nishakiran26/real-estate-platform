package handlers

import (
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rera/property-platform/backend/internal/services"
)

// maxMultipartMemory caps how much of an upload is buffered in RAM before Echo
// spills it to a temporary file.
const maxMultipartMemory = 8 << 20 // 8 MB

// openUpload reads a multipart file field and sniffs its real content type.
// The browser-supplied Content-Type is ignored: a client can set it to anything,
// so the type is detected from the first bytes of the file instead.
// The caller must close the returned file.
func openUpload(c echo.Context, field string) (services.Upload, multipart.File, error) {
	header, err := c.FormFile(field)
	if err != nil {
		return services.Upload{}, nil, services.Invalid("No file was uploaded")
	}
	file, err := header.Open()
	if err != nil {
		return services.Upload{}, nil, services.Invalid("The uploaded file could not be read")
	}

	sniff := make([]byte, 512)
	n, _ := file.Read(sniff)
	if _, err := file.Seek(0, 0); err != nil {
		file.Close()
		return services.Upload{}, nil, services.Invalid("The uploaded file could not be read")
	}

	return services.Upload{
		Filename: header.Filename,
		MimeType: http.DetectContentType(sniff[:n]),
		Size:     header.Size,
		Reader:   file,
	}, file, nil
}
