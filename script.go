package utils

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

const (
	ScriptStatusWaiting = "waiting"
	ScriptStatusRunning = "running"
	ScriptStatusExited  = "exited"
	ScriptStatusKilled  = "killed"
	ScriptStatusFailed  = "failed_start"
)

type ScriptSpec struct {
	Path        string
	Dir         string
	Args        []string
	Env         []string
	SysProcAttr *syscall.SysProcAttr // &syscall.SysProcAttr{Setpgid: true}
	Out         string
}

type Script struct {
	Spec     *ScriptSpec
	Cmd      *exec.Cmd
	Status   string
	callback func()
	input    io.WriteCloser
	output   io.ReadCloser
	wg       *sync.WaitGroup
	Runnable
}

func NewScript(s *ScriptSpec, callback func()) (*Script, error) {
	cmd := &exec.Cmd{
		Path:        s.Path,
		Dir:         s.Dir,
		Args:        append([]string{s.Path}, s.Args...),
		Env:         s.Env,
		SysProcAttr: s.SysProcAttr,
	}

	stdinPipe, err1 := cmd.StdinPipe()
	if err1 != nil {
		return nil, err1
	}

	outWriter, err2 := os.OpenFile(s.Out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, MODE_PERM_RW)
	if err2 != nil {
		return nil, err2
	}
	cmd.Stdout = outWriter
	cmd.Stderr = outWriter

	outReader, err3 := os.Open(s.Out)
	if err3 != nil {
		return nil, err3
	}

	script := &Script{
		Spec:     s,
		Cmd:      cmd,
		Status:   ScriptStatusWaiting,
		callback: callback,
		input:    stdinPipe,
		output:   outReader,
		wg:       &sync.WaitGroup{},
	}

	// for script wait
	script.wg.Add(1)

	return script, nil
}

func (s *Script) Run() {
	// start cmd and wait until end
	s.Status = ScriptStatusRunning
	s.Cmd.Run()

	if s.Cmd.Process == nil {
		s.Status = ScriptStatusFailed // start failed
	} else if s.Cmd.ProcessState != nil && s.Cmd.ProcessState.Exited() {
		s.Status = ScriptStatusExited // existed
	} else {
		s.Status = ScriptStatusKilled // killed
	}

	// callback
	if s.callback != nil {
		s.callback()
	}

	// close i/o
	s.input.Close()
	s.output.Close()

	// release wait
	s.wg.Done()
}

func (s *Script) Input(line string) error {
	if _, err := s.input.Write([]byte(line)); err != nil {
		return err
	}
	return nil
}

func (s *Script) Expect(regex string, timeout time.Duration) bool {
	reader := bufio.NewReader(s.output)
	var i time.Duration
	for i = 0; i < timeout; i += 100 * time.Millisecond {
		matched, err := regexp.MatchReader(regex, reader)
		if err != nil {
			return false
		}
		if matched {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func (s *Script) Wait() {
	s.wg.Wait()
}
