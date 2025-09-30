package services

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"mime/multipart"
	"os"
	"strings"
)

func ExtractTextFromDOCX(fileHeader *multipart.FileHeader) (string, error) {
	// Tạo file tạm
	tmpFile, err := os.CreateTemp("", "upload-*.docx")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Lưu nội dung file vào file tạm
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	if _, err := io.Copy(tmpFile, src); err != nil {
		return "", err
	}

	// Mở file zip (.docx là file zip!)
	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Tìm file document.xml
	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", err
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	// Đọc XML & trích xuất <w:t> tag (văn bản)
	var buf bytes.Buffer
	decoder := xml.NewDecoder(rc)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "t" { // <w:t>
				var text string
				if err := decoder.DecodeElement(&text, &se); err == nil {
					buf.WriteString(text + " ")
				}
			}
		}
	}

	return strings.TrimSpace(buf.String()), nil
}
