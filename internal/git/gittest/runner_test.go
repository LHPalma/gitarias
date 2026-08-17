package gittest

import "testing"

func TestRunnerRecordsTheInput(t *testing.T) {
	runner := NewRunner(map[string]Response{
		"check-ignore -z --stdin -v": {Output: ".gitignore\x001\x00*.log\x00app.log"},
	})

	output, err := runner.RunWithInput(t.Context(), "app.log\x00", "check-ignore", "-z", "--stdin", "-v")

	if err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}
	if output == "" {
		t.Error("a resposta roteirizada deveria ter voltado")
	}
	if got := runner.Inputs["check-ignore -z --stdin -v"]; got != "app.log\x00" {
		t.Errorf("stdin registrado = %q, queria o que foi passado", got)
	}
}

func TestRunnerRecordsInputCallsAmongTheOthers(t *testing.T) {
	runner := NewRunner(map[string]Response{"ls-files": {Output: ""}})

	if _, err := runner.RunWithInput(t.Context(), "", "ls-files"); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(runner.Calls) != 1 || runner.Calls[0] != "ls-files" {
		t.Errorf("Calls = %v, queria a chamada registrada como qualquer outra", runner.Calls)
	}
}

func TestRunnerRejectsUnscriptedCommandWithInput(t *testing.T) {
	runner := NewRunner(map[string]Response{})

	if _, err := runner.RunWithInput(t.Context(), "", "check-ignore"); err == nil {
		t.Error("comando nao roteirizado tem de virar erro, senao o teste passa de graca")
	}
}

func TestRunnerRecordsTheEnv(t *testing.T) {
	runner := NewRunner(map[string]Response{"commit --amend": {Output: ""}})

	env := []string{"GIT_AUTHOR_NAME=Fake Name", "GIT_AUTHOR_EMAIL=fake@fake.com"}

	if _, err := runner.RunWithEnv(t.Context(), env, "commit", "--amend"); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	got := runner.Envs["commit --amend"]
	if len(got) != 2 || got[0] != env[0] || got[1] != env[1] {
		t.Errorf("ambiente registrado = %v, queria %v", got, env)
	}
}

func TestRunnerRecordsEnvCallsAmongTheOthers(t *testing.T) {
	runner := NewRunner(map[string]Response{"ls-files": {Output: ""}})

	if _, err := runner.RunWithEnv(t.Context(), nil, "ls-files"); err != nil {
		t.Fatalf("nao esperava erro, veio %v", err)
	}

	if len(runner.Calls) != 1 || runner.Calls[0] != "ls-files" {
		t.Errorf("Calls = %v, queria a chamada registrada como qualquer outra", runner.Calls)
	}
}

func TestRunnerRejectsUnscriptedCommandWithEnv(t *testing.T) {
	runner := NewRunner(map[string]Response{})

	if _, err := runner.RunWithEnv(t.Context(), nil, "commit", "--amend"); err == nil {
		t.Error("comando nao roteirizado tem de virar erro, senao o teste passa de graca")
	}
}
