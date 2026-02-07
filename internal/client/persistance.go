package client

import (
	"encoding/json"
	"os"
	"path"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
)

type ChatData struct {
	Unread   int                `json:"unread"`
	Messages []protocol.Message `json:"messages"`
}

type PersistedState struct {
	Username string              `json:"username"`
	Chats    map[string]ChatData `json:"chats"`
}

func storagePath(username string) string {
	homeDir, _ := os.UserHomeDir()
	return path.Join(homeDir, ".goChatTUIClient", username)
}

func loadState(username string) PersistedState {
	filePath := storagePath(username)
	state := PersistedState{
		Username: username,
		Chats:    make(map[string]ChatData),
	}
	if fileExists(filePath) {
		content, err := os.ReadFile(filePath)
		if err != nil {
			writeState(state)
			return state
		}
		if err = json.Unmarshal(content, &state); err != nil {
			writeState(state)
			return state
		}
		return state
	} else {
		writeState(state)
		return state
	}
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return true
}

func writeState(state PersistedState) {
	bytes, _ := json.Marshal(&state)
	filePath := storagePath(state.Username)
	os.MkdirAll(path.Dir(filePath), 0755)
	os.WriteFile(filePath, bytes, 0644)
}
