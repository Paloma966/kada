package ai

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The splitter is where a mistake is quiet: passages that are subtly wrong still index, still embed and
// still retrieve - just badly. These cases pin the three things that go wrong: a chunk that exceeds the
// size the embedding call is budgeted for, a boundary that cuts a Chinese character in half, and an
// overlap that is missing or longer than the passage itself.
func TestChunk(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		size      int
		overlap   int
		want      []string
		checkFunc func(t *testing.T, chunks []string)
	}{
		{
			name: "empty text has no passages",
			text: "   \n\n  ",
			size: 20,
			want: nil,
		},
		{
			name: "short text is one passage",
			text: "一段很短的知识。",
			size: 20,
			want: []string{"一段很短的知识。"},
		},
		{
			name: "paragraphs that fit stay together",
			text: "第一段。\n\n第二段。",
			size: 50,
			want: []string{"第一段。\n第二段。"},
		},
		{
			name: "a long line is cut and the next passage carries the overlap",
			text: strings.Repeat("あ", 25),
			size: 10,
			// Bodies are packed to size-overlap-1, so the prefix, its newline and the body still fit: with
			// 7-rune bodies the 25 runes become 7+7+7+4 and the result is four passages of at most 10.
			overlap: 2,
			checkFunc: func(t *testing.T, chunks []string) {
				if len(chunks) != 4 {
					t.Fatalf("got %d passages, want 4: %q", len(chunks), chunks)
				}
				for i, c := range chunks {
					if got := utf8.RuneCountInString(c); got > 10 {
						t.Errorf("passage %d is %d runes, want at most 10: %q", i, got, c)
					}
				}
				if !strings.HasPrefix(chunks[1], tail(chunks[0], 2)) {
					t.Errorf("passage 1 %q does not start with the overlap of passage 0 %q", chunks[1], chunks[0])
				}
				if !strings.HasPrefix(chunks[2], tail(chunks[1], 2)) {
					t.Errorf("passage 2 %q does not start with the overlap of passage 1 %q", chunks[2], chunks[1])
				}
			},
		},
		{
			name:    "an overlap as large as the passage is ignored rather than looping",
			text:    strings.Repeat("x", 30),
			size:    10,
			overlap: 10,
			checkFunc: func(t *testing.T, chunks []string) {
				if len(chunks) != 3 {
					t.Fatalf("got %d passages, want 3", len(chunks))
				}
			},
		},
		{
			name: "carriage returns do not survive into a passage",
			text: "第一行。\r\n\r\n第二行。",
			size: 50,
			want: []string{"第一行。\n第二行。"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunk(tt.text, tt.size, tt.overlap)
			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d passages %q, want %d %q", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("passage %d is %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Every passage must be valid UTF-8: the corpus is Chinese, and a byte-wise cut produces a passage whose
// characters the embedding API rejects or, worse, silently replaces.
func TestChunkKeepsCharactersWhole(t *testing.T) {
	text := strings.Repeat("短链接平台的知识库内容。", 40)
	for i, passage := range chunk(text, 37, 5) {
		if !utf8.ValidString(passage) {
			t.Errorf("passage %d is not valid UTF-8", i)
		}
		if utf8.RuneCountInString(passage) > 37 {
			t.Errorf("passage %d is %d runes, want at most 37", i, utf8.RuneCountInString(passage))
		}
	}
}

// The bundled corpus has to survive the splitter with something to retrieve: a chunker that returns one
// empty passage would make every question fall back to "no relevant knowledge" without failing anything.
func TestDocumentsAreChunkable(t *testing.T) {
	docs, err := Documents()
	if err != nil {
		t.Fatalf("Documents() failed: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("the bundled knowledge base is empty")
	}

	for _, doc := range docs {
		passages := chunk(doc.Text, passageSize, passageOverlap)
		if len(passages) == 0 {
			t.Errorf("%s produced no passages", doc.Source)
		}
		for i, passage := range passages {
			if strings.TrimSpace(passage) == "" {
				t.Errorf("%s passage %d is empty", doc.Source, i)
			}
		}
	}
}
