package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func CallUploadDocumentAPI(file *multipart.FileHeader, userID string, token string, voice string, speakingRate float64) (map[string]interface{}, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Tạo tệp để upload
	fw, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	// Mở file và sao chép nội dung vào form
	fileContent, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer fileContent.Close()

	if _, err := io.Copy(fw, fileContent); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %v", err)
	}

	// Log dữ liệu
	fmt.Printf("Uploading file: %s (voice: %s, rate: %.2f)\n", file.Filename, voice, speakingRate)

	// Ghi các trường vào form
	if err := writer.WriteField("voice", voice); err != nil {
		return nil, fmt.Errorf("failed to write voice field: %v", err)
	}
	if err := writer.WriteField("speaking_rate", fmt.Sprintf("%.2f", speakingRate)); err != nil {
		return nil, fmt.Errorf("failed to write speaking_rate field: %v", err)
	}

	writer.Close()

	// ✅ FIX 1: Kiểm tra API_BASE_URL
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("API_BASE_URL is not set in environment")
	}

	url := fmt.Sprintf("%s/api/admin/documents/upload", baseURL)
	fmt.Printf("Calling API: %s\n", url)

	// Tạo yêu cầu HTTP
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("user_id", userID)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// ✅ FIX 2: Tăng timeout cho Railway (network có thể chậm hơn)
	client := &http.Client{
		Timeout: 120 * time.Second, // Tăng từ default 30s lên 120s
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// ✅ FIX 3: Log chi tiết response khi lỗi
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("ERROR Response (%d): %s\n", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("failed to upload file, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Đọc phản hồi
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Phân tích JSON phản hồi
	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	fmt.Println("✅ Upload successful")
	return result, nil
}

// GenerateSummary nhận nội dung text và trả về tóm tắt
func GenerateSummary(content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("nội dung rỗng")
	}

	if len(content) > 200 {
		return content[:200] + "...", nil
	}
	return content, nil
}

// package services

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"mime/multipart"
// 	"net/http"
// 	"os"
// )

// func CallUploadDocumentAPI(file *multipart.FileHeader, userID string, token string, voice string, speakingRate float64) (map[string]interface{}, error) {
// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)

// 	// Tạo tệp để upload
// 	fw, err := writer.CreateFormFile("file", file.Filename)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create form file: %v", err)
// 	}

// 	// Mở file và sao chép nội dung vào form
// 	fileContent, err := file.Open()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to open file: %v", err)
// 	}
// 	defer fileContent.Close()
// 	if _, err := io.Copy(fw, fileContent); err != nil {
// 		return nil, fmt.Errorf("failed to copy file content: %v", err)
// 	}

// 	// Log dữ liệu file, voice và speakingRate
// 	fmt.Println("Dữ liệu file: ", file.Filename)
// 	fmt.Println("Voice: ", voice)
// 	fmt.Println("Speaking Rate: ", speakingRate)

// 	// Ghi các trường vào form
// 	if err := writer.WriteField("voice", voice); err != nil {
// 		return nil, fmt.Errorf("failed to write voice field: %v", err)
// 	}
// 	if err := writer.WriteField("speaking_rate", fmt.Sprintf("%f", speakingRate)); err != nil {
// 		return nil, fmt.Errorf("failed to write speaking_rate field: %v", err)
// 	}

// 	writer.Close()

// 	// Lấy URL của API từ biến môi trường
// 	baseURL := os.Getenv("API_BASE_URL")
// 	if baseURL == "" {
// 		return nil, fmt.Errorf("API_BASE_URL is not set")
// 	}
// 	url := fmt.Sprintf("%s/api/admin/documents/upload", baseURL)

// 	// Tạo yêu cầu HTTP
// 	req, err := http.NewRequest("POST", url, body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %v", err)
// 	}

// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("user_id", userID)
// 	if token != "" {
// 		req.Header.Set("Authorization", "Bearer "+token)
// 	}

// 	// Gửi yêu cầu
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to send request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	// Kiểm tra mã trạng thái HTTP
// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("failed to upload file, status code: %d", resp.StatusCode)
// 	}

// 	// Đọc phản hồi
// 	respData, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read response body: %v", err)
// 	}

// 	// Phân tích JSON phản hồi
// 	var result map[string]interface{}
// 	if err := json.Unmarshal(respData, &result); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
// 	}

// 	return result, nil
// }

// // GenerateSummary nhận nội dung text và trả về tóm tắt
// func GenerateSummary(content string) (string, error) {
// 	if content == "" {
// 		return "", fmt.Errorf("nội dung rỗng")
// 	}

// 	// TODO: Thay thế bằng logic tóm tắt thực tế (OpenAI, GPT, hoặc thuật toán tự viết)
// 	if len(content) > 200 {
// 		return content[:200] + "...", nil
// 	}
// 	return content, nil
// }
