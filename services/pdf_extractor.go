package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/ledongthuc/pdf"
)

func ExtractTextFromPDF(file multipart.File) (string, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return "", fmt.Errorf("lỗi đọc file PDF: %w", err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return "", fmt.Errorf("không thể tạo reader PDF: %w", err)
	}

	var textBuilder bytes.Buffer
	pages := reader.NumPage()
	for i := 1; i <= pages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		textBuilder.WriteString(content)
	}

	return textBuilder.String(), nil
}
