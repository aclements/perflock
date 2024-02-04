package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	// The sleep duration must be short enough that the tests don't become a chore
	// to run, but long enough that we can avoid the noise of too much antagonist
	// (OS, other procs, ...) jitter.
	sleepDuration = 500 * time.Millisecond
)

// Integration-style tests. These tests emulate what a command-line user would
// do. E.g.: running a daemon and a number of clients. To do so while minimizing
// the hassle needed to run test (`go test .` is sufficient), the test setup
// uses subprocesses of itself to function as daemon, client and user-supplied
// programs.
//
// The environment variables affecting this behaviour are GO_TEST_MODE and
// GO_TEST_PROGRAM_MODE. See the comments below.
//
// About OS support: perflock effectively only works fully on Linux, as it uses
// OS-specific interfaces (e.g. for the CPU governor). The tests use abstract
// names (starting with @) for the UNIX domain socket to listen on. We don't
// bother skipping for non-Linux, as that will hopefully make it clear what
// should be fixed to those who are interested.
func TestMain(m *testing.M) {
	switch os.Getenv("GO_TEST_MODE") {
	case "": // Run tests (top-level).
		os.Exit(m.Run())

	case "perflock": // Act like a perflock.
		// If GO_TEST_PROGRAM_MODE is set, we're a perflock client, and main() will
		// be spawn os.Args[0] again to run in program mode. To activate program
		// mode on the next execution, set GO_TEST_MODE=program.
		if os.Getenv("GO_TEST_PROGRAM_MODE") != "" {
			os.Setenv("GO_TEST_MODE", "program")
		}
		// Remove the flags registered by the testing package.
		//
		// NOTE: flag.Parse() has not been called yet (we're in TestMain), according
		// to https://pkg.go.dev/testing
		flag.CommandLine = flag.NewFlagSet("dummy", flag.ExitOnError) // This call is required because otherwise flags panics, if args are set between flag.Parse calls
		main()

	case "program": // Act like a "normal" program.
		switch pmode := os.Getenv("GO_TEST_PROGRAM_MODE"); pmode {
		case "sleeper":
			sleeper()
		default:
			log.Fatalf("unknown program mode %q", pmode)
		}
	}
}

func waitForDaemon(t *testing.T, ctx context.Context, socket string) bool {
	for {
		c, err := net.Dial("unix", socket)
		if err != nil {
			// TODO(aktau): Deal with errors that aren't not found?
			select {
			case <-ctx.Done():
				return false
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
		c.Close()
		return true
	}
}

func TestExclusive(t *testing.T) {
	t.Parallel()

	socket := socketName(t)

	// 1. Start a daemon.
	mustStartDaemon(t, socket)

	// 2. Start three sleepers in EXCLUSIVE mode, each sleeping for 0.5s.
	start := time.Now()
	var sleepers [3]*exec.Cmd
	for i := range sleepers {
		sleepers[i] = mustStartSleeper(t, socket)
	}

	// 3. Wait for them all to finish.
	for _, sleeper := range sleepers {
		sleeper.Wait()
	}

	// Assert that they ran sequentially by making sure it took longer than
	// sleep_time*num_sleepers.
	if got, want := time.Since(start), time.Duration(len(sleepers))*sleepDuration; got < want {
		t.Errorf("expected %d sleepers each sleeping %v to run sequentially, but time passed is %v",
			len(sleepers), sleepDuration, got)
	}
}

func TestShared(t *testing.T) {
	t.Parallel()

	socket := socketName(t)

	// 1. Start a daemon.
	mustStartDaemon(t, socket)

	// 2. Start three sleepers in SHARED mode, each sleeping for 0.5s.
	start := time.Now()
	var sleepers [3]*exec.Cmd
	for i := range sleepers {
		sleepers[i] = mustStartSleeper(t, socket, "-shared")
	}

	for _, sleeper := range sleepers {
		sleeper.Wait()
	}

	// Assert that they ran concurrently by making sure it was shorter than than
	// sleep_time*num_sleepers.
	if got, maxTime := time.Since(start), time.Duration(len(sleepers))*sleepDuration; got > maxTime {
		t.Errorf("expected %d shared sleepers each sleeping %v to not take as long as them sleeping sequentially, but time passed is %v",
			len(sleepers), sleepDuration, got)
	}
}

// funcname returns the function name of the caller.
func funcname(skip int) string {
	var pcs [1]uintptr
	if runtime.Callers(skip+1, pcs[:]) != 1 {
		return "UNKNOWN"
	}
	fr, _ := runtime.CallersFrames(pcs[:]).Next()
	return fr.Func.Name()
}

// socketName returns a unique socket name per test.
func socketName(t *testing.T) string {
	if runtime.GOOS == "linux" {
		// Abstract sockets are automatically cleaned up when the process that
		// created it (the daemon) exits. Avoids potential complications with the
		// filesystem (read-only fs, ...) and leftovers from ctrl-c'ing the test
		// prematurely.
		return fmt.Sprintf("@perflock.%d.%s", os.Getpid(), funcname(2))
	} else {
		return filepath.Join(t.TempDir(), "perflock.socket")
	}
}

// mustStartSleeper starts a perflock client running a sleeper.
func mustStartSleeper(t *testing.T, socket string, argv ...string) *exec.Cmd {
	t.Helper()
	cmd, err := startProcess(t, append(argv, "-socket="+socket, os.Args[0]), []string{"GO_TEST_MODE=perflock", "GO_TEST_PROGRAM_MODE=sleeper"})
	if err != nil {
		t.Fatalf("could not start sleeper: %v", err)
	}
	return cmd
}

// mustStartDaemon starts a perflock daemon and wait for it to start listening on
// the socket.
func mustStartDaemon(t *testing.T, socket string) {
	t.Helper()
	_, err := startProcess(t, []string{"-socket=" + socket, "-daemon"}, []string{"GO_TEST_MODE=perflock"})
	if err != nil {
		t.Fatalf("could not start daemon: %v", err)
	}
	t.Logf("started daemon... waiting for it to become connectable")
	{
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if !waitForDaemon(t, ctx, socket) {
			t.Fatalf("gave up waiting for daemon")
		}
		t.Logf("daemon started!")
	}
}

func startProcess(t *testing.T, argv []string, env []string) (*exec.Cmd, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, os.Args[0], argv...)
	cmd.WaitDelay = 5 * time.Second // Ensure cleanup if the process refuses to exit after being signaled by cancelling the context.
	cmdReader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(cmdReader)
	var pid int
	go func() {
		envs := strings.Join(env, " ")
		for scanner.Scan() {
			t.Logf("[%12d] %-25s %s\n", pid, envs, scanner.Text())
		}
	}()

	cmd.Stderr = cmd.Stdout
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	pid = cmd.Process.Pid
	t.Cleanup(func() {
		cancel()
		if err := cmd.Wait(); err != nil {
			t.Logf("%s %s exited with error: %v", strings.Join(env, " "), os.Args[0], err)
		}
	})
	return cmd, nil
}

func sleeper() {
	log.Printf("GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
	time.Sleep(sleepDuration)
}
