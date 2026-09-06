package model

type SignResponse struct {
	CloudName string `json:"cloudName"`
	APIKey    string `json:"apiKey"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
	UploadURL string `json:"uploadUrl"`
	Folder    string `json:"folder,omitempty"`
}
