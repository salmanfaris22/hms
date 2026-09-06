package service

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/salman/hms-backend/internal/modules/upload/model"
)

var ErrNotConfigured = errors.New("cloudinary not configured")

type Service struct {
	CloudName string
	APIKey    string
	APISecret string
}

func New(cloudName, apiKey, apiSecret string) *Service {
	return &Service{CloudName: cloudName, APIKey: apiKey, APISecret: apiSecret}
}

func (s *Service) Sign(folder string) (model.SignResponse, error) {
	if s.CloudName == "" || s.APIKey == "" || s.APISecret == "" {
		return model.SignResponse{}, ErrNotConfigured
	}
	params := map[string]string{"timestamp": fmt.Sprintf("%d", time.Now().Unix())}
	if f := strings.TrimSpace(folder); f != "" {
		params["folder"] = f
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params[k])
	}
	sb.WriteString(s.APISecret)

	sum := sha1.Sum([]byte(sb.String()))
	resp := model.SignResponse{
		CloudName: s.CloudName,
		APIKey:    s.APIKey,
		Timestamp: params["timestamp"],
		Signature: hex.EncodeToString(sum[:]),
		UploadURL: fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/auto/upload", s.CloudName),
	}
	if f, ok := params["folder"]; ok {
		resp.Folder = f
	}
	return resp, nil
}
