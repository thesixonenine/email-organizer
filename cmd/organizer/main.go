package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"email-organizer/internal/api"
	"email-organizer/internal/config"
	"email-organizer/internal/engine"
	"email-organizer/internal/mail"
	"email-organizer/internal/mail/types"
	"email-organizer/internal/models"
	"email-organizer/internal/scheduler"
	"email-organizer/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	// 1. Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	slog.Info("config loaded", "mailboxes", len(cfg.Mailboxes), "rules", len(cfg.Rules))

	// 2. Initialize database
	db, err := store.InitDB("data/email-organizer.db")
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer db.Close()

	// 3. Initialize repositories
	mailboxRepo := store.NewMailboxRepository(db)
	msgRepo := store.NewMessageRepository(db)
	ruleRepo := store.NewRuleRepository(db)

	// 4. Sync config mailboxes to DB
	for _, mbCfg := range cfg.Mailboxes {
		mailboxRepo.Upsert(&models.Mailbox{
			Name:         mbCfg.Name,
			Protocol:     mbCfg.Protocol,
			Email:        mbCfg.Email,
			IMAPServer:   mbCfg.IMAPServer,
			IMAPPort:     mbCfg.IMAPPort,
			ExchangeType: mbCfg.ExchangeType,
			EWSEndpoint:  mbCfg.EWSEndpoint,
			TenantID:     mbCfg.TenantID,
			ClientID:     mbCfg.ClientID,
			FetchCount:   mbCfg.FetchCount,
			IsActive:     true,
		})
	}

	// 5. Sync config rules to DB
	for _, rCfg := range cfg.Rules {
		condJSON, _ := json.Marshal(rCfg.Conditions)
		actionJSON, _ := json.Marshal(rCfg.Action)
		ruleRepo.Create(&models.Rule{
			Name:          rCfg.Name,
			Enabled:       rCfg.Enabled,
			Trigger:       rCfg.Trigger,
			ConditionJSON: string(condJSON),
			ActionJSON:    string(actionJSON),
		})
	}

	// 6. Initialize rule engine
	ruleEngine := engine.NewRuleEngine(ruleRepo)

	// 7. Mail client factory with caching
	type cachedClient struct {
		client types.MailClient
	}
	clientCache := make(map[int64]*cachedClient)
	mailFactory := func(mailboxID int64) (types.MailClient, error) {
		if cached, ok := clientCache[mailboxID]; ok && cached.client != nil {
			return cached.client, nil
		}
		mb, err := mailboxRepo.GetByID(mailboxID)
		if err != nil {
			return nil, err
		}
		// Find matching config for auth code (not stored in DB for security)
		var authCode string
		for _, mbCfg := range cfg.Mailboxes {
			if mbCfg.Email == mb.Email {
				authCode = mbCfg.AuthCode
				break
			}
		}
		if authCode == "" {
			return nil, fmt.Errorf("auth code not found for mailbox %s", mb.Email)
		}
		client, err := mail.NewMailClient(config.MailboxConfig{
			Name:         mb.Name,
			Protocol:     mb.Protocol,
			Email:        mb.Email,
			AuthCode:     authCode,
			IMAPServer:   mb.IMAPServer,
			IMAPPort:     mb.IMAPPort,
			ExchangeType: mb.ExchangeType,
			EWSEndpoint:  mb.EWSEndpoint,
			TenantID:     mb.TenantID,
			ClientID:     mb.ClientID,
		})
		if err != nil {
			return nil, err
		}
		if err := client.Login(); err != nil {
			return nil, err
		}
		clientCache[mailboxID] = &cachedClient{client: client}
		return client, nil
	}

	// 8. Login all mailboxes at startup
	allMailboxes, _ := mailboxRepo.List()
	for _, mbCfg := range cfg.Mailboxes {
		var found *models.Mailbox
		for _, m := range allMailboxes {
			if m.Email == mbCfg.Email {
				found = m
				break
			}
		}
		if found == nil {
			slog.Warn("mailbox not found in DB", "email", mbCfg.Email)
			continue
		}
		if _, err := mailFactory(found.ID); err != nil {
			slog.Warn("mailbox login failed", "email", mbCfg.Email, "error", err)
		} else {
			slog.Info("mailbox logged in", "email", mbCfg.Email, "protocol", mbCfg.Protocol)
		}
	}

	// 9. Setup scheduler - poll function
	pollFunc := func() {
		mailboxes, err := mailboxRepo.List()
		if err != nil {
			slog.Error("list mailboxes for poll", "error", err)
			return
		}
		for _, mb := range mailboxes {
			if !mb.IsActive {
				continue
			}
			client, err := mailFactory(mb.ID)
			if err != nil {
				slog.Error("get mail client for poll", "mailbox", mb.Email, "error", err)
				continue
			}
			since := time.Time{}
			if mb.LastSyncAt != nil {
				since = *mb.LastSyncAt
			}
			messages, err := client.FetchMessages("INBOX", mb.FetchCount, since)
			if err != nil {
				slog.Error("fetch messages", "mailbox", mb.Email, "error", err)
				continue
			}
			slog.Info("fetched messages", "mailbox", mb.Email, "count", len(messages))
			for _, msg := range messages {
				msg.MailboxID = mb.ID
				msgID, err := msgRepo.Insert(msg)
				if err != nil {
					slog.Warn("insert message", "error", err)
					continue
				}
				msg.ID = msgID
				if err := ruleEngine.ProcessMessage(client, mb.ID, msg); err != nil {
					slog.Error("process message", "error", err)
				}
			}
			mailboxRepo.UpdateLastSync(mb.ID, time.Now())
		}
	}
	sched := scheduler.NewScheduler(1 * time.Minute)
	sched.Start(pollFunc)

	// 10. Setup API
	mbHandler := api.NewMailboxHandler(mailboxRepo, msgRepo, mailFactory)
	ruleHandler := api.NewRuleHandler(ruleRepo)
	router := api.NewRouter(mbHandler, ruleHandler)

	// 11. Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("api server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api server error", "error", err)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	sched.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}