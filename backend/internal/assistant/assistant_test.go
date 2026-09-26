package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// fakeModel is an Eino chat model that streams scripted chunks. The interface is why the assistant can be
// tested end to end without a key, a network or a provider: what the model does is an implementation
// detail behind it.
type fakeModel struct {
	chunks      []string
	generateErr error
	streamErr   error
	requests    [][]*schema.Message
	// historyWhenCalled is how many turns the store held when the model was called, which is how the test
	// sees that the question was stored first.
	historyWhenCalled int
	onCall            func()
}

func (m *fakeModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	return nil, errors.New("Generate is not used by the assistant")
}

func (m *fakeModel) Stream(_ context.Context, messages []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m.requests = append(m.requests, messages)
	if m.historyWhenCalled == 0 && m.onCall != nil {
		m.onCall()
	}
	if m.streamErr != nil {
		return nil, m.streamErr
	}

	stream, writer := schema.Pipe[*schema.Message](len(m.chunks) + 1)
	go func() {
		defer writer.Close()
		for _, chunk := range m.chunks {
			writer.Send(schema.AssistantMessage(chunk, nil), nil)
		}
	}()
	return stream, nil
}

func (m *fakeModel) BindTools([]*schema.ToolInfo) error { return nil }

func (m *fakeModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) { return m, nil }

// fakeRetriever stands in for the knowledge base.
type fakeRetriever struct {
	passages []string
	err      error
	query    string
}

func (r *fakeRetriever) Retrieve(_ context.Context, query string, _ int) ([]string, error) {
	r.query = query
	if r.err != nil {
		return nil, r.err
	}
	return r.passages, nil
}

// fakeHistory records the turns in the order the assistant stores them.
type fakeHistory struct {
	stored     []ChatMessage
	started    string
	appended   []ChatMessage
	current    Conversation
	loadErr    error
	appendErr  error
	currentErr error
}

func (h *fakeHistory) Start(context.Context, int64) (string, error) {
	h.started = "new-conversation"
	return h.started, nil
}

func (h *fakeHistory) Current(context.Context, int64) (Conversation, error) {
	return h.current, h.currentErr
}

func (h *fakeHistory) Restart(context.Context, int64) (string, error) {
	h.appended = nil
	return "restarted-conversation", nil
}

func (h *fakeHistory) Load(context.Context, int64, string) ([]ChatMessage, error) {
	if h.loadErr != nil {
		return nil, h.loadErr
	}
	return append([]ChatMessage(nil), h.stored...), nil
}

func (h *fakeHistory) Append(_ context.Context, _ int64, _ string, role, content string) error {
	if h.appendErr != nil {
		return h.appendErr
	}
	h.appended = append(h.appended, ChatMessage{Role: role, Content: content})
	return nil
}

// collect drains the events of one turn.
func collect(t *testing.T, events <-chan Event) []Event {
	t.Helper()
	var out []Event
	for event := range events {
		out = append(out, event)
	}
	return out
}

func deltas(events []Event) []string {
	var out []string
	for _, event := range events {
		if event.Name == EventToken {
			out = append(out, event.Data.(tokenPayload).Delta)
		}
	}
	return out
}

// The answer arrives token by token and the last event is the done event, which is what lets the page
// remember the conversation for the next question.
func TestStreamEmitsTokensThenDone(t *testing.T) {
	history := &fakeHistory{}
	a := NewAssistant(&fakeModel{chunks: []string{"你", "好"}}, &fakeRetriever{}, history)

	events, err := a.Stream(context.Background(), 7, "c-1", "打个招呼")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	got := collect(t, events)

	if len(got) != 3 {
		t.Fatalf("got %d events, want 3: %+v", len(got), got)
	}
	if got[0].Name != EventToken || got[1].Name != EventToken || got[2].Name != EventDone {
		t.Fatalf("unexpected event order: %+v", got)
	}
	if after := deltas(got); after[0] != "你" || after[1] != "好" {
		t.Errorf("deltas = %v, want [你 好]", after)
	}
	if payload := got[2].Data.(donePayload); payload.ConversationID != "c-1" {
		t.Errorf("done carries conversation %q, want c-1", payload.ConversationID)
	}

	// Both turns are stored: the question before the model runs, the answer after it finishes.
	if len(history.appended) != 2 ||
		history.appended[0].Role != "user" || history.appended[0].Content != "打个招呼" ||
		history.appended[1].Role != "assistant" || history.appended[1].Content != "你好" {
		t.Errorf("stored turns = %+v, want the question then the answer", history.appended)
	}
}

// The question is stored before the model is called: an answer that fails half way must still leave the
// user's question behind, which is what makes a retry readable.
func TestStreamStoresTheQuestionBeforeTheModelRuns(t *testing.T) {
	history := &fakeHistory{}
	chatModel := &fakeModel{chunks: []string{"好"}}
	chatModel.onCall = func() { chatModel.historyWhenCalled = len(history.appended) }
	a := NewAssistant(chatModel, &fakeRetriever{}, history)

	events, err := a.Stream(context.Background(), 7, "c-1", "问题")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	collect(t, events)

	if chatModel.historyWhenCalled != 1 {
		t.Errorf("the store held %d turns when the model was called, want the question already stored",
			chatModel.historyWhenCalled)
	}
}

// A conversation id the caller does not have yet means a new conversation, and the done event is what
// tells the page which one it got.
func TestStreamStartsAConversationWhenThereIsNone(t *testing.T) {
	history := &fakeHistory{}
	a := NewAssistant(&fakeModel{chunks: []string{"好"}}, &fakeRetriever{}, history)

	events, err := a.Stream(context.Background(), 7, "", "第一句")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	got := collect(t, events)

	if history.started != "new-conversation" {
		t.Error("no conversation was started for a request without one")
	}
	if payload := got[len(got)-1].Data.(donePayload); payload.ConversationID != "new-conversation" {
		t.Errorf("done carries conversation %q, want the new one", payload.ConversationID)
	}
}

// The knowledge base is an enhancement, not a dependency: when it fails the answer still arrives, and the
// prompt says there is no reference material rather than pretending there is.
func TestStreamAnswersWithoutContextWhenRetrievalFails(t *testing.T) {
	chatModel := &fakeModel{chunks: []string{"好"}}
	a := NewAssistant(chatModel, &fakeRetriever{err: errors.New("pgvector is not installed")}, &fakeHistory{})

	events, err := a.Stream(context.Background(), 7, "c-1", "问题")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	got := collect(t, events)

	for _, event := range got {
		if event.Name == EventError {
			t.Fatalf("a retrieval failure must not reach the user: %+v", event)
		}
	}
	if got[len(got)-1].Name != EventDone {
		t.Fatalf("the turn did not finish: %+v", got)
	}

	last := chatModel.requests[0][len(chatModel.requests[0])-1]
	if !strings.Contains(last.Content, noKnowledge) {
		t.Errorf("prompt = %q, want it to say the knowledge base holds nothing relevant", last.Content)
	}
}

// Retrieved passages are put in front of the question, separated, exactly where the Python prompt had them.
func TestStreamPutsThePassagesInFrontOfTheQuestion(t *testing.T) {
	chatModel := &fakeModel{chunks: []string{"好"}}
	retriever := &fakeRetriever{passages: []string{"短链是一条记录。", "点击会被记录。"}}
	a := NewAssistant(chatModel, retriever, &fakeHistory{})

	events, err := a.Stream(context.Background(), 7, "c-1", "短链是什么")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	collect(t, events)

	if retriever.query != "短链是什么" {
		t.Errorf("retrieved for %q, want the question", retriever.query)
	}
	last := chatModel.requests[0][len(chatModel.requests[0])-1]
	want := "【参考资料】\n短链是一条记录。\n\n---\n\n点击会被记录。\n\n【用户问题】短链是什么"
	if last.Content != want {
		t.Errorf("prompt = %q, want %q", last.Content, want)
	}
	if role := chatModel.requests[0][0].Role; role != schema.System {
		t.Errorf("first message role = %q, want the system prompt", role)
	}
}

// A model that refuses the request is reported as an error event - the headers are already sent, so there
// is no status code left to use - and nothing is stored as the answer.
func TestStreamReportsAModelFailureAsAnErrorEvent(t *testing.T) {
	history := &fakeHistory{}
	a := NewAssistant(&fakeModel{streamErr: errors.New("401 unauthorized")}, &fakeRetriever{}, history)

	events, err := a.Stream(context.Background(), 7, "c-1", "问题")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	got := collect(t, events)

	if len(got) != 1 || got[0].Name != EventError {
		t.Fatalf("events = %+v, want a single error event", got)
	}
	if message := got[0].Data.(errorPayload).Message; !strings.HasPrefix(message, "AI 服务异常：") {
		t.Errorf("message = %q, want the page's prefix", message)
	}
	for _, turn := range history.appended {
		if turn.Role == "assistant" {
			t.Error("a failed answer was stored")
		}
	}
}

// A question that cannot even be loaded fails before the stream starts, so the handler can still answer
// with a status code. This is the branch the page turns into a toast rather than an empty bubble.
func TestStreamFailsBeforeTheFirstEvent(t *testing.T) {
	a := NewAssistant(&fakeModel{}, &fakeRetriever{}, &fakeHistory{loadErr: errors.New("database is down")})

	if _, err := a.Stream(context.Background(), 7, "c-1", "问题"); err == nil {
		t.Fatal("expected the load failure to be returned to the caller")
	}
}

// When the browser goes away the producer stops: no done event, no half-written answer stored. Without the
// context check the goroutine would block forever on a channel nobody reads.
func TestStreamStopsWhenTheClientGoesAway(t *testing.T) {
	history := &fakeHistory{}
	a := NewAssistant(&fakeModel{chunks: []string{"你", "好"}}, &fakeRetriever{}, history)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events, err := a.Stream(ctx, 7, "c-1", "问题")
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	got := collect(t, events)

	if len(got) != 0 {
		t.Errorf("events = %+v, want none for a client that is gone", got)
	}
	for _, turn := range history.appended {
		if turn.Role == "assistant" {
			t.Error("an answer nobody could read was stored")
		}
	}
}
