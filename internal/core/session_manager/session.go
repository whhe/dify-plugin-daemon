package session_manager

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/langgenius/dify-plugin-daemon/internal/core/dify_invocation"
	"github.com/langgenius/dify-plugin-daemon/internal/core/plugin_daemon/access_types"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/cache"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/log"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/parser"
	"github.com/langgenius/dify-plugin-daemon/pkg/entities/plugin_entities"
)

var (
	sessions     map[string]*Session = map[string]*Session{}
	session_lock sync.RWMutex
)

// session need to implement the backwards_invocation.BackwardsInvocationWriter interface
type Session struct {
	ID                  string                              `json:"id"`
	runtime             plugin_entities.PluginLifetime      `json:"-"`
	backwardsInvocation dify_invocation.BackwardsInvocation `json:"-"`

	TenantID               string                                 `json:"tenant_id"`
	UserID                 string                                 `json:"user_id"`
	PluginUniqueIdentifier plugin_entities.PluginUniqueIdentifier `json:"plugin_unique_identifier"`
	ClusterID              string                                 `json:"cluster_id"`
	InvokeFrom             access_types.PluginAccessType          `json:"invoke_from"`
	Action                 access_types.PluginAccessAction        `json:"action"`
	Declaration            *plugin_entities.PluginDeclaration     `json:"declaration"`

	// information about incoming request
	ConversationID *string `json:"conversation_id"`
	MessageID      *string `json:"message_id"`
	AppID          *string `json:"app_id"`
	EndpointID     *string `json:"endpoint_id"`
}

func sessionKey(id string) string {
	return fmt.Sprintf("session_info:%s", id)
}

type NewSessionPayload struct {
	TenantID               string                                 `json:"tenant_id"`
	UserID                 string                                 `json:"user_id"`
	PluginUniqueIdentifier plugin_entities.PluginUniqueIdentifier `json:"plugin_unique_identifier"`
	ClusterID              string                                 `json:"cluster_id"`
	InvokeFrom             access_types.PluginAccessType          `json:"invoke_from"`
	Action                 access_types.PluginAccessAction        `json:"action"`
	Declaration            *plugin_entities.PluginDeclaration     `json:"declaration"`
	BackwardsInvocation    dify_invocation.BackwardsInvocation    `json:"backwards_invocation"`
	IgnoreCache            bool                                   `json:"ignore_cache"`
	ConversationID         *string                                `json:"conversation_id"`
	MessageID              *string                                `json:"message_id"`
	AppID                  *string                                `json:"app_id"`
	EndpointID             *string                                `json:"endpoint_id"`
}

func NewSession(payload NewSessionPayload) *Session {
	s := &Session{
		ID:                     uuid.New().String(),
		TenantID:               payload.TenantID,
		UserID:                 payload.UserID,
		PluginUniqueIdentifier: payload.PluginUniqueIdentifier,
		ClusterID:              payload.ClusterID,
		InvokeFrom:             payload.InvokeFrom,
		Action:                 payload.Action,
		Declaration:            payload.Declaration,
		backwardsInvocation:    payload.BackwardsInvocation,
		ConversationID:         payload.ConversationID,
		MessageID:              payload.MessageID,
		AppID:                  payload.AppID,
		EndpointID:             payload.EndpointID,
	}

	session_lock.Lock()
	sessions[s.ID] = s
	session_lock.Unlock()

	if !payload.IgnoreCache {
		if err := cache.Store(sessionKey(s.ID), s, time.Minute*30); err != nil {
			log.Error("set session info to cache failed, %s", err)
		}
	}

	return s
}

type GetSessionPayload struct {
	ID          string `json:"id"`
	IgnoreCache bool   `json:"ignore_cache"`
}

func GetSession(payload GetSessionPayload) *Session {
	session_lock.RLock()
	session := sessions[payload.ID]
	session_lock.RUnlock()

	if session == nil {
		// if session not found, it may be generated by another node, try to get it from cache
		session, err := cache.Get[Session](sessionKey(payload.ID))
		if err != nil {
			log.Error("get session info from cache failed, %s", err)
			return nil
		}
		return session
	}

	return session
}

type DeleteSessionPayload struct {
	ID          string `json:"id"`
	IgnoreCache bool   `json:"ignore_cache"`
}

func DeleteSession(payload DeleteSessionPayload) {
	session_lock.Lock()
	delete(sessions, payload.ID)
	session_lock.Unlock()

	if !payload.IgnoreCache {
		if _, err := cache.Del(sessionKey(payload.ID)); err != nil {
			log.Error("delete session info from cache failed, %s", err)
		}
	}
}

type CloseSessionPayload struct {
	IgnoreCache bool `json:"ignore_cache"`
}

func (s *Session) Close(payload CloseSessionPayload) {
	DeleteSession(DeleteSessionPayload{
		ID:          s.ID,
		IgnoreCache: payload.IgnoreCache,
	})
}

func (s *Session) BindRuntime(runtime plugin_entities.PluginLifetime) {
	s.runtime = runtime
}

func (s *Session) Runtime() plugin_entities.PluginLifetime {
	return s.runtime
}

func (s *Session) BindBackwardsInvocation(backwardsInvocation dify_invocation.BackwardsInvocation) {
	s.backwardsInvocation = backwardsInvocation
}

func (s *Session) BackwardsInvocation() dify_invocation.BackwardsInvocation {
	return s.backwardsInvocation
}

type PLUGIN_IN_STREAM_EVENT string

const (
	PLUGIN_IN_STREAM_EVENT_REQUEST  PLUGIN_IN_STREAM_EVENT = "request"
	PLUGIN_IN_STREAM_EVENT_RESPONSE PLUGIN_IN_STREAM_EVENT = "backwards_response"
)

func (s *Session) Message(event PLUGIN_IN_STREAM_EVENT, data any) []byte {
	return parser.MarshalJsonBytes(map[string]any{
		"session_id":      s.ID,
		"conversation_id": s.ConversationID,
		"message_id":      s.MessageID,
		"app_id":          s.AppID,
		"endpoint_id":     s.EndpointID,
		"event":           event,
		"data":            data,
	})
}

func (s *Session) Write(event PLUGIN_IN_STREAM_EVENT, action access_types.PluginAccessAction, data any) error {
	if s.runtime == nil {
		return errors.New("runtime not bound")
	}
	s.runtime.Write(s.ID, action, s.Message(event, data))
	return nil
}
