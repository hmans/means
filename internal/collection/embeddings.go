package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// EmbeddingConfig opts a collection into network-backed embedding maintenance.
// Credentials and endpoint overrides belong in the process environment.
type EmbeddingConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

func embeddingConfig(value any) (*EmbeddingConfig, error) {
	if value == false {
		return nil, nil
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("embeddings must be an object or false")
	}
	cfg := &EmbeddingConfig{}
	for key, value := range fields {
		switch key {
		case "provider":
			provider, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("embeddings.provider must be openai or openrouter")
			}
			cfg.Provider = provider
		case "model":
			model, ok := value.(string)
			if !ok || model == "" {
				return nil, fmt.Errorf("embeddings.model must be a string")
			}
			cfg.Model = model
		case "dimensions":
			n, ok := value.(json.Number)
			if !ok {
				return nil, fmt.Errorf("embeddings.dimensions must be a positive integer")
			}
			i, err := n.Int64()
			if err != nil || i < 1 || i > 3072 {
				return nil, fmt.Errorf("embeddings.dimensions must be an integer from 1 to 3072")
			}
			cfg.Dimensions = int(i)
		default:
			return nil, fmt.Errorf("unknown embeddings configuration field %q", key)
		}
	}
	prefix := ""
	switch cfg.Provider {
	case "openai":
	case "openrouter":
		prefix = "openai/"
	default:
		return nil, fmt.Errorf("embeddings.provider is required and must be openai or openrouter")
	}
	if cfg.Model == "" {
		cfg.Model = prefix + "text-embedding-3-small"
	}
	maxDimensions := 1536
	switch cfg.Model {
	case prefix + "text-embedding-3-small":
	case prefix + "text-embedding-3-large":
		maxDimensions = 3072
	default:
		return nil, fmt.Errorf("embeddings.model for %s must be %s or %s", cfg.Provider, prefix+"text-embedding-3-small", prefix+"text-embedding-3-large")
	}
	if cfg.Dimensions == 0 {
		cfg.Dimensions = 512
	}
	if cfg.Dimensions > maxDimensions {
		return nil, fmt.Errorf("embeddings.dimensions exceeds model maximum %d", maxDimensions)
	}
	return cfg, nil
}

func (c *Collection) Embeddings() *EmbeddingConfig {
	if c.config == nil {
		return nil
	}
	value, exists := c.config["embeddings"]
	if !exists {
		return nil
	}
	cfg, _ := embeddingConfig(value) // Validated when loading configuration.
	return cfg
}

// The input version is part of the cache identity. Do not include IDs or file
// formatting: renames and unrelated application-data edits reuse paid vectors.
func embeddingInput(frontmatter map[string]any, body string) string {
	return "Title: " + frontmatter["title"].(string) + "\n\n" + body
}

type embeddingRecord struct {
	ID     string    `json:"id"`
	Hash   string    `json:"input_sha256"`
	Vector []float32 `json:"vector"`
}

type embeddingHeader struct {
	Version int    `json:"version"`
	Profile string `json:"profile_sha256"`
	Hash    string `json:"records_sha256"`
}

type embeddingStore struct {
	progress  func(string)
	generated int
	ctx       context.Context
	cfg       *EmbeddingConfig
	endpoint  string
	filename  string
	profile   string
	lock      *flock.Flock
	records   map[string]embeddingRecord
	dirty     bool
}

func (s *embeddingStore) report(format string, args ...any) {
	if s.progress != nil {
		s.progress(fmt.Sprintf(format, args...))
	}
}

func (s *embeddingStore) close() {
	if s != nil {
		_ = s.lock.Unlock()
	}
}

// Acquire before reading a document for mutation. Only opted-in collections
// take this lock. External editors remain outside the lock, as with other writes.
func (c *Collection) openEmbeddings() (*embeddingStore, error) {
	s, err := c.embeddingStore()
	if s == nil || err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.filename), 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(filepath.Dir(s.filename), "embeddings.lock")
	if err := embeddingRegularFile(lockPath); err != nil {
		return nil, err
	}
	s.lock, err = tryLock(s.ctx, lockPath, "embeddings", "embeddings are busy; retry after the current operation finishes")
	if err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// embeddingStore resolves the configured profile without locking or loading the
// cache. It returns nil when embeddings are disabled.
func (c *Collection) embeddingStore() (*embeddingStore, error) {
	cfg := c.Embeddings()
	if cfg == nil {
		return nil, nil
	}
	operationContext := c.ctx
	if operationContext == nil {
		operationContext = context.Background()
	}
	endpoint := strings.TrimRight(os.Getenv("MEANS_EMBEDDING_BASE_URL"), "/")
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
		if cfg.Provider == "openrouter" {
			endpoint = "https://openrouter.ai/api/v1"
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("MEANS_EMBEDDING_BASE_URL must be an HTTP(S) base URL without credentials, query, or fragment")
	}
	cache := filepath.Join(c.Path, ".means")
	if stat, err := os.Lstat(cache); err == nil && (!stat.IsDir() || stat.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf(".means must be a directory, not a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	profile, _ := json.Marshal(struct {
		Version  int
		Config   *EmbeddingConfig
		Endpoint string
	}{1, cfg, endpoint})
	return &embeddingStore{ctx: operationContext, cfg: cfg, endpoint: endpoint, filename: filepath.Join(cache, "embeddings.jsonl"), profile: contentHash(string(profile)), records: map[string]embeddingRecord{}, progress: c.embeddingProgress}, nil
}

func embeddingRegularFile(filename string) error {
	stat, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() {
		return fmt.Errorf("embedding cache path %q must be a regular file", filename)
	}
	return nil
}

func (s *embeddingStore) load() error {
	if err := embeddingRegularFile(s.filename); err != nil {
		return err
	}
	data, err := os.ReadFile(s.filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	header, records, ok := bytes.Cut(data, []byte("\n"))
	var h embeddingHeader
	// Missing, obsolete, and corrupt derived data can be regenerated. Never
	// treat filesystem errors as corruption or discard a readable valid prefix.
	if !ok || json.Unmarshal(header, &h) != nil || h.Version != 1 || h.Profile != s.profile || h.Hash != contentHash(string(records)) {
		s.dirty = true
		return nil
	}
	decoded := map[string]embeddingRecord{}
	d := json.NewDecoder(bytes.NewReader(records))
	for {
		var r embeddingRecord
		err := d.Decode(&r)
		if err == io.EOF {
			s.records = decoded
			return nil
		}
		if err != nil || validateDocumentID(r.ID) != nil || !validHash.MatchString(r.Hash) || !validVector(r.Vector, s.cfg.Dimensions) {
			s.dirty = true
			return nil
		}
		if _, exists := decoded[r.ID]; exists {
			s.dirty = true
			return nil
		}
		decoded[r.ID] = r
	}
}

func validVector(vector []float32, dimensions int) bool {
	if len(vector) != dimensions {
		return false
	}
	var norm float64
	for _, value := range vector {
		v := float64(value)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
		norm += v * v
	}
	return norm > 0
}

func (s *embeddingStore) save() error {
	if s == nil || !s.dirty {
		return nil
	}
	if err := embeddingRegularFile(s.filename); err != nil {
		return err
	}
	ids := make([]string, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var records, output bytes.Buffer
	for _, id := range ids {
		if err := json.NewEncoder(&records).Encode(s.records[id]); err != nil {
			return err
		}
	}
	if err := json.NewEncoder(&output).Encode(embeddingHeader{1, s.profile, contentHash(records.String())}); err != nil {
		return err
	}
	output.Write(records.Bytes())
	f, err := os.CreateTemp(filepath.Dir(s.filename), ".embeddings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(output.Bytes())
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), s.filename); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

func (s *embeddingStore) upsert(id string, frontmatter map[string]any, body string) error {
	if s == nil {
		return nil
	}
	input := embeddingInput(frontmatter, body)
	hash := contentHash(input)
	if r, exists := s.records[id]; exists && r.Hash == hash {
		return nil
	}
	s.report("Embedding document %q using %s (%s)", id, s.cfg.Model, s.cfg.Provider)
	vector, err := s.embed(input)
	if err != nil {
		return err
	}
	s.records[id] = embeddingRecord{id, hash, vector}
	s.generated++
	s.dirty = true
	return nil
}

func (s *embeddingStore) remove(id string) error {
	if s == nil {
		return nil
	}
	if _, exists := s.records[id]; exists {
		delete(s.records, id)
		s.dirty = true
	}
	return s.save()
}

func embeddingAPIKey(provider string) (string, error) {
	if key := os.Getenv("MEANS_EMBEDDING_API_KEY"); key != "" {
		return key, nil
	}
	variable := "OPENAI_API_KEY"
	if provider == "openrouter" {
		variable = "OPENROUTER_API_KEY"
	}
	if key := os.Getenv(variable); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("embeddings require %s or MEANS_EMBEDDING_API_KEY", variable)
}

func (s *embeddingStore) embed(input string) ([]float32, error) {
	vectors, err := s.embedBatch([]string{input})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

// 32 valid inputs of at most 8192 tokens stay below the API's 300,000-token
// request limit. Over-limit individual inputs still fail without truncation.
const embeddingBatchSize = 32

func (s *embeddingStore) embedBatch(inputs []string) ([][]float32, error) {
	if len(inputs) == 0 || len(inputs) > embeddingBatchSize {
		return nil, fmt.Errorf("embedding batch must contain 1 to %d inputs", embeddingBatchSize)
	}
	key, err := embeddingAPIKey(s.cfg.Provider)
	if err != nil {
		return nil, err
	}
	var input any = inputs
	if len(inputs) == 1 {
		input = inputs[0]
	}
	payload, err := json.Marshal(map[string]any{"model": s.cfg.Model, "dimensions": s.cfg.Dimensions, "encoding_format": "float", "input": input})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		// Provider error bodies can echo document contents or credentials.
		return nil, fmt.Errorf("embedding endpoint returned HTTP %d; check credentials, model, input length, quota, and provider availability", response.StatusCode)
	}
	maxResponse := int64(len(inputs)) * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}
	if int64(len(data)) > maxResponse {
		return nil, fmt.Errorf("embedding response exceeds %d MiB", len(inputs))
	}
	var result struct {
		Data []struct {
			Index     *int      `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid embedding response JSON")
	}
	if len(result.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding response contains %d vectors; expected %d", len(result.Data), len(inputs))
	}
	vectors := make([][]float32, len(inputs))
	for _, item := range result.Data {
		if item.Index == nil || *item.Index < 0 || *item.Index >= len(inputs) {
			return nil, fmt.Errorf("embedding response has a missing or out-of-range index")
		}
		if vectors[*item.Index] != nil {
			return nil, fmt.Errorf("embedding response has duplicate index %d", *item.Index)
		}
		if !validVector(item.Embedding, s.cfg.Dimensions) {
			return nil, fmt.Errorf("embedding response must contain finite, nonzero vectors with %d dimensions", s.cfg.Dimensions)
		}
		vectors[*item.Index] = item.Embedding
	}
	return vectors, nil
}

// Embed refreshes the derived vectors without changing Markdown. It also prunes
// deleted, malformed, duplicate, and newly nested documents.
func (c *Collection) Embed() (*ListResult, error) {
	return c.EmbedWithOptions(EmbedOptions{})
}

type EmbedOptions struct {
	Force bool // Clear stored vectors before regenerating all visible documents.
}

func (c *Collection) EmbedWithOptions(opts EmbedOptions) (*ListResult, error) {
	s, err := c.openEmbeddings()
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("embeddings are not enabled; add embeddings: {provider: openai} or embeddings: {provider: openrouter} to MEANS.md frontmatter")
	}
	defer s.close()
	if opts.Force {
		s.report("Clearing stored embeddings for forced refresh")
		s.records = map[string]embeddingRecord{}
		s.dirty = true
		if err := s.save(); err != nil {
			return nil, err
		}
	}
	return c.reconcileEmbeddings(s)
}

func (c *Collection) reconcileEmbeddings(s *embeddingStore) (*ListResult, error) {
	s.report("Scanning documents for embeddings")
	result, err := c.ListWithOptions(ListOptions{WithBody: true})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	before := s.generated
	s.report("Checking embeddings for %d documents", len(result.Documents))
	var pending []DocumentSummary
	for _, doc := range result.Documents {
		seen[doc.ID] = true
		if record, ok := s.records[doc.ID]; !ok || record.Hash != contentHash(embeddingInput(doc.Frontmatter, *doc.Body)) {
			pending = append(pending, doc)
		}
	}
	removed := 0
	for id := range s.records {
		if !seen[id] {
			delete(s.records, id)
			removed++
			s.dirty = true
		}
	}
	for start := 0; start < len(pending); start += embeddingBatchSize {
		end := min(start+embeddingBatchSize, len(pending))
		batch := pending[start:end]
		inputs := make([]string, len(batch))
		for i, doc := range batch {
			inputs[i] = embeddingInput(doc.Frontmatter, *doc.Body)
		}
		s.report("Embedding documents %d–%d of %d using %s (%s)", start+1, end, len(pending), s.cfg.Model, s.cfg.Provider)
		vectors, err := s.embedBatch(inputs)
		if err != nil {
			return nil, fmt.Errorf("embed documents %d–%d of %d: %w", start+1, end, len(pending), err)
		}
		// Validate the whole response before changing records, then checkpoint
		// once per batch. A later failure preserves all completed batches.
		for i, doc := range batch {
			s.records[doc.ID] = embeddingRecord{doc.ID, contentHash(inputs[i]), vectors[i]}
		}
		s.dirty = true
		if err := s.save(); err != nil {
			return nil, err
		}
		s.generated += len(batch)
		s.report("Embedded %d/%d documents", end, len(pending))
	}
	if err := s.save(); err != nil {
		return nil, err
	}
	s.report("Embeddings ready: %d generated, %d reused, %d removed", s.generated-before, len(result.Documents)-(s.generated-before), removed)
	return result, nil
}

func (c *Collection) searchSemantic(opts SearchOptions) (*SearchResult, error) {
	s, err := c.embeddingStore()
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("semantic search requires embeddings in MEANS.md")
	}
	// Search never writes the cache, and writers replace it atomically, so read
	// a snapshot without waiting for the lock held by embed or mutations.
	if err := s.load(); err != nil {
		return nil, err
	}
	listed, err := c.ListWithOptions(ListOptions{WithBody: true})
	if err != nil {
		return nil, err
	}
	result := &SearchResult{Documents: []SearchHit{}, Diagnostics: listed.Diagnostics}
	var candidates []DocumentSummary
	for _, doc := range listed.Documents {
		record, ok := s.records[doc.ID]
		if !ok || record.Hash != contentHash(embeddingInput(doc.Frontmatter, *doc.Body)) {
			continue
		}
		if matchesFilter(doc.Frontmatter["data"].(map[string]any), opts.Filter) {
			candidates = append(candidates, doc)
		}
	}
	if len(candidates) == 0 {
		return result, nil
	}
	query, err := s.embed(opts.Query)
	if err != nil {
		return nil, err
	}
	minScore := DefaultSemanticMinScore
	if opts.MinScore != nil {
		minScore = *opts.MinScore
	}
	for _, doc := range candidates {
		score := cosine(query, s.records[doc.ID].Vector)
		if score < minScore {
			continue
		}
		hit := SearchHit{ID: doc.ID, Frontmatter: doc.Frontmatter, Score: &score}
		if opts.WithBody {
			hit.Body = doc.Body
		}
		result.Documents = append(result.Documents, hit)
	}
	sort.Slice(result.Documents, func(i, j int) bool {
		a, b := result.Documents[i], result.Documents[j]
		if *a.Score == *b.Score {
			return a.ID < b.ID
		}
		return *a.Score > *b.Score
	})
	if opts.Limit > 0 && len(result.Documents) > opts.Limit {
		result.HasMore = true
		result.Documents = result.Documents[:opts.Limit]
	}
	return result, nil
}

func cosine(a, b []float32) float64 {
	var dot, aa, bb float64
	for i, x := range a {
		y := b[i]
		dot += float64(x) * float64(y)
		aa += float64(x) * float64(x)
		bb += float64(y) * float64(y)
	}
	return max(-1, min(1, dot/math.Sqrt(aa*bb)))
}
