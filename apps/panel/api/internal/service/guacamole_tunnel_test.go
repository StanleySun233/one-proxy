package service

import (
	"bufio"
	"strings"
	"testing"
)

func TestFormatGuacamoleInstructionUsesRuneLength(t *testing.T) {
	got := formatGuacamoleInstruction("name", "测试")
	if got != "4.name,2.测试;" {
		t.Fatalf("formatGuacamoleInstruction() = %q", got)
	}
}

func TestReadGuacamoleInstructionKeepsRawInstruction(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("4.name,2.测试;4.sync,1.1;"))
	instruction, err := readGuacamoleInstruction(reader)
	if err != nil {
		t.Fatalf("readGuacamoleInstruction: %v", err)
	}
	if instruction.Opcode != "name" || len(instruction.Args) != 1 || instruction.Args[0] != "测试" {
		t.Fatalf("instruction = %#v", instruction)
	}
	if instruction.Raw != "4.name,2.测试;" {
		t.Fatalf("raw = %q", instruction.Raw)
	}
	next, err := readGuacamoleInstruction(reader)
	if err != nil {
		t.Fatalf("read next instruction: %v", err)
	}
	if next.Opcode != "sync" || next.Args[0] != "1" {
		t.Fatalf("next = %#v", next)
	}
}

func TestIsGuacamoleInternalPing(t *testing.T) {
	if !isGuacamoleInternalPing("0.,4.ping,13.1710000000000;") {
		t.Fatal("expected internal ping")
	}
	if isGuacamoleInternalPing("4.ping;") {
		t.Fatal("expected normal instruction")
	}
}
