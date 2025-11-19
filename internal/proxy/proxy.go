package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/martian/v3"
	"github.com/google/martian/v3/fifo"
	"github.com/google/martian/v3/mitm"
	"github.com/standrze/rogue/internal/cert"
	"github.com/standrze/rogue/internal/logger"
)

type Proxy struct {
	Port         int
	Host         string
	CertPath     string
	KeyPath      string
	SessionDir   string
	LogRequests  bool
	LogResponses bool
	LogHeaders   bool
	LogBody      bool
	MaxBodySize  int
}

type ProxyOption func(p *Proxy)

func WithPort(port int) ProxyOption {
	return func(p *Proxy) {
		p.Port = port
	}
}

func WithHost(host string) ProxyOption {
	return func(p *Proxy) {
		p.Host = host
	}
}

func WithCert(certPath, keyPath string) ProxyOption {
	return func(p *Proxy) {
		p.CertPath = certPath
		p.KeyPath = keyPath
	}
}

func WithSessionDir(dir string) ProxyOption {
	return func(p *Proxy) {
		p.SessionDir = dir
	}
}

func WithLogging(logRequests, logResponses, logHeaders, logBody bool, maxBodySize int) ProxyOption {
	return func(p *Proxy) {
		p.LogRequests = logRequests
		p.LogResponses = logResponses
		p.LogHeaders = logHeaders
		p.LogBody = logBody
		p.MaxBodySize = maxBodySize
	}
}

type RequestModifier struct {
	Logger *logger.SessionLogger
}

func (r *RequestModifier) ModifyRequest(req *http.Request) error {
	reqID := fmt.Sprintf("%d", time.Now().UnixNano())
	req.Header.Set("X-Rogue-Request-ID", reqID)
	return r.Logger.LogRequest(req, reqID)
}

type ResponseModifier struct {
	Logger *logger.SessionLogger
}

func (r *ResponseModifier) ModifyResponse(res *http.Response) error {
	reqID := res.Request.Header.Get("X-Rogue-Request-ID")
	return r.Logger.LogResponse(res, reqID)
}

func NewProxyServer(option ...ProxyOption) *martian.Proxy {
	proxyOpts := &Proxy{
		Port:         8080,
		Host:         "0.0.0.0",
		CertPath:     "certs/ca.crt",
		KeyPath:      "certs/ca.key",
		SessionDir:   "logs",
		LogRequests:  true,
		LogResponses: true,
		LogHeaders:   true,
		LogBody:      true,
		MaxBodySize:  1024 * 1024,
	}

	for _, opt := range option {
		opt(proxyOpts)
	}

	if !cert.Exists(proxyOpts.CertPath, proxyOpts.KeyPath) {
		if err := cert.GenerateSelfSigned("Rogue Proxy", "Rogue CA", 365, proxyOpts.CertPath, proxyOpts.KeyPath); err != nil {
			panic(fmt.Sprintf("failed to generate certs: %v", err))
		}
	}

	ca, priv, err := cert.Load(proxyOpts.CertPath, proxyOpts.KeyPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load certs: %v", err))
	}

	mc, err := mitm.NewConfig(ca, priv)
	if err != nil {
		panic(fmt.Sprintf("failed to create MITM config: %v", err))
	}

	// Trust the CA in the proxy's TLS config so it can verify itself if needed,
	// though mostly this is for the client to trust.
	// Martian handles the MITM logic.

	// Create proxy
	p := martian.NewProxy()
	p.SetMITM(mc)

	// Logger
	sl, err := logger.NewSessionLogger(proxyOpts.SessionDir, proxyOpts.LogHeaders, proxyOpts.LogBody, proxyOpts.MaxBodySize)
	if err != nil {
		panic(fmt.Sprintf("failed to create session logger: %v", err))
	}

	// Modifiers
	fg := fifo.NewGroup()

	if proxyOpts.LogRequests {
		reqMod := &RequestModifier{Logger: sl}
		fg.AddRequestModifier(reqMod)
	}

	if proxyOpts.LogResponses {
		respMod := &ResponseModifier{Logger: sl}
		fg.AddResponseModifier(respMod)
	}

	p.SetRequestModifier(fg)
	p.SetResponseModifier(fg)

	return p
}
