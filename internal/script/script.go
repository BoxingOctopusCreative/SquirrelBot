package script

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BoxingOctopus/squirrelbot/internal/domain"
	"github.com/osteele/liquid"
	lua "github.com/yuin/gopher-lua"
)

const (
	KindText = "text"
	KindLua  = "lua"

	OpReply      = "reply"
	OpReact      = "react"
	OpDelete     = "delete"
	OpAddRole    = "addRole"
	OpRemoveRole = "removeRole"

	luaBudget      = 200 * time.Millisecond
	maxRoleChanges = 5
	maxLiquid      = 4000
)

var commandLiquid = func() *liquid.Engine {
	engine := liquid.NewEngine()
	engine.StrictVariables()
	engine.UnregisterTag("include")
	engine.UnregisterTag("render")
	return engine
}()

var hiddenGlobals = []string{
	"load", "loadstring", "loadfile", "dofile", "getfenv", "setfenv",
	"print", "collectgarbage", "module", "require", "newproxy", "_printregs", "_G",
}

var safeGlobals = []string{
	"assert", "error", "ipairs", "next", "pairs", "pcall", "rawequal",
	"rawget", "rawset", "select", "tonumber", "tostring", "type", "unpack", "xpcall",
	"string", "table", "math",
}

// Role is a role a script is allowed to add or remove.
type Role struct {
	ID   string
	Name string
}

// Input is the triggering message and the roles selected for the command.
type Input struct {
	Source        string
	Kind          string
	Prefix        string
	Args          []string
	ArgsText      string
	AuthorID      string
	AuthorName    string
	AuthorDisplay string
	AuthorRoleIDs []string
	ChannelID     string
	GuildID       string
	GuildName     string
	Allowed       []Role
}

// Effect is one action to apply after a script returns.
type Effect struct {
	Op    string
	Value string
}

type runtime struct {
	in          Input
	effects     []Effect
	replied     bool
	reacted     bool
	deleted     bool
	roleChanges int
}

// Compile checks Lua syntax without running the script.
func Compile(source string) error {
	state := newState()
	defer state.Close()
	_, err := state.LoadString(source)
	if err != nil {
		return err
	}
	return nil
}

// Run executes a command. Text commands return their stored body as a reply.
// Lua commands run in a fresh state with no file, process, or package access.
func Run(ctx context.Context, in Input) ([]Effect, error) {
	if in.Kind != KindLua {
		if strings.TrimSpace(in.Source) == "" {
			return nil, nil
		}
		return []Effect{{Op: OpReply, Value: in.Source}}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	state := newState()
	defer state.Close()
	lua.OpenBase(state)
	lua.OpenString(state)
	lua.OpenTable(state)
	lua.OpenMath(state)

	rt := &runtime{in: in}
	api := state.NewTable()
	for _, name := range safeGlobals {
		api.RawSetString(name, state.GetGlobal(name))
	}
	for _, name := range hiddenGlobals {
		state.SetGlobal(name, lua.LNil)
	}
	api.RawSetString("reply", state.NewFunction(rt.reply))
	api.RawSetString("liquid", state.NewFunction(rt.liquid))
	api.RawSetString("react", state.NewFunction(rt.react))
	api.RawSetString("delete", state.NewFunction(rt.deleteMessage))
	api.RawSetString("addRole", state.NewFunction(rt.addRole))
	api.RawSetString("removeRole", state.NewFunction(rt.removeRole))
	api.RawSetString("args", stringTable(state, in.Args))
	api.RawSetString("author", authorTable(state, in))
	api.RawSetString("guild", guildTable(state, in))
	api.RawSetString("channelId", lua.LString(in.ChannelID))
	api.RawSetString("prefix", lua.LString(in.Prefix))

	env := state.NewTable()
	meta := state.NewTable()
	meta.RawSetString("__index", api)
	meta.RawSetString("__newindex", state.NewFunction(readOnlyGlobal))
	meta.RawSetString("__metatable", lua.LString("locked"))
	state.SetMetatable(env, meta)

	state.SetTop(0)
	fn, err := state.LoadString(in.Source)
	if err != nil {
		return nil, err
	}
	state.SetFEnv(fn, env)
	luaCtx, cancel := context.WithTimeout(ctx, luaBudget)
	defer cancel()
	state.SetContext(luaCtx)
	state.Push(fn)
	if err := state.PCall(0, lua.MultRet, nil); err != nil {
		return nil, err
	}
	if !rt.replied && state.GetTop() >= 1 && state.Get(1).Type() == lua.LTString {
		text := state.ToString(1)
		if strings.TrimSpace(text) != "" {
			if utf8.RuneCountInString(text) > domain.MaxCustomResponse {
				return nil, errors.New("reply must be 1500 characters or fewer")
			}
			rt.effects = append(rt.effects, Effect{Op: OpReply, Value: text})
		}
	}
	return rt.effects, nil
}

// ErrorText flattens an error into one short line for an admin.
func ErrorText(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.Join(strings.Fields(message), " ")
	if utf8.RuneCountInString(message) > 180 {
		message = string([]rune(message)[:180])
	}
	return message
}

func newState() *lua.LState {
	return lua.NewState(lua.Options{
		CallStackSize:       128,
		RegistrySize:        1024,
		RegistryMaxSize:     1024,
		SkipOpenLibs:        true,
		IncludeGoStackTrace: false,
	})
}

func readOnlyGlobal(state *lua.LState) int {
	state.RaiseError("globals are read-only")
	return 0
}

func (rt *runtime) reply(state *lua.LState) int {
	if rt.replied {
		state.RaiseError("reply can be called once")
		return 0
	}
	text := state.CheckString(1)
	if strings.TrimSpace(text) == "" {
		state.RaiseError("reply text is empty")
		return 0
	}
	if utf8.RuneCountInString(text) > domain.MaxCustomResponse {
		state.RaiseError("reply must be 1500 characters or fewer")
		return 0
	}
	rt.replied = true
	rt.effects = append(rt.effects, Effect{Op: OpReply, Value: text})
	return 0
}

func (rt *runtime) liquid(state *lua.LState) int {
	template := state.CheckString(1)
	if utf8.RuneCountInString(template) > maxLiquid {
		state.RaiseError("liquid template is too long")
		return 0
	}
	rendered, err := commandLiquid.ParseAndRenderString(template, commandBindings(rt.in))
	if err != nil {
		state.RaiseError("%s", ErrorText(err))
		return 0
	}
	if utf8.RuneCountInString(rendered) > domain.MaxCustomResponse {
		state.RaiseError("liquid result is too long")
		return 0
	}
	state.Push(lua.LString(rendered))
	return 1
}

func (rt *runtime) react(state *lua.LState) int {
	if rt.reacted {
		state.RaiseError("react can be called once")
		return 0
	}
	emoji, err := normalizeEmoji(state.CheckString(1))
	if err != nil {
		state.RaiseError("%s", err.Error())
		return 0
	}
	rt.reacted = true
	rt.effects = append(rt.effects, Effect{Op: OpReact, Value: emoji})
	return 0
}

func (rt *runtime) deleteMessage(state *lua.LState) int {
	if rt.deleted {
		state.RaiseError("delete can be called once")
		return 0
	}
	rt.deleted = true
	rt.effects = append(rt.effects, Effect{Op: OpDelete})
	return 0
}

func (rt *runtime) addRole(state *lua.LState) int {
	return rt.changeRole(state, OpAddRole)
}

func (rt *runtime) removeRole(state *lua.LState) int {
	return rt.changeRole(state, OpRemoveRole)
}

func (rt *runtime) changeRole(state *lua.LState, op string) int {
	if rt.roleChanges >= maxRoleChanges {
		state.RaiseError("a command can change at most 5 roles")
		return 0
	}
	id, err := resolveRole(state.CheckString(1), rt.in.Allowed)
	if err != nil {
		state.RaiseError("%s", err.Error())
		return 0
	}
	rt.roleChanges++
	rt.effects = append(rt.effects, Effect{Op: op, Value: id})
	return 0
}

func commandBindings(in Input) map[string]any {
	return map[string]any{
		"author": map[string]any{
			"name":         in.AuthorName,
			"id":           in.AuthorID,
			"display_name": in.AuthorDisplay,
		},
		"args": in.ArgsText,
		"guild": map[string]any{
			"name": in.GuildName,
		},
		"prefix": in.Prefix,
	}
}

func resolveRole(token string, allowed []Role) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("name a role")
	}
	for _, role := range allowed {
		if role.ID == token {
			return role.ID, nil
		}
	}
	id := ""
	matches := 0
	for _, role := range allowed {
		if strings.EqualFold(role.Name, token) {
			id = role.ID
			matches++
		}
	}
	if matches == 1 {
		return id, nil
	}
	if matches > 1 {
		return "", errors.New("that role name matches more than one allowed role")
	}
	return "", errors.New("that role is not allowed for this command")
}

func normalizeEmoji(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "<") && strings.HasSuffix(raw, ">") && len(raw) >= 2 {
		raw = raw[1 : len(raw)-1]
	}
	raw = strings.TrimPrefix(raw, ":")
	if raw == "" || len(raw) > 64 || strings.ContainsAny(raw, " \n\t") {
		return "", errors.New("use an emoji, or a custom emoji as name:id")
	}
	return raw, nil
}

func stringTable(state *lua.LState, values []string) *lua.LTable {
	table := state.NewTable()
	for i, value := range values {
		table.RawSetInt(i+1, lua.LString(value))
	}
	return table
}

func authorTable(state *lua.LState, in Input) *lua.LTable {
	table := state.NewTable()
	table.RawSetString("id", lua.LString(in.AuthorID))
	table.RawSetString("name", lua.LString(in.AuthorName))
	table.RawSetString("displayName", lua.LString(in.AuthorDisplay))
	table.RawSetString("roleIds", stringTable(state, in.AuthorRoleIDs))
	return table
}

func guildTable(state *lua.LState, in Input) *lua.LTable {
	table := state.NewTable()
	table.RawSetString("id", lua.LString(in.GuildID))
	table.RawSetString("name", lua.LString(in.GuildName))
	return table
}
