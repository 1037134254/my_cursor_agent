package rag

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/qdrant/go-client/qdrant"

	"my_cursor/internal/embed"
	"my_cursor/internal/textchunk"
)

var (
	svc     *Service
	svcErr  error
	svcOnce sync.Once
)

// Service 连接 Qdrant，完成切片、嵌入、写入与向量检索。
type Service struct {
	client     *qdrant.Client
	collection string
	vectorSize uint64
	pointSeq   uint64
}

// Get 懒加载单例；失败时返回错误（例如 Qdrant 未启动）。
func Get() (*Service, error) {
	svcOnce.Do(func() {
		svc, svcErr = newService()
	})
	return svc, svcErr
}

func newService() (*Service, error) {
	port, err := strconv.Atoi(envOrDefault("QDRANT_GRPC_PORT", "6334"))
	if err != nil || port <= 0 {
		port = 6334
	}
	vecSize, err := strconv.ParseUint(envOrDefault("RAG_VECTOR_SIZE", "768"), 10, 64)
	if err != nil || vecSize == 0 {
		vecSize = 768
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:     envOrDefault("QDRANT_HOST", "127.0.0.1"),
		Port:     port,
		PoolSize: 1,
	})
	if err != nil {
		return nil, err
	}
	return &Service{
		client:     client,
		collection: envOrDefault("RAG_COLLECTION", "local_docs"),
		vectorSize: vecSize,
	}, nil
}

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func (s *Service) ensureCollection(ctx context.Context) error {
	ok, err := s.client.CollectionExists(ctx, s.collection)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     s.vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

// Ingest 将长文本切片、嵌入后写入 Qdrant。
func (s *Service) Ingest(ctx context.Context, text, source string, maxRunes, overlap int) (int, error) {
	if err := s.ensureCollection(ctx); err != nil {
		return 0, err
	}
	chunks := textchunk.Split(text, maxRunes, overlap)
	if len(chunks) == 0 {
		return 0, fmt.Errorf("无有效文本")
	}
	wait := true
	var points []*qdrant.PointStruct
	for i, ch := range chunks {
		vec, err := embed.EmbedOne(ch)
		if err != nil {
			return 0, err
		}
		if uint64(len(vec)) != s.vectorSize {
			return 0, fmt.Errorf("向量维度=%d 与 RAG_VECTOR_SIZE=%d 不一致，请调整 EMBED_MODEL 或环境变量", len(vec), s.vectorSize)
		}
		id := atomic.AddUint64(&s.pointSeq, 1)
		pl, err := qdrant.TryValueMap(map[string]any{
			"text":        ch,
			"chunk_index": i,
			"source":      source,
		})
		if err != nil {
			return 0, err
		}
		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(id),
			Vectors: qdrant.NewVectorsDense(vec),
			Payload: pl,
		})
	}
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points:         points,
		Wait:           &wait,
	})
	if err != nil {
		return 0, err
	}
	return len(points), nil
}

// Hit 单条检索结果。
type Hit struct {
	Score float32 `json:"score"`
	Text  string  `json:"text"`
}

// Search 对查询句嵌入后在集合中做近邻检索。
func (s *Service) Search(ctx context.Context, query string, limit uint64) ([]Hit, error) {
	if err := s.ensureCollection(ctx); err != nil {
		return nil, err
	}
	vec, err := embed.EmbedOne(query)
	if err != nil {
		return nil, err
	}
	if uint64(len(vec)) != s.vectorSize {
		return nil, fmt.Errorf("向量维度=%d 与 RAG_VECTOR_SIZE=%d 不一致", len(vec), s.vectorSize)
	}
	if limit == 0 {
		limit = 5
	}
	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collection,
		Query:          qdrant.NewQueryDense(vec),
		Limit:          qdrant.PtrOf(limit),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(res))
	for _, sp := range res {
		txt := ""
		if sp.Payload != nil {
			if v := sp.Payload["text"]; v != nil {
				txt = v.GetStringValue()
			}
		}
		out = append(out, Hit{Score: sp.Score, Text: txt})
	}
	return out, nil
}
