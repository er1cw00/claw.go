package session

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/provider"
	"github.com/google/uuid"
	"github.com/pkoukk/tiktoken-go"
)

// BuildSessionKey concatenates agent, channel and chatId with ":" and returns
// the MD5 hex digest of the combined string.
func SessionKey(agent, channel, chatId string) string {
	raw := agent + ":" + channel + ":" + chatId
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

type Session struct {
	key       string
	sessionID string
	messages  []provider.Message
	createdAt time.Time
	updatedAt time.Time
	metadata  map[string]any
}

func (s *Session) Key() string {
	return s.key
}

func (s *Session) SessionID() string {
	return s.sessionID
}

func newSession(key string) *Session {
	sid := uuid.New().String()
	return &Session{
		key:       key,
		sessionID: sid,
		messages:  []provider.Message{},
		createdAt: time.Now(),
		updatedAt: time.Now(),
		metadata: map[string]any{
			"session_id": sid,
		},
	}
}

func (s *Session) AddMessage(message provider.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func (s *Session) GetHistory(limit int) []provider.Message {
	if limit <= 0 || len(s.messages) == 0 {
		return []provider.Message{}
	}
	n := len(s.messages)
	if limit >= n {
		out := make([]provider.Message, n)
		copy(out, s.messages)
		return s.trimHistoryStart(out)
	}
	out := make([]provider.Message, limit)
	copy(out, s.messages[n-limit:])
	return s.trimHistoryStart(out)
}

// trimHistoryStart drops leading tool/assistant messages so the earliest
// remaining message in the returned slice is always from the user.
func (s *Session) trimHistoryStart(messages []provider.Message) []provider.Message {
	start := 0
	for i, msg := range messages {
		if msg.Role == provider.RoleUser {
			start = i
			break
		}
	}
	if start == 0 {
		return messages
	}
	return messages[start:]
}

func (s *Session) SetMessages(messages []provider.Message) {
	s.messages = messages
	s.updatedAt = time.Now()
}

func (s *Session) Reset() {
	s.messages = s.messages[:0]
	s.sessionID = uuid.New().String()
	if s.metadata == nil {
		s.metadata = make(map[string]any)
	}
	s.metadata["session_id"] = s.sessionID
	s.updatedAt = time.Now()
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) UpdatedAt() time.Time {
	return s.updatedAt
}

func (s *Session) Metadata() map[string]any {
	return s.metadata
}

func (s *Session) SetMetadata(metadata map[string]any) {
	s.metadata = metadata
}

func (s *Session) Messages() []provider.Message {
	return s.messages
}

type Service struct {
	storage  string
	sessions map[string]*Session
	tik      *tiktoken.Tiktoken
	mu       sync.RWMutex
}

var ssService = &Service{
	sessions: make(map[string]*Session),
}

func GetService() *Service {
	return ssService
}

func (ss *Service) Start() error {
	var err error = nil
	ss.storage = filepath.Join(base.GetSettings().Workspace, "session")
	ss.tik, err = tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		logger.Errorf("[Session] create tiktoken fail, err: %v", err)
		return err
	}
	logger.Info("[Session] Service Start !")
	return nil
}

// Stop 停止缓存服务
func (s *Service) Stop() {
	logger.Info("[Session] Serivce Stop")
}

func (s *Service) Name() string {
	return "Session"
}

func (ss *Service) Save(s *Session) error {
	if s == nil {
		return nil
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	path := filepath.Join(ss.storage, s.key+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	metadata := map[string]any{
		"_type":       "metadata",
		"created_at":  s.createdAt.Format(time.RFC3339Nano),
		"updated_at":  s.updatedAt.Format(time.RFC3339Nano),
		"session_id":  s.sessionID,
		"metadata":    s.metadata,
	}
	metaLine, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if _, err := file.Write(metaLine); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	for _, msg := range s.messages {
		line, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if _, err := file.Write(line); err != nil {
			return err
		}
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}

	s.updatedAt = time.Now()
	return nil
}

func (ss *Service) GetOrCreate(key string) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if s, ok := ss.sessions[key]; ok {
		return s
	}

	path := filepath.Join(ss.storage, key+".jsonl")
	file, err := os.Open(path)
	if err != nil {
		s := newSession(key)
		ss.sessions[key] = s
		return s
	}
	defer file.Close()

	s := newSession(key)

	scanner := bufio.NewScanner(file)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if first {
			first = false
			var meta struct {
				SessionID string         `json:"session_id"`
				CreatedAt string         `json:"created_at"`
				UpdatedAt string         `json:"updated_at"`
				Metadata  map[string]any `json:"metadata"`
			}
			if err := json.Unmarshal(line, &meta); err == nil {
				if t, err := time.Parse(time.RFC3339Nano, meta.CreatedAt); err == nil {
					s.createdAt = t
				}
				if t, err := time.Parse(time.RFC3339Nano, meta.UpdatedAt); err == nil {
					s.updatedAt = t
				}
				if meta.SessionID != "" {
					s.sessionID = meta.SessionID
				}
				s.metadata = meta.Metadata
				if s.sessionID == "" {
					s.sessionID = uuid.New().String()
					s.metadata["session_id"] = s.sessionID
				}
			}
			continue
		}
		var msg provider.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		s.messages = append(s.messages, msg)
	}

	ss.sessions[key] = s
	return s
}

// EstimateMessageTokens estimates the prompt tokens contributed by one persisted message.
func (ss *Service) EstimateMessageTokens(message provider.Message) int {
	var parts []string

	if message.Content != "" {
		parts = append(parts, message.Content)
	}

	if message.Name != "" {
		parts = append(parts, message.Name)
	}
	if message.ToolCallID != "" {
		parts = append(parts, message.ToolCallID)
	}
	if len(message.ToolCalls) > 0 {
		if b, err := json.Marshal(message.ToolCalls); err == nil {
			parts = append(parts, string(b))
		}
	}

	payload := strings.Join(parts, "\n")
	if payload == "" {
		return 4
	}

	tokens := ss.tik.Encode(payload, nil, nil)
	return max(4, len(tokens)+4)
}
