package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/0ya-sh0/GoChatTUI/internal/protocol"
	"github.com/gorilla/websocket"
)

type Stats struct {
	Sent     uint64
	Received uint64
	Errors   uint64
}

func main() {
	// ---- flags ----
	var (
		username = flag.String("user", "", "username")
		toUser   = flag.String("to", "", "send messages to user")
		duration = flag.Duration("duration", 60*time.Second, "how long to run")
		interval = flag.Duration("interval", time.Second, "send interval")
	)
	flag.Parse()

	if *username == "" || *toUser == "" {
		log.Fatal("user and to flags are required")
	}

	// ---- context with timeout + signals ----
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	stats := &Stats{}
	start := time.Now()

	if err := runClient(ctx, *username, *toUser, *interval, stats); err != nil {
		log.Printf("[%s] exited with error: %v", *username, err)
	}

	// ---- final report ----
	elapsed := time.Since(start)
	fmt.Printf(
		"\n--- client report ---\nuser=%s to=%s\nruntime=%s\nsent=%d recv=%d errors=%d\n",
		*username,
		*toUser,
		elapsed.Truncate(time.Millisecond),
		atomic.LoadUint64(&stats.Sent),
		atomic.LoadUint64(&stats.Received),
		atomic.LoadUint64(&stats.Errors),
	)
}

func runClient(
	ctx context.Context,
	username, to string,
	interval time.Duration,
	stats *Stats,
) error {

	u := url.URL{Scheme: "ws", Host: "localhost:8123", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// claim username
	if err := conn.WriteJSON(&protocol.ClaimUsernameRequest{
		Username: username,
	}); err != nil {
		return fmt.Errorf("claim username: %w", err)
	}

	// ---- reader ----
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					atomic.AddUint64(&stats.Errors, 1)
					return
				}
				_ = msg
				atomic.AddUint64(&stats.Received, 1)
			}
		}
	}()

	// ---- writer ----
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := conn.WriteJSON(&protocol.ForwardMessageRequest{
				ToUsername: to,
				Content:    fmt.Sprintf("m %d", i),
			})
			if err != nil {
				atomic.AddUint64(&stats.Errors, 1)
				return err
			}
			atomic.AddUint64(&stats.Sent, 1)
			i++
		}
	}
}
