package harness

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// discordFake answers the Discord REST calls the API and bot make during a dashboard session.
type discordFake struct {
	mu          sync.Mutex
	access      string
	guestAccess string
	refresh     string
	channels    map[string]map[string]any
	roles       map[string]map[string]any
	members     map[string]member
	messages    map[string]map[string]any
	nextID      int64
}

type member struct {
	id         string
	username   string
	globalName string
	nick       string
	roles      []string
	bot        bool
}

func newDiscord(access, guestAccess, refresh string) *discordFake {
	roles := map[string]map[string]any{
		GuildID:      roleJSON(GuildID, "@everyone", 0, false),
		MemberRoleID: roleJSON(MemberRoleID, "Members", 10, false),
		ModRoleID:    roleJSON(ModRoleID, "Mods", 20, false),
		AdminRoleID:  roleJSON(AdminRoleID, "Admins", 30, false),
		BotRoleID:    roleJSON(BotRoleID, "SquirrelBot", 50, true),
	}
	channels := map[string]map[string]any{
		GeneralChannelID:  channelJSON(GeneralChannelID, "general"),
		AnnounceChannelID: channelJSON(AnnounceChannelID, "announcements"),
	}
	members := map[string]member{
		UserID:       {id: UserID, username: "acorn", globalName: UserName, roles: []string{AdminRoleID}},
		TargetUserID: {id: TargetUserID, username: TargetUsername, globalName: TargetName, nick: TargetName, roles: []string{MemberRoleID}},
		BotUserID:    {id: BotUserID, username: "squirrelbot", globalName: "squirrelbot", roles: []string{BotRoleID}, bot: true},
	}
	return &discordFake{
		access:      access,
		guestAccess: guestAccess,
		refresh:     refresh,
		channels:    channels,
		roles:       roles,
		members:     members,
		messages:    map[string]map[string]any{},
		nextID:      130000000000000000,
	}
}

func (d *discordFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "api" {
		d.miss(w, r)
		return
	}
	if parts[1] != "v9" {
		d.oauth(w, r, parts[1:])
		return
	}
	d.rest(w, r, parts[2:])
}

func (d *discordFake) oauth(w http.ResponseWriter, r *http.Request, parts []string) {
	switch strings.Join(parts, "/") {
	case "oauth2/token":
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token":  d.access,
			"refresh_token": d.refresh,
			"expires_in":    3600,
		})
	case "users/@me":
		token := bearer(r)
		switch token {
		case d.access:
			writeJSON(w, http.StatusOK, map[string]any{
				"id": UserID, "username": "acorn", "global_name": UserName, "avatar": "",
			})
		case d.guestAccess:
			writeJSON(w, http.StatusOK, map[string]any{
				"id": GuestUserID, "username": "guest", "global_name": "Guest", "avatar": "",
			})
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "401: Unauthorized"})
		}
	case "users/@me/guilds":
		token := bearer(r)
		switch token {
		case d.access:
			writeJSON(w, http.StatusOK, []any{
				guildSummary(GuildID, GuildName, true, "32"),
				guildSummary(HollowGuildID, "Hollow", true, "32"),
			})
		case d.guestAccess:
			writeJSON(w, http.StatusOK, []any{
				guildSummary(GuildID, GuildName, false, "0"),
			})
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "401: Unauthorized"})
		}
	default:
		d.miss(w, r)
	}
}

func (d *discordFake) rest(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		d.miss(w, r)
		return
	}
	switch parts[0] {
	case "guilds":
		d.guildRoute(w, r, parts[1:])
	case "channels":
		d.channelRoute(w, r, parts[1:])
	case "users":
		if len(parts) == 3 && parts[1] == "@me" && parts[2] == "channels" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusOK, channelJSON(d.allocID(), "dm"))
			return
		}
		d.miss(w, r)
	default:
		d.miss(w, r)
	}
}

func (d *discordFake) guildRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		d.miss(w, r)
		return
	}
	guildID := parts[0]
	if guildID != GuildID {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Guild", "code": 10004})
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, d.guildBody())
		return
	}
	if len(parts) < 2 {
		d.miss(w, r)
		return
	}
	switch parts[1] {
	case "channels":
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, d.channelList())
			return
		}
		if r.Method == http.MethodPost {
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := d.allocID()
			item := channelJSON(id, body.Name)
			d.mu.Lock()
			d.channels[id] = item
			d.mu.Unlock()
			writeJSON(w, http.StatusOK, item)
			return
		}
	case "roles":
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, d.roleList())
			return
		}
		if r.Method == http.MethodPost {
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := d.allocID()
			item := roleJSON(id, body.Name, 15, false)
			d.mu.Lock()
			d.roles[id] = item
			d.mu.Unlock()
			writeJSON(w, http.StatusOK, item)
			return
		}
	case "emojis":
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []any{
				map[string]any{"id": EmojiID, "name": "acorn", "animated": false},
			})
			return
		}
	case "members":
		d.memberRoute(w, r, parts[2:])
		return
	}
	d.miss(w, r)
}

func (d *discordFake) memberRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, d.memberPage(r.URL.Query().Get("after")))
		return
	}
	if len(parts) == 0 {
		d.miss(w, r)
		return
	}
	userID := parts[0]
	if userID == "@me" {
		userID = BotUserID
	}
	if len(parts) == 1 && (r.Method == http.MethodGet || r.Method == http.MethodPatch) {
		d.mu.Lock()
		member, ok := d.members[userID]
		d.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Member", "code": 10007})
			return
		}
		if r.Method == http.MethodPatch {
			var body struct {
				Nick *string `json:"nick"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Nick != nil {
				d.mu.Lock()
				member.nick = *body.Nick
				d.members[userID] = member
				d.mu.Unlock()
			}
		}
		writeJSON(w, http.StatusOK, member.discord())
		return
	}
	if len(parts) == 3 && parts[1] == "roles" && (r.Method == http.MethodPut || r.Method == http.MethodDelete) {
		roleID := parts[2]
		d.mu.Lock()
		defer d.mu.Unlock()
		member, ok := d.members[userID]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Member", "code": 10007})
			return
		}
		if r.Method == http.MethodPut {
			if !contains(member.roles, roleID) {
				member.roles = append(member.roles, roleID)
			}
		} else {
			next := member.roles[:0]
			for _, id := range member.roles {
				if id != roleID {
					next = append(next, id)
				}
			}
			member.roles = next
		}
		d.members[userID] = member
		w.WriteHeader(http.StatusNoContent)
		return
	}
	d.miss(w, r)
}

func (d *discordFake) channelRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 2 || parts[1] != "messages" {
		d.miss(w, r)
		return
	}
	channelID := parts[0]
	if len(parts) == 2 && r.Method == http.MethodPost {
		io.Copy(io.Discard, r.Body)
		id := d.allocID()
		item := map[string]any{"id": id, "channel_id": channelID, "content": "ok"}
		d.mu.Lock()
		d.messages[id] = item
		d.mu.Unlock()
		writeJSON(w, http.StatusOK, item)
		return
	}
	if len(parts) < 3 {
		d.miss(w, r)
		return
	}
	messageID := parts[2]
	if len(parts) == 3 && r.Method == http.MethodGet {
		d.mu.Lock()
		item, ok := d.messages[messageID]
		d.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Message", "code": 10008})
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodPatch {
		d.mu.Lock()
		item, ok := d.messages[messageID]
		d.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Message", "code": 10008})
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		d.mu.Lock()
		delete(d.messages, messageID)
		d.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(parts) >= 5 && parts[3] == "reactions" {
		d.mu.Lock()
		_, ok := d.messages[messageID]
		d.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Message", "code": 10008})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	d.miss(w, r)
}

func (d *discordFake) guildBody() map[string]any {
	return map[string]any{
		"id":       GuildID,
		"name":     GuildName,
		"owner_id": OwnerID,
		"roles":    d.roleList(),
	}
}

func (d *discordFake) channelList() []any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]any, 0, len(d.channels))
	for _, item := range d.channels {
		out = append(out, item)
	}
	return out
}

func (d *discordFake) roleList() []any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]any, 0, len(d.roles))
	for _, item := range d.roles {
		out = append(out, item)
	}
	return out
}

func (d *discordFake) memberPage(after string) []any {
	d.mu.Lock()
	defer d.mu.Unlock()
	ids := make([]string, 0, len(d.members))
	for id := range d.members {
		if after == "" || id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, d.members[id].discord())
	}
	return out
}

func (d *discordFake) allocID() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nextID++
	return strconv.FormatInt(d.nextID, 10)
}

func (m member) discord() map[string]any {
	roles := m.roles
	if roles == nil {
		roles = []string{}
	}
	return map[string]any{
		"nick":  m.nick,
		"roles": roles,
		"user": map[string]any{
			"id":          m.id,
			"username":    m.username,
			"global_name": m.globalName,
			"bot":         m.bot,
		},
	}
}

func roleJSON(id, name string, position int, managed bool) map[string]any {
	return map[string]any{
		"id": id, "name": name, "position": position, "managed": managed, "permissions": "0",
	}
}

func channelJSON(id, name string) map[string]any {
	kind := 0
	if name == "dm" {
		kind = 1
	}
	return map[string]any{"id": id, "guild_id": GuildID, "name": name, "type": kind}
}

func guildSummary(id, name string, owner bool, permissions string) map[string]any {
	return map[string]any{
		"id": id, "name": name, "icon": "", "owner": owner, "permissions": permissions,
	}
}

func bearer(r *http.Request) string {
	value := r.Header.Get("Authorization")
	return strings.TrimPrefix(value, "Bearer ")
}

func contains(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func (d *discordFake) miss(w http.ResponseWriter, r *http.Request) {
	log.Printf("discord stand-in: unhandled %s %s", r.Method, r.URL.RequestURI())
	writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown", "code": 0})
}

// rewriteTransport sends Discord HTTP calls to the stand-in.
type rewriteTransport struct {
	base http.RoundTripper
	dest *url.URL
}

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := req
	if req.URL.Hostname() == "discord.com" {
		next = req.Clone(req.Context())
		next.URL.Scheme = rt.dest.Scheme
		next.URL.Host = rt.dest.Host
		next.Host = rt.dest.Host
	}
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(next)
}
