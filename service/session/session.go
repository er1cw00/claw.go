package session

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/provider"
)

// BuildSessionKey concatenates agent, channel and chatId with ":" and returns
// the MD5 hex digest of the combined string.
func SessionKey(agent, channel, chatId string) string {
	raw := agent + ":" + channel + ":" + chatId
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

type Session struct {
	key       string
	messages  []provider.Message
	createdAt time.Time
	updatedAt time.Time
	metadata  map[string]any
}

func (s *Session) Key() string {
	return s.key
}

func (s *Session) AddMessage(message provider.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func (s *Session) GetHistory(limit int) []provider.Message {
	if limit <= 0 {
		return []provider.Message{}
	}
	n := len(s.messages)
	if limit >= n {
		out := make([]provider.Message, n)
		copy(out, s.messages)
		return out
	}
	out := make([]provider.Message, limit)
	copy(out, s.messages[n-limit:])
	return out
}

func (s *Session) Clear() {
	s.messages = []provider.Message{}
	s.messages = s.messages[:0]
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
	mu       sync.RWMutex
}

var ssService *Service = &Service{
	sessions: make(map[string]*Session),
}

func GetService() *Service {
	return ssService
}

func (ss *Service) Start() error {
	logger.Info("[Session] Service Start !")
	ss.storage = filepath.Join(base.GetSettings().WorkPath, "session")
	return nil
}

// Stop 停止缓存服务
func (s *Service) Stop() {
	logger.Info("[Session] Service Stop")
}

func (s *Service) Name() string {
	return "Session"
}

func (ss *Service) NewSession(key string) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if s, ok := ss.sessions[key]; ok {
		return s
	}
	s := &Session{
		key:       key,
		messages:  []provider.Message{},
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
	ss.sessions[key] = s
	return s
}

func (ss *Service) sessionFilePath(key string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return filepath.Join(ss.storage, hash+".jsonl")
}

func (ss *Service) SaveSession(s *Session) error {
	if s == nil {
		return nil
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	path := ss.sessionFilePath(s.key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	metadata := map[string]any{
		"_type":      "metadata",
		"created_at": s.createdAt.Format(time.RFC3339Nano),
		"updated_at": s.updatedAt.Format(time.RFC3339Nano),
		"metadata":   s.metadata,
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

func (ss *Service) LoadSession(key string) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if s, ok := ss.sessions[key]; ok {
		return s
	}

	path := ss.sessionFilePath(key)
	file, err := os.Open(path)
	if err != nil {
		s := &Session{
			key:       key,
			messages:  []provider.Message{},
			createdAt: time.Now(),
			updatedAt: time.Now(),
		}
		ss.sessions[key] = s
		return s
	}
	defer file.Close()

	s := &Session{
		key:       key,
		messages:  []provider.Message{},
		createdAt: time.Now(),
		updatedAt: time.Now(),
		metadata:  map[string]any{},
	}

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
				s.metadata = meta.Metadata
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
