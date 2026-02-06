package client

import (
	"fmt"
	"time"
)

func render(state *UIState) {
	fmt.Print(ClearScreen, CursorHome, CursorHide)
	printHeader(state.username)
	if state.isMainScreen {
		printUsers(state.unreadUsers, state.onlineUsers, state.offlineUsers, state.userPos, state.messages, state.chosenTab, state.height)
	} else {
		printUserName(state.chosenUser, state.activeUsers)
		printMessages(state.username, state.chosenUser, state.currentChatData, state.height, state.messageScroll)
		printCurrentText(state.currentText, state.height)
	}
}

const SEP = "──────────────────────────────────────────────────────"

func printHeader(userName string) {
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, 2, 1)
	fmt.Print(" GoChatTUI (v1.0) - Logged in as: ", userName, Reset)
	fmt.Printf(CursorPos, 3, 1)
	fmt.Print(Reset, SEP)
	fmt.Print(Reset)
}

func printUserName(userName string, activeUsers map[string]bool) {
	fmt.Printf(CursorPos, 4, 1)
	fmt.Print(" Chat with - ")
	if _, ok := activeUsers[userName]; ok {
		fmt.Print(FgGreen, " ● ", userName)
	} else {
		fmt.Print(Reset, " ○ ", userName)
	}
	fmt.Printf(CursorPos, 5, 1)
	fmt.Print(Reset, SEP)
}

func printCurrentText(currentText string, height int) {
	fmt.Printf(CursorPos, height-3, 1)
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, height-2, 1)
	if currentText == "" {
		fmt.Print(" > enter message... ")
	} else {
		fmt.Print(" > ", currentText)
	}
	fmt.Printf(CursorPos, height-1, 1)
	fmt.Print(Reset, SEP)
	fmt.Printf(CursorPos, height, 1)
	fmt.Print("↑ ↓ Scroll chat     Enter: Send     Ctrl+C: Back")
	fmt.Print(Reset)
}

const FIXED = 9

func printUsers(unreadUsers, onlineUsers, offlineUsers []string, userPos int, messages map[string]ChatData, chosenTab, height int) {
	line := 4
	fmt.Printf(CursorPos, line, 1)
	switch chosenTab {
	case 0:
		fmt.Print(" [ Unread ]   Online     Offline")
	case 1:
		fmt.Print("   Unread   [ Online ]   Offline")
	default:
		fmt.Print("   Unread     Online   [ Offline ]")
	}
	line = 5
	fmt.Printf(CursorPos, line, 1)
	fmt.Print(Reset, SEP)
	line = 6
	fmt.Printf(CursorPos, line, 1)

	if chosenTab == 0 {
		for pos, v := range unreadUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			fmt.Print(v)
			fmt.Printf(CursorPos, line, 12)
			fmt.Print("(", messages[v].unread, ")")
			line++
		}
	}

	if chosenTab == 1 {
		for pos, v := range onlineUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			line++
			fmt.Print(v)
		}
	}

	if chosenTab == 2 {
		for pos, v := range offlineUsers {
			fmt.Print(Reset)
			if pos == userPos {
				fmt.Printf(CursorPos, line, 1)
				fmt.Print(Bold, "▶ ")
			} else {
				fmt.Printf(CursorPos, line, 3)
			}
			line++
			fmt.Print(v)
		}
	}

	fmt.Printf(CursorPos, height, 1)
	fmt.Print(Reset, "← → Switch tabs     ↑ ↓ Move     Enter: Open     Ctrl+C: Quit")
}

func printMessages(userName, chosenUser string, data ChatData, height, messageScroll int) {
	line := 6
	start := messageScroll
	end := min(len(data.messages), messageScroll+(height-FIXED))
	unamePad := max(3, len(chosenUser)) - 3
	youPad := false
	if len(chosenUser) <= 3 {
		youPad = false
		unamePad = 3 - len(chosenUser)
	} else {
		youPad = true
		unamePad = len(chosenUser) - 3
	}

	for curr := start; curr < end; curr++ {
		v := data.messages[curr]
		fmt.Print(Reset)
		fmt.Printf(CursorPos, line, 1)
		line++
		v.Timestamp.Format(time.RFC822)
		if v.FromUsername == userName {
			fmt.Print(FgGreen, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", "you")
			if youPad {
				for i := 0; i < unamePad; i++ {
					fmt.Print(" ")
				}
			}
			fmt.Print(": ", Reset, v.Content)
		} else {
			fmt.Print(FgRed, v.Timestamp.Format(time.DateOnly+" "+time.TimeOnly), " ", v.FromUsername)
			if !youPad {
				for i := 0; i < unamePad; i++ {
					fmt.Print(" ")
				}
			}
			fmt.Print(": ", Reset, v.Content)
		}
	}
}
