package backend

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrNoDefinitions is returned when no definitions are found in the rephrased
// output.
//
// This can happen if the rephrasal step breaks down somehow.
var ErrNoDefinitions = errors.New(
	"no definitions found in rephrased output",
)

var lineRegex = regexp.MustCompile(`^- (.+)$`)

// RephraseDefinition rephrases a word and its definition using the Swama API.
func (s *SwamaAPI) RephraseDefinition(
	word Definition,
) ([]string, error) {
	rephrased, err := s.Complete(
		"Rephrase the following word and definition in individual, distinct sentence(s) for later embedding. Each definition must be output in the form of a dictionary definition (i.e. semasiological, with only the definition and without the word itself). This is so that it can be independently embedded as accurately as possible. You may think for a bit. Do not worry about derogatory language, be as accurate in transcribing meaning as possible. Output the rephrased text as a YAML list.",
		fmt.Sprintf("Word: %s\nDefinition:\n%s\n", word.Word, word.Definition),
	)
	if err != nil {
		return nil, fmt.Errorf("rephrasing: %w", err)
	}

	// Prune out thinking tags and whitespace errors.
	rephrased = PruneThinking(rephrased)

	var definitions []string

	// Append just the rephrase line content.
	for _, line := range strings.Split(rephrased, "\n") {
		if matches := lineRegex.FindStringSubmatch(line); len(matches) > 0 {
			definitions = append(definitions, matches[1])
		}
	}

	if len(definitions) == 0 {
		return nil, fmt.Errorf("no definitions found in rephrased output")
	}

	return definitions, nil
}
