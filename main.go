package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultPort              = "5464"
	defaultLogPath           = "/tmp/requestHeadersQueryParamsAndBody.log"
	defaultShutdownTimeout   = 10 * time.Second
	curlConfigPath           = "/_config/curl"
	logLevelConfigPath       = "/_config/log-level"
	requestDebuggerURLHeader = "Requestdebugger_url"
	requestDebuggerURLEnvVar = "REQUESTDEBUGGER_URL"
	logFormatEnvVar          = "LOG_FORMAT"
	enableCurlEnvVar         = "ENABLE_CURL"
)

type logLevel int

const (
	logLevelError logLevel = iota
	logLevelDebug
	logLevelTrace
)

func (l logLevel) String() string {
	switch l {
	case logLevelError:
		return "error"
	case logLevelDebug:
		return "debug"
	case logLevelTrace:
		return "trace"
	default:
		return "unknown"
	}
}

type logFormat int

const (
	logFormatText logFormat = iota
	logFormatJSON
)

func (f logFormat) String() string {
	switch f {
	case logFormatText:
		return "text"
	case logFormatJSON:
		return "json"
	default:
		return "unknown"
	}
}

func parseLogLevel(raw string) (logLevel, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "error":
		return logLevelError, nil
	case "debug":
		return logLevelDebug, nil
	case "trace":
		return logLevelTrace, nil
	default:
		return logLevelDebug, fmt.Errorf("invalid log level %q (expected error, debug, or trace)", raw)
	}
}

func parseLogFormat(raw string) (logFormat, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "text", "plain":
		return logFormatText, nil
	case "json":
		return logFormatJSON, nil
	default:
		return logFormatText, fmt.Errorf("invalid log format %q (expected text or json)", raw)
	}
}

func resolveLogLevel(flagValue, envValue, fallback string) (logLevel, error) {
	raw := strings.TrimSpace(flagValue)
	if raw == "" {
		raw = strings.TrimSpace(envValue)
	}
	if raw == "" {
		raw = fallback
	}
	return parseLogLevel(raw)
}

func resolveLogFormat(flagValue, envValue, fallback string) (logFormat, error) {
	raw := strings.TrimSpace(flagValue)
	if raw == "" {
		raw = strings.TrimSpace(envValue)
	}
	if raw == "" {
		raw = fallback
	}
	return parseLogFormat(raw)
}

func parseBoolEnv(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func resolveCurlEnabled(flagEnabled bool, envValue string) bool {
	if flagEnabled {
		return true
	}
	if enabled, ok := parseBoolEnv(envValue); ok {
		return enabled
	}
	return false
}

type runtimeConfig struct {
	mu          sync.RWMutex
	curlEnabled bool
	level       logLevel
}

func (c *runtimeConfig) curlEnabledValue() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.curlEnabled
}

func (c *runtimeConfig) setCurlEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.curlEnabled = enabled
}

func (c *runtimeConfig) logLevelValue() logLevel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.level
}

func (c *runtimeConfig) setLogLevel(level logLevel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.level = level
}

func (c *runtimeConfig) logError(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}

func (c *runtimeConfig) logDebug(format string, args ...any) {
	if c.logLevelValue() >= logLevelDebug {
		log.Printf("[DEBUG] "+format, args...)
	}
}

type curlConfigRequest struct {
	Enabled bool `json:"enabled"`
}

type curlConfigResponse struct {
	CurlEnabled bool `json:"curl_enabled"`
}

type logLevelConfigRequest struct {
	Level string `json:"level"`
}

type logLevelConfigResponse struct {
	Level string `json:"level"`
}

type appConfig struct {
	logPath            string
	logFormat          logFormat
	requestDebuggerURL string
	runtime            *runtimeConfig
}

type exchangeRecord struct {
	Timestamp   string              `json:"timestamp"`
	LogLevel    string              `json:"log_level"`
	Request     exchangeRequest     `json:"request"`
	Response    exchangeResponse    `json:"response"`
	CurlCommand string              `json:"curl_command,omitempty"`
}

type exchangeRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Body        string              `json:"body"`
	QueryParams map[string][]string `json:"query_params,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
}

type exchangeResponse struct {
	Status  int                 `json:"status"`
	Body    string              `json:"body"`
	Headers map[string][]string `json:"headers,omitempty"`
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK)
	}
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}

func (rec *responseRecorder) WriteHeader(statusCode int) {
	if rec.wroteHeader {
		return
	}
	rec.wroteHeader = true
	rec.status = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func main() {
	enableCurl := flag.Bool("curl", false, "generate curl commands in request logs")
	logLevelFlag := flag.String("log-level", "", "log level: error, debug, or trace")
	logFormatFlag := flag.String("log-format", "", "exchange log format: text or json")
	flag.Parse()

	level, err := resolveLogLevel(*logLevelFlag, os.Getenv("LOG_LEVEL"), "debug")
	if err != nil {
		log.Fatalf("invalid log level: %v", err)
	}

	format, err := resolveLogFormat(*logFormatFlag, os.Getenv(logFormatEnvVar), "text")
	if err != nil {
		log.Fatalf("invalid log format: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = defaultLogPath
	}

	cfg := &runtimeConfig{
		curlEnabled: resolveCurlEnabled(*enableCurl, os.Getenv(enableCurlEnvVar)),
		level:       level,
	}

	app := &appConfig{
		logPath:            logPath,
		logFormat:          format,
		requestDebuggerURL: strings.TrimSpace(os.Getenv(requestDebuggerURLEnvVar)),
		runtime:            cfg,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(curlConfigPath, cfg.serveCurlConfig)
	mux.HandleFunc(logLevelConfigPath, cfg.serveLogLevelConfig)
	mux.HandleFunc("/", newDebugHandler(cfg))

	addr := ":" + port
	server := &http.Server{
		Addr:    addr,
		Handler: withExchangeLogging(app, mux),
	}

	go func() {
		log.Printf("starting server on %s (log level: %s, log format: %s, curl generation: %v, requestdebugger_url: %q)", addr, cfg.logLevelValue(), app.logFormat, cfg.curlEnabledValue(), app.requestDebuggerURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.logError("server failed: %v", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sig := <-stop
	cfg.logDebug("received %s, shutting down gracefully", sig)

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		cfg.logError("graceful shutdown failed: %v", err)
		os.Exit(1)
	}

	cfg.logDebug("server stopped")
}

func withExchangeLogging(app *appConfig, next http.Handler) http.Handler {
	cfg := app.runtime

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAt := time.Now().UTC()

		reqBody, err := readRequestBody(r)
		if err != nil {
			cfg.logError("read request body: %v", err)
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(reqBody))

		curlHost := applyRequestDebuggerURL(r, app.requestDebuggerURL)

		rec := newResponseRecorder(w)
		next.ServeHTTP(rec, r)

		level := cfg.logLevelValue()
		if !shouldLogExchange(level, rec.status) {
			return
		}

		includeCurl := cfg.curlEnabledValue() && !strings.HasPrefix(r.URL.Path, "/_config/")
		detailLevel := exchangeDetailLevel(level, rec.status)
		record := buildExchangeRecord(receivedAt, r, reqBody, rec.status, rec.ResponseWriter.Header(), rec.body.Bytes(), includeCurl, detailLevel, curlHost)
		entry, err := formatExchangeLog(record, app.logFormat)
		if err != nil {
			cfg.logError("format exchange log: %v", err)
			return
		}

		if err := app.writeExchangeLog(entry); err != nil {
			cfg.logError("write exchange log: %v", err)
		}
	})
}

func (app *appConfig) writeExchangeLog(content string) error {
	if _, err := fmt.Fprint(os.Stdout, content); err != nil {
		return fmt.Errorf("write stdout: %w", err)
	}
	return appendToFile(app.logPath, content)
}

func applyRequestDebuggerURL(r *http.Request, envURL string) string {
	if envURL != "" {
		r.Header.Set(requestDebuggerURLHeader, envURL)
		return envURL
	}

	if headerURL := strings.TrimSpace(r.Header.Get(requestDebuggerURLHeader)); headerURL != "" {
		return headerURL
	}

	return "{{host}}"
}

func shouldLogExchange(level logLevel, status int) bool {
	switch level {
	case logLevelError:
		return status >= http.StatusBadRequest
	case logLevelDebug, logLevelTrace:
		return true
	default:
		return false
	}
}

func exchangeDetailLevel(configured logLevel, status int) logLevel {
	if configured == logLevelError && status >= http.StatusBadRequest {
		return logLevelTrace
	}
	return configured
}

func (cfg *runtimeConfig) serveCurlConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, curlConfigResponse{CurlEnabled: cfg.curlEnabledValue()})
	case http.MethodPut, http.MethodPost:
		var req curlConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body; expected {\"enabled\": true|false}", http.StatusBadRequest)
			return
		}
		cfg.setCurlEnabled(req.Enabled)
		cfg.logDebug("curl generation set to %v", req.Enabled)
		writeJSON(w, http.StatusOK, curlConfigResponse{CurlEnabled: cfg.curlEnabledValue()})
	default:
		w.Header().Set("Allow", "GET, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (cfg *runtimeConfig) serveLogLevelConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, logLevelConfigResponse{Level: cfg.logLevelValue().String()})
	case http.MethodPut, http.MethodPost:
		var req logLevelConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body; expected {\"level\": \"error\"|\"debug\"|\"trace\"}", http.StatusBadRequest)
			return
		}
		level, err := parseLogLevel(req.Level)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg.setLogLevel(level)
		cfg.logDebug("log level set to %s", level)
		writeJSON(w, http.StatusOK, logLevelConfigResponse{Level: cfg.logLevelValue().String()})
	default:
		w.Header().Set("Allow", "GET, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ERROR] write JSON response: %v", err)
	}
}

func newDebugHandler(cfg *runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, r.Body); err != nil {
			cfg.logError("write response: %v", err)
		}
	}
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func buildExchangeRecord(receivedAt time.Time, r *http.Request, reqBody []byte, status int, respHeader http.Header, respBody []byte, includeCurl bool, level logLevel, curlHost string) exchangeRecord {
	record := exchangeRecord{
		Timestamp: receivedAt.Format(time.RFC3339Nano),
		LogLevel:  level.String(),
		Request: exchangeRequest{
			Method: r.Method,
			URL:    r.URL.Path,
		},
		Response: exchangeResponse{
			Status: status,
		},
	}

	if level >= logLevelDebug {
		record.Request.Body = string(reqBody)
		record.Request.QueryParams = sortedQueryParams(r)
		record.Response.Body = string(respBody)
	}

	if level >= logLevelTrace {
		record.Request.Headers = sortedHeaderMap(r.Header)
		record.Response.Headers = sortedHeaderMap(respHeader)
	}

	if includeCurl && level >= logLevelDebug {
		record.CurlCommand = buildCurlCommand(r, reqBody, curlHost)
	}

	return record
}

func formatExchangeLog(record exchangeRecord, format logFormat) (string, error) {
	switch format {
	case logFormatJSON:
		data, err := json.Marshal(record)
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	default:
		return formatExchangeLogText(record), nil
	}
}

func formatExchangeLogText(record exchangeRecord) string {
	var b strings.Builder
	includeDetail := record.LogLevel == logLevelDebug.String() || record.LogLevel == logLevelTrace.String()

	b.WriteString("###################################################################\n")
	fmt.Fprintf(&b, "TIMESTAMP:     %s\n", record.Timestamp)
	fmt.Fprintf(&b, "LOG LEVEL:     %s\n", record.LogLevel)
	b.WriteString("---------- REQUEST ----------\n")
	fmt.Fprintf(&b, "HTTP Method:   %s\n", record.Request.Method)
	fmt.Fprintf(&b, "REQUEST URL:   %s\n", record.Request.URL)

	if includeDetail {
		fmt.Fprintf(&b, "REQUEST BODY:  %s\n", record.Request.Body)
		writeSortedQueryParamsMap(&b, record.Request.QueryParams)
	}

	if len(record.Request.Headers) > 0 {
		writeSortedHeadersMap(&b, record.Request.Headers, "REQUEST HEADER")
	}

	if record.CurlCommand != "" {
		b.WriteString("\nCURL COMMAND: \n")
		b.WriteString(record.CurlCommand)
		b.WriteString("\n\n")
	}

	b.WriteString("---------- RESPONSE ----------\n")
	fmt.Fprintf(&b, "HTTP Status:   %d\n", record.Response.Status)

	if includeDetail {
		fmt.Fprintf(&b, "RESPONSE BODY: %s\n", record.Response.Body)
	}

	if len(record.Response.Headers) > 0 {
		writeSortedHeadersMap(&b, record.Response.Headers, "RESPONSE HEADER")
	}

	b.WriteString("###################################################################\n")

	return b.String()
}

func sortedQueryParams(r *http.Request) map[string][]string {
	query := r.URL.Query()
	if len(query) == 0 {
		return nil
	}

	result := make(map[string][]string, len(query))
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := append([]string(nil), query[key]...)
		sort.Strings(values)
		result[key] = values
	}

	return result
}

func sortedHeaderMap(header http.Header) map[string][]string {
	if len(header) == 0 {
		return nil
	}

	result := make(map[string][]string, len(header))
	names := make([]string, 0, len(header))
	for name := range header {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		values := append([]string(nil), header[name]...)
		result[name] = values
	}

	return result
}

func writeSortedQueryParamsMap(b *strings.Builder, query map[string][]string) {
	if len(query) == 0 {
		return
	}

	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range query[key] {
			fmt.Fprintf(b, "Query Param:   %s = %s\n", key, value)
		}
	}
}

func writeSortedHeadersMap(b *strings.Builder, header map[string][]string, label string) {
	if len(header) == 0 {
		return
	}

	names := make([]string, 0, len(header))
	for name := range header {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		for _, value := range header[name] {
			fmt.Fprintf(b, "%-15s %s = %s\n", label+":", name, value)
		}
	}
}

func buildCurlCommand(r *http.Request, body []byte, host string) string {
	var b strings.Builder

	b.WriteString("curl -X")
	b.WriteString(r.Method)
	b.WriteString(" '")
	b.WriteString(strings.TrimRight(host, "/"))
	b.WriteString(r.URL.Path)

	if rawQuery := r.URL.RawQuery; rawQuery != "" {
		b.WriteByte('?')
		b.WriteString(rawQuery)
	}
	b.WriteByte('\'')

	names := make([]string, 0, len(r.Header))
	for name := range r.Header {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if host != "{{host}}" && strings.EqualFold(name, requestDebuggerURLHeader) {
			continue
		}
		for _, value := range r.Header[name] {
			b.WriteString(" \\\n-H '")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(value)
			b.WriteString("'")
		}
	}

	if len(body) > 0 {
		b.WriteString(" \\\n--data-urlencode '")
		b.Write(body)
		b.WriteByte('\'')
	}

	return b.String()
}

func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write log file: %w", err)
	}

	return nil
}
