package assistant

import (
	"strings"
	"unicode/utf8"
)

// Document is one source text for the knowledge base.
type Document struct {
	// Source names where the text came from. It is stored with every passage, so the context behind an
	// answer can be traced back to a file instead of disappearing into a vector.
	Source string
	Text   string
}

// Passage sizes, matching what the Python service ingested with
// (RecursiveCharacterTextSplitter(chunk_size=500, chunk_overlap=100)).
//
// They are counted in runes, not bytes. The knowledge base is written in Chinese: a 500-byte chunk is
// about 160 characters, and slicing bytes would cut multi-byte characters in half, which the embedding
// API then answers with an error or, worse, a vector of a different text than the one stored.
const (
	passageSize    = 500
	passageOverlap = 100
)

// chunk splits text into passages of at most size runes, overlapping consecutive passages by overlap
// runes.
//
// The overlap is what keeps a sentence that straddles a boundary answerable: the passage that ends with
// half a rule and the one that begins with the other half are both retrieved, and the model can join
// them. It is deliberately not a port of LangChain's splitter - the boundaries differ, and the passages
// are not the same text - because what has to stay compatible is the embedding model and the vector
// length, not the exact cut positions. The test calls this with small sizes, which is why the sizes are
// parameters and not the constants above.
func chunk(text string, size, overlap int) []string {
	if size <= 0 {
		return nil
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")

	// The bodies are packed to leave room for the overlap that will be prepended to them: the prefix, the
	// newline that follows it and the body together have to stay within size, or "at most size" would be a
	// claim the first passage kept and the rest broke.
	bodyLimit := size - overlap
	if overlap > 0 {
		bodyLimit--
	}

	var (
		bodies  []string
		current strings.Builder
	)
	for _, block := range blocks(text, bodyLimit) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		if current.Len() > 0 &&
			utf8.RuneCountInString(current.String())+1+utf8.RuneCountInString(block) > bodyLimit {
			bodies = append(bodies, current.String())
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(block)
	}
	if strings.TrimSpace(current.String()) != "" {
		bodies = append(bodies, current.String())
	}

	passages := make([]string, 0, len(bodies))
	for i, body := range bodies {
		switch {
		case i == 0 || overlap == 0:
			passages = append(passages, body)
		default:
			passages = append(passages, tail(passages[i-1], overlap)+"\n"+body)
		}
	}
	return passages
}

// blocks breaks text down to units that each fit in one passage: first paragraphs, then the lines of a
// paragraph that is too long, then a line that is still too long is cut to size.
func blocks(text string, size int) []string {
	var out []string
	for _, paragraph := range strings.Split(text, "\n\n") {
		if utf8.RuneCountInString(paragraph) <= size {
			out = append(out, paragraph)
			continue
		}
		for _, line := range strings.Split(paragraph, "\n") {
			if utf8.RuneCountInString(line) <= size {
				out = append(out, line)
				continue
			}
			out = append(out, cut(line, size)...)
		}
	}
	return out
}

// cut slices a long line into consecutive pieces of at most size runes.
func cut(line string, size int) []string {
	runes := []rune(line)
	var out []string
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

// tail returns the last n runes of text, for the overlap.
func tail(text string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[len(runes)-n:])
}
