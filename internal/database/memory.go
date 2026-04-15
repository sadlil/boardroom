package database

import (
	"context"
	"hash/fnv"
	"math"

	"github.com/golang/glog"
	"github.com/philippgille/chromem-go"
)

// LocalEmbeddingFunc returns a chromem.EmbeddingFunc that produces deterministic
// embeddings locally using FNV hashing. This eliminates any dependency on external
// APIs (e.g., OpenAI) and prevents 401 Unauthorized errors.
// The embedding is a fixed-size vector of 384 dimensions derived from character
// n-grams of the input text.
func LocalEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		const dims = 384
		vec := make([]float32, dims)

		// Generate overlapping 3-char n-grams and hash each into a dimension
		for i := 0; i < len(text); i++ {
			end := i + 3
			if end > len(text) {
				end = len(text)
			}
			gram := text[i:end]
			h := fnv.New32a()
			h.Write([]byte(gram))
			idx := h.Sum32() % uint32(dims)
			vec[idx] += 1.0
		}

		// L2-normalize the vector so cosine similarity works correctly
		var norm float64
		for _, v := range vec {
			norm += float64(v) * float64(v)
		}
		if norm > 0 {
			norm = math.Sqrt(norm)
			for i := range vec {
				vec[i] = float32(float64(vec[i]) / norm)
			}
		}

		return vec, nil
	}
}

type VectorMemory struct {
	collection *chromem.Collection
}

func NewVectorMemory(dbPath string, embedFunc chromem.EmbeddingFunc) (*VectorMemory, error) {
	// Persistent vector database instance
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, err
	}

	// Create or get collection using the provided embedding function
	collection, err := db.GetOrCreateCollection("boardroom_memory", nil, embedFunc)
	if err != nil {
		return nil, err
	}

	glog.Infoln("Chromem-go vector memory initialized")
	return &VectorMemory{collection: collection}, nil
}

func (m *VectorMemory) AddDocument(ctx context.Context, id, content string, metadata map[string]string) error {
	doc := chromem.Document{
		ID:       id,
		Metadata: metadata,
		Content:  content,
	}
	// We pass simply an array of docs
	return m.collection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

func (m *VectorMemory) Search(ctx context.Context, query string, limit int) ([]chromem.Document, error) {
	count := m.collection.Count()
	if count == 0 {
		return []chromem.Document{}, nil
	}
	if limit > count {
		limit = count
	}
	res, err := m.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, err
	}

	docs := make([]chromem.Document, 0, len(res))
	for _, r := range res {
		docs = append(docs, chromem.Document{
			ID:        r.ID,
			Metadata:  r.Metadata,
			Embedding: r.Embedding,
			Content:   r.Content,
		})
	}
	return docs, nil
}

func (m *VectorMemory) GetAllDocuments(ctx context.Context) ([]chromem.Document, error) {
	count := m.collection.Count()
	if count == 0 {
		return []chromem.Document{}, nil
	}

	// Query explicitly rejects empty strings in chromem-go v0.7.0, and there is no Get() method.
	// We use a dummy string (e.g., "*") and set nResults to the total count to retrieve all documents.
	res, err := m.collection.Query(ctx, "*", count, nil, nil)
	if err != nil {
		return nil, err
	}

	docs := make([]chromem.Document, 0, len(res))
	for _, r := range res {
		docs = append(docs, chromem.Document{
			ID:        r.ID,
			Metadata:  r.Metadata,
			Embedding: r.Embedding,
			Content:   r.Content,
		})
	}
	return docs, nil
}
