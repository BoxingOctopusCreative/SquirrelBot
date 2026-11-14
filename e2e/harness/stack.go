package harness

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/BoxingOctopus/squirrelbot/internal/api"
	"github.com/BoxingOctopus/squirrelbot/internal/config"
	"github.com/BoxingOctopus/squirrelbot/internal/db"
	"github.com/BoxingOctopus/squirrelbot/internal/discord"
	"github.com/BoxingOctopus/squirrelbot/internal/secretbox"
	"github.com/BoxingOctopus/squirrelbot/internal/store"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options selects where the API listens and which origin the browser uses.
// PublicURL defaults to the listen address. The dashboard dev server sets
// PublicURL to its own origin so same-origin checks match the browser.
type Options struct {
	Addr      string
	PublicURL string
}

// Stack is a migrated database, a Discord stand-in, and the HTTP API.
type Stack struct {
	BaseURL     string
	Origin      string
	Cookie      string
	GuestCookie string
	close       func()
}

// Start boots the stack. Close drops the database.
func Start(ctx context.Context, opt Options) (*Stack, error) {
	dbName, appURL, admin, err := openDatabase(ctx)
	if err != nil {
		return nil, err
	}
	stack := &Stack{}
	var cleanups []func()
	fail := func(err error) (*Stack, error) {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		_ = dropDatabase(context.Background(), admin, dbName)
		return nil, err
	}
	pool, err := db.Connect(ctx, appURL)
	if err != nil {
		return fail(err)
	}
	cleanups = append(cleanups, func() { pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		return fail(err)
	}
	st := store.New(pool)
	if err := st.EnsureGuild(ctx, GuildID, GuildName, ""); err != nil {
		return fail(err)
	}
	creds, err := newCredentials()
	if err != nil {
		return fail(err)
	}
	if err := seedSessions(ctx, st, creds); err != nil {
		return fail(err)
	}

	fake := newDiscord(creds.access, creds.guest, creds.refresh)
	discordServer := http.Server{Handler: fake}
	discordLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fail(err)
	}
	go discordServer.Serve(discordLn)
	cleanups = append(cleanups, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = discordServer.Shutdown(shutdownCtx)
	})
	dest, err := url.Parse("http://" + discordLn.Addr().String())
	if err != nil {
		return fail(err)
	}
	transport := rewriteTransport{dest: dest}
	client := &http.Client{Timeout: 15 * time.Second, Transport: transport}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot, err := discord.New(creds.botToken, st, log)
	if err != nil {
		return fail(err)
	}
	bot.SetReady(true)
	session := bot.Session()
	session.Client = &http.Client{Timeout: 15 * time.Second, Transport: transport}
	session.State.User = &discordgo.User{ID: BotUserID, Username: "squirrelbot", Bot: true}

	addr := opt.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fail(err)
	}
	publicURL := opt.PublicURL
	if publicURL == "" {
		publicURL = "http://" + ln.Addr().String()
	}
	server := api.New(config.Config{
		PublicURL:           publicURL,
		SessionSecret:       creds.session,
		DiscordClientID:     "e2e-client",
		DiscordClientSecret: creds.clientSecret,
		DiscordToken:        creds.botToken,
		WebDir:              "missing",
	}, st, bot, log)
	server.SetHTTPClient(client)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("e2e api", "err", err)
		}
	}()
	base := "http://" + ln.Addr().String()
	if err := waitHealthy(ctx, base); err != nil {
		_ = server.Shutdown()
		return fail(err)
	}
	stack.BaseURL = base
	stack.Origin = config.Config{PublicURL: publicURL}.Origin()
	stack.Cookie = SessionID
	stack.GuestCookie = GuestSessionID
	var poolRef *pgxpool.Pool = pool
	stack.close = func() {
		_ = server.Shutdown()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = discordServer.Shutdown(shutdownCtx)
		poolRef.Close()
		_ = dropDatabase(context.Background(), admin, dbName)
	}
	return stack, nil
}

// Close stops the API and drops the database.
func (s *Stack) Close() {
	if s != nil && s.close != nil {
		s.close()
	}
}

type credentials struct {
	session      string
	access       string
	guest        string
	refresh      string
	clientSecret string
	botToken     string
}

func newCredentials() (credentials, error) {
	values := make([]string, 6)
	for i := range values {
		raw := make([]byte, 24)
		if _, err := rand.Read(raw); err != nil {
			return credentials{}, err
		}
		values[i] = hex.EncodeToString(raw)
	}
	return credentials{
		session: values[0], access: values[1], guest: values[2],
		refresh: values[3], clientSecret: values[4], botToken: values[5],
	}, nil
}

func seedSessions(ctx context.Context, st *store.Store, creds credentials) error {
	key := secretbox.Key(creds.session)
	access, err := secretbox.Seal(key, creds.access)
	if err != nil {
		return err
	}
	refresh, err := secretbox.Seal(key, creds.refresh)
	if err != nil {
		return err
	}
	guestAccess, err := secretbox.Seal(key, creds.guest)
	if err != nil {
		return err
	}
	now := time.Now()
	sessions := []store.Session{
		{
			ID: SessionID, UserID: UserID, Username: UserName,
			AccessToken: access, RefreshToken: refresh,
			TokenExpiresAt: now.Add(time.Hour), ExpiresAt: now.Add(14 * 24 * time.Hour),
		},
		{
			ID: GuestSessionID, UserID: GuestUserID, Username: "Guest",
			AccessToken: guestAccess, RefreshToken: refresh,
			TokenExpiresAt: now.Add(time.Hour), ExpiresAt: now.Add(14 * 24 * time.Hour),
		},
	}
	for _, session := range sessions {
		if err := st.SaveSession(ctx, session); err != nil {
			return fmt.Errorf("save session: %w", err)
		}
	}
	return nil
}

func waitHealthy(ctx context.Context, base string) error {
	deadline := time.Now().Add(5 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
		if err != nil {
			return err
		}
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("health status %d", res.StatusCode)
		} else {
			last = err
		}
		time.Sleep(20 * time.Millisecond)
	}
	if last == nil {
		last = errors.New("api did not start")
	}
	return last
}
