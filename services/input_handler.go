package services

import "errors"

// Hàm ánh xạ phần mở rộng file sang InputType
func GetInputTypeFromExt(ext string) (InputType, error) {
	switch ext {
	case ".pdf":
		return InputPDF, nil
	case ".docx":
		return InputDOCX, nil
	case ".txt":
		return InputTXT, nil
	default:
		return "", errors.New("định dạng file không hỗ trợ")
	}
}
