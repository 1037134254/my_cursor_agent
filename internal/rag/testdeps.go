package rag

// ChunkPointIDForTest 暴露 chunkPointID，供 internal/rag_test 使用。
func ChunkPointIDForTest(source, text string) uint64 {
	return chunkPointID(source, text)
}
