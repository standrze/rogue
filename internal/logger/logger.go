package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
			logSize := min(len(bodyBytes), sl.maxBodySize)
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
			logSize := min(len(bodyBytes), sl.maxBodySize)
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
