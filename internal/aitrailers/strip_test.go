package aitrailers

import "testing"

func TestStripRemovesOnlyTheAITrailer(t *testing.T) {
	block := "Co-Authored-By: Real Human <human@example.com>\n" +
		"Co-Authored-By: Claude <noreply@anthropic.com>\n" +
		"Claude-Session: https://claude.ai/code/session_123\n"
	raw := "feat: something\n\nbody line one\nbody line two\n\n" + block

	got, changed := Strip(raw, block, block)
	if !changed {
		t.Fatal("esperava changed = true")
	}

	want := "feat: something\n\nbody line one\nbody line two\n\nCo-Authored-By: Real Human <human@example.com>"
	if got != want {
		t.Errorf("mensagem = %q, queria %q", got, want)
	}
}

func TestStripDropsTheWholeBlockWhenEverythingMatches(t *testing.T) {
	block := "Co-Authored-By: Claude <noreply@anthropic.com>\n"
	raw := "feat: something\n\nbody\n\n" + block

	got, changed := Strip(raw, block, block)
	if !changed {
		t.Fatal("esperava changed = true")
	}
	if got != "feat: something\n\nbody" {
		t.Errorf("mensagem = %q", got)
	}
}

func TestStripLeavesTheMessageAloneWithoutAnyTrailer(t *testing.T) {
	raw := "feat: something\n\nbody\n"

	got, changed := Strip(raw, "", "")
	if changed || got != raw {
		t.Errorf("mensagem = %q, changed = %v, sem trailer nada pode mudar", got, changed)
	}
}

func TestStripLeavesTheMessageAloneWhenNoTrailerMatches(t *testing.T) {
	block := "Co-Authored-By: Real Human <human@example.com>\n"
	raw := "feat: something\n\nbody\n\n" + block

	got, changed := Strip(raw, block, block)
	if changed || got != raw {
		t.Errorf("mensagem = %q, changed = %v, trailer humano não pode ser removido nem marcar mudança", got, changed)
	}
}

func TestStripKeepsAMalformedTrailerLineDefensively(t *testing.T) {
	block := "linha sem dois pontos\n"
	raw := "feat: something\n\nbody\n\n" + block

	got, changed := Strip(raw, block, block)
	if changed {
		t.Errorf("changed = true, mas linha malformada não bate em assinatura nenhuma")
	}
	if got != raw {
		t.Errorf("mensagem = %q, sem remoção real nada muda", got)
	}
}
