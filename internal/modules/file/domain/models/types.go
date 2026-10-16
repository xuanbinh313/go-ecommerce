package models

type UploadInput struct {
	Filename string
	MIMEType string
	Content  []byte
	Access   string
}
type PresignedInput struct {
	Filename     string `json:"filename"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	Access       string `json:"access"`
	OriginalName string `json:"originalname"`
}
type File struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type PresignedOutput struct {
	URL          string `json:"url"`
	Filename     string `json:"filename"`
	MIMEType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	Extension    string `json:"extension"`
	OriginalName string `json:"originalname"`
}
