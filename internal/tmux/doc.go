// Package tmux provides a thin wrapper around tmux for driving and inspecting
// terminal sessions.
//
// nightshift uses tmux to run AI agents in detached panes and to scrape their
// output. The [Session] type wraps a named tmux session and exposes the
// operations needed for this workflow: [Session.Start] creates the session,
// [Session.SendKeys] sends input, [Session.CapturePane] reads the current pane
// contents, [Session.Resize] sets the pane size, [Session.Kill] tears it down,
// and [Session.WaitForPattern] polls capture-pane until a regex matches or the
// timeout elapses.
//
// Sessions are constructed with [NewSession] and functional [SessionOption]s.
// Command execution is abstracted behind the [CommandRunner] interface (with
// [ExecRunner] as the os/exec-based default) so behaviour can be tested
// without a real tmux. [StripANSI] strips escape codes from captured output,
// and [ErrTmuxNotFound] is returned when tmux is not installed.
package tmux
