package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gmuxapp/gmux/cli/gmux-run/internal/metadata"
	"github.com/gmuxapp/gmux/cli/gmux-run/internal/naming"
)

func main() {
	log.SetPrefix("gmux-run: ")
	log.SetFlags(0)

	kind := flag.String("adapter", "pi", "session adapter kind (pi, generic, opencode)")
	title := flag.String("title", "", "optional session title")
	cwd := flag.String("cwd", "", "working directory (default: current)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		// Default to adapter name as command for pi/opencode
		if *kind == "pi" {
			args = []string{"pi"}
		} else {
			log.Fatal("no command specified")
		}
	}

	workDir := *cwd
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			log.Fatalf("cannot determine cwd: %v", err)
		}
	}

	abducoName := naming.AbducoName(*kind, workDir)
	sessionID := naming.SessionID()

	sessionTitle := *title
	if sessionTitle == "" {
		sessionTitle = strings.Join(args, " ")
	}

	// Build full command: abduco -n <name> <cmd...>
	abducoArgs := append([]string{"-n", abducoName}, args...)

	abducoPath, err := exec.LookPath("abduco")
	if err != nil {
		log.Fatalf("abduco not found in PATH: %v", err)
	}

	// Write initial metadata
	meta := metadata.New(sessionID, abducoName, *kind, workDir, args)
	meta.SessionFile = "" // filled by adapter if relevant
	if err := meta.Write(); err != nil {
		log.Fatalf("failed to write metadata: %v", err)
	}

	fmt.Printf("session: %s\n", sessionID)
	fmt.Printf("abduco:  %s\n", abducoName)
	fmt.Printf("command: %s\n", strings.Join(args, " "))

	// Launch abduco
	cmd := exec.Command(abducoPath, abducoArgs...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"GMUX_SESSION_ID="+sessionID,
		"GMUX_ABDUCO_NAME="+abducoName,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		meta.SetError(fmt.Sprintf("failed to start: %v", err))
		log.Fatalf("failed to start abduco: %v", err)
	}

	meta.SetRunning(cmd.Process.Pid)

	// Forward signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(sig)
		}
	}()

	// Wait for exit
	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			meta.SetError(fmt.Sprintf("wait error: %v", err))
			meta.Cleanup()
			os.Exit(1)
		}
	}

	meta.SetExited(exitCode)
	// Leave metadata for gmuxd to discover; it handles cleanup policy
	fmt.Printf("exited:  %d\n", exitCode)
	os.Exit(exitCode)
}
