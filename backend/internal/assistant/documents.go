package assistant

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

// knowledgeFiles is the corpus the assistant answers from, compiled into the binary.
//
// Embedded rather than read from disk: the deployment ships binaries to a host that has no checkout, so a
// path here would be a file that exists only on the machine that built it.
//
//go:embed knowledge/*.md
var knowledgeFiles embed.FS

// Documents returns the bundled knowledge base, one Document per file.
func Documents() ([]Document, error) {
	paths, err := fs.Glob(knowledgeFiles, "knowledge/*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to list the knowledge base: %w", err)
	}

	docs := make([]Document, 0, len(paths))
	for _, path := range paths {
		text, err := knowledgeFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		docs = append(docs, Document{
			Source: strings.TrimPrefix(path, "knowledge/"),
			Text:   string(text),
		})
	}
	return docs, nil
}
