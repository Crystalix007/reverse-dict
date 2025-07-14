package backend

import (
	"regexp"
	"strings"

	"github.com/bobg/go-generics/v4/slices"
)

var referenceRegex = regexp.MustCompile(`\[([^\]]+)\]`)

type Definition struct {
	Word       string
	Definition string
	Example    string
	Embeddings []SubDefinition
}

type SimilarDefinition struct {
	Definition
	Phrase   string
	Distance float64
}

type SubDefinition struct {
	Phrase string
	Vector Vector
}

type Vector []float32

func NewEmbeddingFromFloat64(vector []float64) Vector {
	embedding := make(Vector, len(vector))

	for i, v := range vector {
		embedding[i] = float32(v)
	}

	return embedding
}

func SplitDefinition(definition string) []string {
	lines := strings.Split(definition, "\n")

	// Filter out empty lines.
	lines = slices.Map(lines, func(line string) string {
		line = strings.TrimSpace(line)

		line = referenceRegex.ReplaceAllString(line, "$1")

		return line
	})

	lines = slices.Filter(lines, func(line string) bool {
		return line != ""
	})

	return lines
}
