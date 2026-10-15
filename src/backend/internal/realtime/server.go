package realtime

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/googollee/go-socket.io"
	engineio "github.com/googollee/go-socket.io/engineio"
)

// Server exposes the Socket.IO transport used for ephemeral collaboration
// events. Persistence remains owned by the scene HTTP API.
type Server struct {
	socket         *socketio.Server
	secret         []byte
	mu             sync.Mutex
	rooms          map[string]map[string]roomMember
	turnURL        string
	turnUsername   string
	turnPassword   string
	turnAuthSecret []byte
	authorizeRoom  func(context.Context, uuid.UUID, string) bool
}

func (s *Server) SetTURNConfig(url, username, password string) {
	s.turnURL, s.turnUsername, s.turnPassword = url, username, password
}

func (s *Server) SetTURNAuthSecret(secret, username string) {
	s.turnAuthSecret = []byte(secret)
	s.turnUsername = username
}

// SetWorkspaceAuthorizer restricts realtime room joins to users authorized for
// the workspace in the room key. HTTP middleware enforces the same boundary
// for persisted scene operations.
func (s *Server) SetWorkspaceAuthorizer(authorizer func(context.Context, uuid.UUID, string) bool) {
	s.authorizeRoom = authorizer
}

func (s *Server) ConfigHandler(w http.ResponseWriter, _ *http.Request) {
	iceServers := []map[string]any{}
	if s.turnURL != "" {
		username, credential := s.turnUsername, s.turnPassword
		if len(s.turnAuthSecret) > 0 {
			username, credential = turnRESTCredential(s.turnUsername, s.turnAuthSecret, time.Now(), time.Hour)
		}
		urls := []string{s.turnURL}
		if strings.HasPrefix(s.turnURL, "turn:") && !strings.Contains(s.turnURL, "transport=") {
			urls = append(urls, s.turnURL+"?transport=tcp")
		}
		iceServers = append(iceServers, map[string]any{"urls": urls, "username": username, "credential": credential})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"iceServers": iceServers})
}

func turnRESTCredential(label string, secret []byte, now time.Time, ttl time.Duration) (string, string) {
	username := strconv.FormatInt(now.Add(ttl).Unix(), 10) + ":" + label
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write([]byte(username))
	return username, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type roomMember struct {
	socket socketio.Conn
	userID string
}

type joinPayload struct {
	WorkspaceID string `json:"workspaceId"`
	SceneID     string `json:"sceneId"`
}

type eventPayload struct {
	WorkspaceID string `json:"workspaceId"`
	SceneID     string `json:"sceneId"`
	Data        any    `json:"data,omitempty"`
}

func NewServer(jwtSecret []byte) (*Server, error) {
	transport := socketio.NewServer(&engineio.Options{RequestChecker: func(r *http.Request) (http.Header, error) {
		if _, err := userIDFromRequest(r, jwtSecret); err != nil {
			return nil, err
		}
		return http.Header{}, nil
	}})
	server := &Server{socket: transport, secret: jwtSecret, rooms: make(map[string]map[string]roomMember)}
	transport.OnConnect("", server.onConnect)
	transport.OnDisconnect("", server.onDisconnect)
	transport.OnEvent("", "scene:join", server.onJoin)
	for _, event := range []string{"cursor:move", "scene:update", "voice:offer", "voice:answer", "voice:ice"} {
		transport.OnEvent("", event, server.eventHandler(event))
	}
	return server, nil
}

func (s *Server) Handler() http.Handler { return s.socket }

type connectionState struct {
	room   string
	userID string
}

func (s *Server) onConnect(conn socketio.Conn) error {
	request := &http.Request{Header: conn.RemoteHeader(), URL: &url.URL{RawQuery: conn.URL().RawQuery}}
	userID, err := userIDFromRequest(request, s.secret)
	if err != nil {
		return err
	}
	conn.SetContext(&connectionState{userID: userID})
	return nil
}

func (s *Server) onJoin(conn socketio.Conn, payload joinPayload) {
	state, _ := conn.Context().(*connectionState)
	if state == nil {
		conn.Emit("realtime:error", map[string]string{"message": "authentication is required"})
		return
	}
	room, err := validateRoom(payload)
	if err != nil {
		conn.Emit("realtime:error", map[string]string{"message": err.Error()})
		return
	}
	if s.authorizeRoom != nil {
		workspaceID, _ := uuid.Parse(payload.WorkspaceID)
		if !s.authorizeRoom(context.Background(), workspaceID, state.userID) {
			conn.Emit("realtime:error", map[string]string{"message": "you do not have access to this workspace"})
			return
		}
	}
	if state.room != "" {
		s.leave(state.room, conn)
		conn.Leave(state.room)
	}
	state.room = room
	s.join(room, conn, state.userID)
	conn.Join(room)
	message := eventPayload{WorkspaceID: payload.WorkspaceID, SceneID: payload.SceneID, Data: map[string]string{"userId": state.userID}}
	conn.Emit("presence:update", message)
	s.socket.BroadcastToRoom("", room, "presence:update", message)
}

func (s *Server) eventHandler(eventName string) func(socketio.Conn, eventPayload) {
	return func(conn socketio.Conn, payload eventPayload) {
		state, _ := conn.Context().(*connectionState)
		if state == nil || state.room == "" || payload.WorkspaceID+":"+payload.SceneID != state.room {
			conn.Emit("realtime:error", map[string]string{"message": "join a scene before publishing events"})
			return
		}
		s.socket.BroadcastToRoom("", state.room, eventName, payload)
	}
}

func (s *Server) onDisconnect(conn socketio.Conn, _ string) {
	state, _ := conn.Context().(*connectionState)
	if state == nil || state.room == "" {
		return
	}
	s.leave(state.room, conn)
	s.socket.BroadcastToRoom("", state.room, "presence:update", eventPayload{Data: map[string]string{"userId": state.userID, "status": "offline"}})
}

func (s *Server) join(room string, so socketio.Conn, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rooms[room] == nil {
		s.rooms[room] = make(map[string]roomMember)
	}
	s.rooms[room][so.ID()] = roomMember{socket: so, userID: userID}
}

func (s *Server) leave(room string, so socketio.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if members := s.rooms[room]; members != nil {
		delete(members, so.ID())
		if len(members) == 0 {
			delete(s.rooms, room)
		}
	}
}

func validateRoom(payload joinPayload) (string, error) {
	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil || workspaceID == uuid.Nil {
		return "", fmt.Errorf("workspaceId must be a UUID")
	}
	sceneID, err := uuid.Parse(payload.SceneID)
	if err != nil || sceneID == uuid.Nil {
		return "", fmt.Errorf("sceneId must be a UUID")
	}
	return workspaceID.String() + ":" + sceneID.String(), nil
}

func userIDFromRequest(r *http.Request, secret []byte) (string, error) {
	if r == nil {
		return "", fmt.Errorf("missing socket request")
	}
	tokenValue := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if tokenValue == "" {
		tokenValue = r.URL.Query().Get("token")
	}
	if tokenValue == "" {
		if len(secret) == 0 {
			return "anonymous", nil
		}
		return "", fmt.Errorf("authentication token is required")
	}
	token, err := jwt.Parse(tokenValue, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid or expired token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("user id missing from token")
	}
	if _, err := uuid.Parse(sub); err != nil {
		return "", fmt.Errorf("invalid user id")
	}
	return sub, nil
}
