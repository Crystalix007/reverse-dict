package backend

type Model int

const (
	ModelQwen3Embedding8B4B_DWQ     Model = 1
	ModelAppleNLContextualEmbedding Model = 2
	ModelOpenAITextEmbedding3Large  Model = 3
)

func (m Model) String() string {
	switch m {
	case ModelQwen3Embedding8B4B_DWQ:
		return "mlx-community/Qwen3-Embedding-8B-4bit-DWQ"
	case ModelAppleNLContextualEmbedding:
		return "apple/nlcontextualembedding"
	case ModelOpenAITextEmbedding3Large:
		return "openai/text-embedding-3-large"
	}

	panic("unknown model")
}

var Models = []Model{
	ModelQwen3Embedding8B4B_DWQ,
	//.TODO: re-enable when the Apple NL Contextual Embedding is deployed.
	// ModelAppleNLContextualEmbedding,
	ModelOpenAITextEmbedding3Large,
}
