package cmd

import (
	"bytes"
	"context"
	"errors"

	"github.com/wungjyan/aicommit/internal/ai"
	"github.com/wungjyan/aicommit/internal/config"
)

// fakeGit is an in-memory GitService that records commits and returns canned
// results, so tests never touch a real repository.
type fakeGit struct {
	isRepoErr error
	diff      string
	diffErr   error

	commitErr error
	committed []string
}

func (f *fakeGit) IsGitRepo() error { return f.isRepoErr }

func (f *fakeGit) GetStagedDiff() (string, error) {
	if f.diffErr != nil {
		return "", f.diffErr
	}
	return f.diff, nil
}

func (f *fakeGit) Commit(message string) error {
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = append(f.committed, message)
	return nil
}

// fakeConfig returns a canned configuration and records saves.
type fakeConfig struct {
	cfg     config.Config
	err     error
	saveErr error

	saved   []config.Config
	path    string
	pathErr error
}

func (f *fakeConfig) Load() (config.Config, error) { return f.cfg, f.err }

func (f *fakeConfig) Save(cfg config.Config) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, cfg)
	f.cfg = cfg
	return nil
}

func (f *fakeConfig) Path() (string, error) {
	if f.pathErr != nil {
		return "", f.pathErr
	}
	if f.path == "" {
		return "/tmp/aicommit/config.json", nil
	}
	return f.path, nil
}

// fakeBackend is a stub BackendService for command tests.
type fakeBackend struct {
	checkErr error
	status   ai.CLIStatus
}

func (f fakeBackend) Check(ctx context.Context, cfg config.Config) error { return f.checkErr }
func (f fakeBackend) Status(cfg config.Config) ai.CLIStatus              { return f.status }

// fakeProvider yields queued messages and records how many times Generate ran.
type fakeProvider struct {
	messages []string
	genErr   error
	calls    int
	diffs    []string
}

func (f *fakeProvider) Generate(ctx context.Context, diff string) (string, error) {
	f.diffs = append(f.diffs, diff)
	if f.genErr != nil {
		return "", f.genErr
	}
	msg := ""
	if f.calls < len(f.messages) {
		msg = f.messages[f.calls]
	} else if len(f.messages) > 0 {
		msg = f.messages[len(f.messages)-1]
	}
	f.calls++
	return msg, nil
}

// fakeFactory returns a preconfigured provider or a build error.
type fakeFactory struct {
	provider Provider
	err      error
}

func (f fakeFactory) New(cfg config.Config) (Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.provider, nil
}

// recordingUI captures status output for assertions instead of writing to a
// terminal. Spinner just runs the function inline.
type recordingUI struct {
	success []string
	errs    []string
	warns   []string
	infos   []string
}

func (u *recordingUI) Success(msg string) { u.success = append(u.success, msg) }
func (u *recordingUI) Error(msg string)   { u.errs = append(u.errs, msg) }
func (u *recordingUI) Warn(msg string)    { u.warns = append(u.warns, msg) }
func (u *recordingUI) Info(msg string)    { u.infos = append(u.infos, msg) }
func (u *recordingUI) Spinner(label string, fn func() error) error {
	return fn()
}

// scriptedConfirm replays a fixed sequence of confirmation actions.
type scriptedConfirm struct {
	actions []confirmStep
	calls   int
}

type confirmStep struct {
	action string
	edited string
	err    error
}

func (c *scriptedConfirm) Confirm(message string, valid bool) (string, string, error) {
	if c.calls >= len(c.actions) {
		return "quit", "", nil
	}
	step := c.actions[c.calls]
	c.calls++
	if step.err != nil {
		return "", "", step.err
	}
	edited := step.edited
	if step.action == "commit" && edited == "" {
		edited = message
	}
	return step.action, edited, nil
}

// testDeps assembles a Dependencies with fakes and in-memory I/O buffers.
// Callers tweak the returned struct before building the command.
func testDeps() (Dependencies, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	deps := Dependencies{
		In:       new(bytes.Buffer),
		Out:      &out,
		ErrOut:   &errOut,
		Git:      &fakeGit{},
		Config:   &fakeConfig{},
		Provider: fakeFactory{provider: &fakeProvider{messages: []string{"feat: add thing"}}},
		Backend:  fakeBackend{},
		UI:       &recordingUI{},
		Confirm:  &scriptedConfirm{},
		Version:  VersionInfo{Version: "1.2.3", Commit: "abc1234", Date: "2026-07-16T00:00:00Z"},
	}
	return deps, &out, &errOut
}

// errBoom is a generic sentinel for failure-path tests.
var errBoom = errors.New("boom")
