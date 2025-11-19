package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RequestLog struct {
	Timestamp time.Time         `json:"timestamp"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	RequestID string            `json:"request_id"`
}

type ResponseLog struct {
	Timestamp  time.Time         `json:"timestamp"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	RequestID  string            `json:"request_id"`
}

type SessionLogger struct {
	sessionFile *os.File
	sessionName string
	sessionDir  string
	logHeaders  bool
	logBody     bool
	maxBodySize int
	encoder     *json.Encoder
	firstEntry  bool
}

func NewSessionLogger(sessionDir string, logHeaders, logBody bool, maxBodySize int) (*SessionLogger, error) {
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, err
	}

	sessionName := fmt.Sprintf("session_%s.json", time.Now().Format("20060102_150405"))
	sessionPath := filepath.Join(sessionDir, sessionName)

	file, err := os.Create(sessionPath)
	if err != nil {
		return nil, err
	}

	// Start the JSON array
	if _, err := file.WriteString("[\n"); err != nil {
		file.Close()
		return nil, err
	}

	sl := &SessionLogger{
		sessionFile: file,
		sessionName: sessionName,
		sessionDir:  sessionDir,
		logHeaders:  logHeaders,
		logBody:     logBody,
		maxBodySize: maxBodySize,
		encoder:     json.NewEncoder(file),
		firstEntry:  true,
	}

	sl.encoder.SetIndent("", "  ")

	return sl, nil
}

func (sl *SessionLogger) LogRequest(req *http.Request, requestID string) error {
	reqLog := RequestLog{
		Timestamp: time.Now(),
		Method:    req.Method,
		URL:       req.URL.String(),
		RequestID: requestID,
	}

	if sl.logHeaders && req.Header != nil {
		reqLog.Headers = make(map[string]string)
		for k, v := range req.Header {
			if len(v) > 0 {
				reqLog.Headers[k] = v[0]
			}
		}
	}

	if sl.logBody && req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			logSize := len(bodyBytes)
			if logSize > sl.maxBodySize {
				logSize = sl.maxBodySize
			}
			reqLog.Body = string(bodyBytes[:logSize])
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	if !sl.firstEntry {
		if _, err := sl.sessionFile.WriteString(",\n"); err != nil {
			return err
		}
	}
	sl.firstEntry = false

	return sl.encoder.Encode(map[string]any{
		"type": "request",
		"data": reqLog,
	})
}

func (sl *SessionLogger) LogResponse(resp *http.Response, requestID string) error {
	respLog := ResponseLog{
		Timestamp:  time.Now(),
		StatusCode: resp.StatusCode,
		RequestID:  requestID,
	}

	if sl.logHeaders && resp.Header != nil {
		respLog.Headers = make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				respLog.Headers[k] = v[0]
			}
		}
	}

	if sl.logBody && resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			logSize := len(bodyBytes)
			if logSize > sl.maxBodySize {
				logSize = sl.maxBodySize
			}
			respLog.Body = string(bodyBytes[:logSize])
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	if !sl.firstEntry {
		if _, err := sl.sessionFile.WriteString(",\n"); err != nil {
			return err
		}
	}
	sl.firstEntry = false

	return sl.encoder.Encode(map[string]any{
		"type": "response",
		"data": respLog,
	})
}

func (sl *SessionLogger) Close() error {
	// End the JSON array
	if _, err := sl.sessionFile.WriteString("\n]"); err != nil {
		sl.sessionFile.Close()
		return err
	}
	return sl.sessionFile.Close()
}

func (sl *SessionLogger) GetSessionName() string {
	return sl.sessionName
}

func ListSessions(sessionDir string) ([]string, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			sessions = append(sessions, entry.Name())
		}
	}

	return sessions, nil
}

func LoadSession(sessionDir, sessionName string) ([]byte, error) {
	sessionPath := filepath.Join(sessionDir, sessionName)
	return os.ReadFile(sessionPath)
}

func ExportSessionToMarkdown(sessionDir, sessionName, outputPath string) error {
	data, err := LoadSession(sessionDir, sessionName)
	if err != nil {
		return err
	}

	var entries []map[string]any
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) > 0 && trimmedData[0] == '[' {
		if err := json.Unmarshal(data, &entries); err != nil {
			return err
		}
	} else {
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var entry map[string]any
			if err := decoder.Decode(&entry); err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
	}

	var markdown strings.Builder
	markdown.WriteString(fmt.Sprintf("# Session Log: %s\n\n", sessionName))

	for _, entry := range entries {
		typeStr, ok := entry["type"].(string)
		if !ok {
			continue
		}

		dataMap, ok := entry["data"].(map[string]any)
		if !ok {
			continue
		}

		timestampStr, _ := dataMap["timestamp"].(string)
		requestID, _ := dataMap["request_id"].(string)

		switch typeStr {
		case "request":
			method, _ := dataMap["method"].(string)
			url, _ := dataMap["url"].(string)
			markdown.WriteString(fmt.Sprintf("## Request %s\n", requestID))
			markdown.WriteString(fmt.Sprintf("**Time:** %s\n\n", timestampStr))
			markdown.WriteString(fmt.Sprintf("`%s %s`\n\n", method, url))

			if headers, ok := dataMap["headers"].(map[string]any); ok && len(headers) > 0 {
				markdown.WriteString("### Headers\n")
				markdown.WriteString("| Key | Value |\n| --- | --- |\n")
				for k, v := range headers {
					markdown.WriteString(fmt.Sprintf("| %s | %s |\n", k, v))
				}
				markdown.WriteString("\n")
			}

			if body, ok := dataMap["body"].(string); ok && body != "" {
				markdown.WriteString("### Body\n")
				markdown.WriteString("```\n")
				markdown.WriteString(body)
				markdown.WriteString("\n```\n\n")
			}

		case "response":
			statusCode, _ := dataMap["status_code"].(float64)
			markdown.WriteString(fmt.Sprintf("## Response %s\n", requestID))
			markdown.WriteString(fmt.Sprintf("**Time:** %s\n\n", timestampStr))
			markdown.WriteString(fmt.Sprintf("**Status:** %d\n\n", int(statusCode)))

			if headers, ok := dataMap["headers"].(map[string]any); ok && len(headers) > 0 {
				markdown.WriteString("### Headers\n")
				markdown.WriteString("| Key | Value |\n| --- | --- |\n")
				for k, v := range headers {
					markdown.WriteString(fmt.Sprintf("| %s | %s |\n", k, v))
				}
				markdown.WriteString("\n")
			}

			if body, ok := dataMap["body"].(string); ok && body != "" {
				markdown.WriteString("### Body\n")
				markdown.WriteString("```\n")
				markdown.WriteString(body)
				markdown.WriteString("\n```\n\n")
			}
		}
		markdown.WriteString("---\n\n")
	}

	return os.WriteFile(outputPath, []byte(markdown.String()), 0644)
}
