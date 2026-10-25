package script

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReplyAndReturn(t *testing.T) {
	returned, err := Run(context.Background(), Input{Kind: KindLua, Source: `return "hello"`})
	if err != nil {
		t.Fatal(err)
	}
	if len(returned) != 1 || returned[0].Op != OpReply || returned[0].Value != "hello" {
		t.Fatalf("return: %+v", returned)
	}
	called, err := Run(context.Background(), Input{Kind: KindLua, Source: `reply("hello")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(called) != 1 || called[0].Op != OpReply || called[0].Value != "hello" {
		t.Fatalf("reply: %+v", called)
	}
	_, err = Run(context.Background(), Input{Kind: KindLua, Source: `reply("one") reply("two")`})
	if err == nil || !strings.Contains(err.Error(), "once") {
		t.Fatalf("second reply: %v", err)
	}
}

func TestLiquidFillsAuthorName(t *testing.T) {
	effects, err := Run(context.Background(), Input{
		Kind: KindLua, Source: `reply(liquid("Hi {{ author.name }} in {{ guild.name }} {{ args }} {{ prefix }}"))`,
		AuthorName: "Ada", GuildName: "Canopy", ArgsText: "hello", Prefix: "!",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Value != "Hi Ada in Canopy hello !" {
		t.Fatalf("%+v", effects)
	}
	_, err = Run(context.Background(), Input{Kind: KindLua, Source: `liquid('{% include "secret.liquid" %}')`})
	if err == nil {
		t.Fatal("include should not run")
	}
}

func TestAddRoleAllowlist(t *testing.T) {
	allowed := []Role{{ID: "42", Name: "Member"}}
	effects, err := Run(context.Background(), Input{
		Kind: KindLua, Source: `addRole("Member")`, Allowed: allowed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Op != OpAddRole || effects[0].Value != "42" {
		t.Fatalf("%+v", effects)
	}
	effects, err = Run(context.Background(), Input{
		Kind: KindLua, Source: `addRole("Admin")`, Allowed: allowed,
	})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("refused role: %v effects %+v", err, effects)
	}
}

func TestSandboxHidesHostLibraries(t *testing.T) {
	effects, err := Run(context.Background(), Input{
		Kind: KindLua,
		Source: `if os ~= nil or io ~= nil or loadstring ~= nil or load ~= nil or loadfile ~= nil or dofile ~= nil or getfenv ~= nil or setfenv ~= nil or require ~= nil or debug ~= nil then error("sandbox leak") end
return "ok"`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Value != "ok" {
		t.Fatalf("%+v", effects)
	}
}

func TestInfiniteLoopStops(t *testing.T) {
	start := time.Now()
	_, err := Run(context.Background(), Input{Kind: KindLua, Source: `while true do end`})
	if err == nil {
		t.Fatal("loop should stop")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("loop ran for %s", time.Since(start))
	}
}

func TestTextCommandIsNotLua(t *testing.T) {
	start := time.Now()
	effects, err := Run(context.Background(), Input{Kind: KindText, Source: "while true do end"})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("text command was executed")
	}
	if len(effects) != 1 || effects[0].Op != OpReply || effects[0].Value != "while true do end" {
		t.Fatalf("%+v", effects)
	}
}

func TestCompileRejectsSyntaxErrors(t *testing.T) {
	if err := Compile(`return "ok"`); err != nil {
		t.Fatal(err)
	}
	if err := Compile(`function`); err == nil {
		t.Fatal("expected a syntax error")
	}
	start := time.Now()
	if err := Compile(`while true do end`); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("compile executed the script")
	}
}
