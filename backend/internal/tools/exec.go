package tools

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"sync"
)

// ScanLines splits on \r or \n, since progress output from yt-dlp and
// ffmpeg is written with carriage returns rather than newlines, per
// .agents/skills/sm-external-tools.
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// RunStreaming starts cmd, feeding every stdout and stderr line (split on
// \r or \n) to the given callbacks as it runs, then waits for it to exit
// and returns cmd.Wait()'s error. Both pipes are fully drained before
// Wait is called, per .agents/skills/sm-external-tools.
func RunStreaming(cmd *exec.Cmd, onStdout, onStderr func(line string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanLines(stdout, onStdout)
	}()
	go func() {
		defer wg.Done()
		scanLines(stderr, onStderr)
	}()
	wg.Wait()

	return cmd.Wait()
}

func scanLines(r io.Reader, onLine func(line string)) {
	scanner := bufio.NewScanner(r)
	scanner.Split(ScanLines)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
}
