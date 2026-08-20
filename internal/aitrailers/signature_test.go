package aitrailers

import "testing"

func TestSignatureClaude(t *testing.T) {
	trailer, err := Signature("claude")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := Trailer{Key: "Co-Authored-By", Value: "Claude <noreply@anthropic.com>", Tool: "Claude Code"}
	if trailer != want {
		t.Errorf("trailer = %+v, queria %+v", trailer, want)
	}
}

func TestSignatureCopilot(t *testing.T) {
	trailer, err := Signature("copilot")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if trailer.Tool != "GitHub Copilot" || trailer.Key != "Co-Authored-By" {
		t.Errorf("trailer = %+v", trailer)
	}
}

func TestSignatureRoundTripsThroughDetection(t *testing.T) {
	for _, name := range []string{"claude", "copilot"} {
		trailer, err := Signature(name)
		if err != nil {
			t.Fatalf("%s: não esperava erro, veio %v", name, err)
		}

		got, matched := tool(trailer.Key, trailer.Value)
		if !matched {
			t.Fatalf("%s: o trailer fabricado tem de bater na própria detecção, mas não bateu: %+v", name, trailer)
		}
		if got != trailer.Tool {
			t.Errorf("%s: tool() devolveu %q, queria %q", name, got, trailer.Tool)
		}
	}
}

func TestSignatureRejectsAnUnknownTool(t *testing.T) {
	if _, err := Signature("chatgpt"); err == nil {
		t.Fatal("ferramenta desconhecida tem de virar erro")
	}
}
