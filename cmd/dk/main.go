package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/barrett/dk/internal/docker"
	"github.com/barrett/dk/internal/server"
	"github.com/barrett/dk/web"
)

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var args []string
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	switch command {
	case "":
		cmdStatus()
	case "ls":
		cmdList(args)
	case "start":
		cmdStart(args)
	case "stop":
		cmdStop(args)
	case "restart":
		cmdRestart(args)
	case "sh":
		cmdShell(args)
	case "logs":
		cmdLogs(args)
	case "ports":
		cmdPorts(args)
	case "rm":
		cmdRemove(args)
	case "rmi":
		cmdRemoveImage(args)
	case "clean":
		cmdClean()
	case "web":
		cmdWeb(args)
	case "_serve":
		cmdServe(args)
	case "help":
		cmdHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'dk help' for usage.\n", command)
		os.Exit(1)
	}
}

// ─── Commands ────────────────────────────────────────────────────────

func cmdStatus() {
	containers, err := docker.ListContainers(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	running := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}
	stopped := len(containers) - running

	images, _ := docker.ListImages()

	fmt.Printf("\033[1mdk — Docker Overview\033[0m\n\n")
	fmt.Printf("  Running:  %d containers\n", running)
	fmt.Printf("  Stopped:  %d containers\n", stopped)
	fmt.Printf("  Images:   %d\n\n", len(images))

	if running > 0 {
		var active []docker.Container
		for _, c := range containers {
			if c.State == "running" {
				active = append(active, c)
			}
		}
		printContainerTable(active)
	}
}

func cmdList(args []string) {
	all := contains(args, "-a")
	grouped := contains(args, "-g")

	if grouped {
		data, err := docker.ListGrouped(all)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		for _, project := range data.Projects {
			color := "\033[33m"
			if project.Running == project.Total {
				color = "\033[32m"
			}
			fmt.Printf("\n\033[1m%s\033[0m %s[%d/%d running]\033[0m\n",
				project.Name, color, project.Running, project.Total)

			for _, c := range project.Containers {
				stateColor := "\033[31m"
				if c.State == "running" {
					stateColor = "\033[32m"
				}
				fmt.Printf("  %s●\033[0m %-18s %-20s %-14s %s\n",
					stateColor, docker.ShortName(c), c.Image, c.Status, strings.Join(c.Ports, ", "))
			}
		}

		if len(data.Ungrouped) > 0 {
			fmt.Printf("\n\033[1mUngrouped\033[0m\n")
			for _, c := range data.Ungrouped {
				stateColor := "\033[31m"
				if c.State == "running" {
					stateColor = "\033[32m"
				}
				fmt.Printf("  %s●\033[0m %-18s %-20s %-14s %s\n",
					stateColor, c.Name, c.Image, c.Status, strings.Join(c.Ports, ", "))
			}
		}
		fmt.Println()
	} else {
		containers, err := docker.ListContainers(all)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if len(containers) == 0 {
			fmt.Println("No containers found.")
			return
		}
		printContainerTable(containers)
	}
}

func cmdStart(args []string) {
	name := resolveContainer(args, false)
	if name == "" {
		return
	}
	docker.Start(name)
	fmt.Printf("\033[32mStarted:\033[0m %s\n", name)
}

func cmdStop(args []string) {
	name := resolveContainer(args, true)
	if name == "" {
		return
	}
	docker.Stop(name)
	fmt.Printf("\033[33mStopped:\033[0m %s\n", name)
}

func cmdRestart(args []string) {
	name := resolveContainer(args, true)
	if name == "" {
		return
	}
	docker.Restart(name)
	fmt.Printf("\033[32mRestarted:\033[0m %s\n", name)
}

func cmdShell(args []string) {
	name := resolveContainer(args, true)
	if name == "" {
		return
	}
	shell := docker.DetectShell(name)
	fmt.Printf("Connecting to \033[1m%s\033[0m (%s)...\n", name, shell)
	code := docker.Shell(name)
	os.Exit(code)
}

func cmdLogs(args []string) {
	follow := contains(args, "-f")
	filtered := filterOut(args, "-f")

	name := resolveContainer(filtered, true)
	if name == "" {
		return
	}
	code := docker.Logs(name, follow, 100)
	os.Exit(code)
}

func cmdPorts(args []string) {
	if len(args) > 0 {
		matches, err := docker.FindContainers(args[0], true)
		if err != nil || len(matches) == 0 {
			fmt.Printf("No running containers matching '%s'.\n", args[0])
			return
		}
		for _, c := range matches {
			ports := strings.Join(c.Ports, ", ")
			if ports == "" {
				ports = "none"
			}
			fmt.Printf("\033[1m%s\033[0m  %s\n", c.Name, ports)
		}
	} else {
		containers, err := docker.ListContainers(false)
		if err != nil || len(containers) == 0 {
			fmt.Println("No running containers.")
			return
		}
		maxName := 0
		for _, c := range containers {
			if len(c.Name) > maxName {
				maxName = len(c.Name)
			}
		}
		for _, c := range containers {
			ports := strings.Join(c.Ports, ", ")
			if ports == "" {
				ports = "none"
			}
			fmt.Printf("%-*s  %s\n", maxName, c.Name, ports)
		}
	}
}

func cmdRemove(args []string) {
	force := contains(args, "-f")
	filtered := filterOut(args, "-f")

	name := resolveContainer(filtered, false)
	if name == "" {
		return
	}
	docker.Remove(name, force)
	fmt.Printf("\033[31mRemoved:\033[0m %s\n", name)
}

func cmdRemoveImage(args []string) {
	force := contains(args, "-f")
	filtered := filterOut(args, "-f")

	if len(filtered) > 0 {
		docker.RemoveImage(filtered[0], force)
		fmt.Println("\033[31mRemoved image.\033[0m")
		return
	}

	images, err := docker.ListImages()
	if err != nil || len(images) == 0 {
		fmt.Println("No images found.")
		return
	}

	fmt.Println("Select an image to remove:")
	for i, img := range images {
		fmt.Printf("  \033[1m%d)\033[0m %s:%s  (%s)\n", i+1, img.Repo, img.Tag, img.Size)
	}

	choice := prompt("> ")
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(images) {
		fmt.Println("Invalid selection.")
		return
	}

	docker.RemoveImage(images[idx-1].ID, force)
	fmt.Println("\033[31mRemoved image.\033[0m")
}

func cmdClean() {
	fmt.Println("Pruning stopped containers, dangling images, and unused volumes...")
	result, err := docker.Clean()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	fmt.Println("\033[32mDone.\033[0m")
}

func cmdWeb(args []string) {
	if len(args) > 0 && args[0] == "stop" {
		cmdWebStop()
		return
	}

	port := "8080"
	if len(args) > 0 {
		port = args[0]
	}

	pidFile := pidFilePath()

	// Check if already running
	if pid, p := readPidFile(pidFile); pid != 0 {
		fmt.Printf("dk web is already running (PID %d) on port %s\n", pid, p)
		fmt.Println("Run 'dk web stop' to stop it.")
		return
	}

	// Fork a background process using exec.Command for proper PID tracking
	exe, _ := os.Executable()
	cmd := exec.Command(exe, "_serve", port)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	// Detach from terminal
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start background server: %s\n", err)
		os.Exit(1)
	}

	pid := cmd.Process.Pid
	cmd.Process.Release()

	// Write PID file from the parent since we know the child PID immediately
	writePidFile(pidFile, pid, port)

	fmt.Printf("dk web UI started on \033[1mhttp://localhost:%s\033[0m (PID %d)\n", port, pid)
	fmt.Println("Run 'dk web stop' to stop it.")
}

func cmdWebStop() {
	pidFile := pidFilePath()
	pid, port := readPidFile(pidFile)
	if pid == 0 {
		fmt.Println("dk web is not running.")
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		fmt.Println("dk web is not running.")
		return
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		os.Remove(pidFile)
		fmt.Println("dk web is not running.")
		return
	}

	os.Remove(pidFile)
	fmt.Printf("Stopped dk web (PID %d, port %s)\n", pid, port)
}

func cmdServe(args []string) {
	port := "8080"
	if len(args) > 0 {
		port = args[0]
	}

	// Clean up PID file on exit
	defer os.Remove(pidFilePath())

	webFS, err := fs.Sub(web.FS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if err := server.Start(port, webFS); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		os.Exit(1)
	}
}

func cmdHelp() {
	fmt.Print(`
dk — Docker management CLI

Usage: dk <command> [options]

Commands:
  (none)      Status overview
  ls          List containers (-a all, -g grouped)
  start       Start a container
  stop        Stop a container
  restart     Restart a container
  sh          Shell into a container
  logs        Tail logs (-f to follow)
  ports       Show port mappings
  rm          Remove a container (-f force)
  rmi         Remove an image (-f force)
  clean       Prune stopped containers, dangling images, volumes
  web         Launch web UI (optional port, default 8080)
  web stop    Stop the web UI
  help        Show this help

Container names support partial matching. Omit the name for interactive selection.
`)
}

// ─── Helpers ─────────────────────────────────────────────────────────

func resolveContainer(args []string, runningOnly bool) string {
	if len(args) > 0 {
		matches, err := docker.FindContainers(args[0], runningOnly)
		if err != nil || len(matches) == 0 {
			scope := ""
			if runningOnly {
				scope = "running "
			}
			fmt.Printf("No %scontainers matching '%s'.\n", scope, args[0])
			return ""
		}
		if len(matches) == 1 {
			return matches[0].Name
		}
		fmt.Printf("Multiple containers match '%s':\n", args[0])
		return pickContainer(matches)
	}

	containers, err := docker.ListContainers(!runningOnly)
	if err != nil || len(containers) == 0 {
		fmt.Println("No containers found.")
		return ""
	}

	fmt.Println("Select a container:")
	return pickContainer(containers)
}

func pickContainer(containers []docker.Container) string {
	for i, c := range containers {
		stateColor := "\033[31m"
		if c.State == "running" {
			stateColor = "\033[32m"
		}
		fmt.Printf("  \033[1m%d)\033[0m %s●\033[0m %s\n", i+1, stateColor, c.Name)
	}

	choice := prompt("> ")
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(containers) {
		fmt.Println("Invalid selection.")
		return ""
	}
	return containers[idx-1].Name
}

func prompt(label string) string {
	fmt.Print(label)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func printContainerTable(containers []docker.Container) {
	maxName := 9
	maxImage := 5
	for _, c := range containers {
		if len(c.Name) > maxName {
			maxName = len(c.Name)
		}
		if len(c.Image) > maxImage {
			maxImage = len(c.Image)
		}
	}

	fmt.Printf("\033[1m%-*s  %-*s  %-14s  %s\033[0m\n", maxName, "CONTAINER", maxImage, "IMAGE", "STATUS", "PORTS")
	for _, c := range containers {
		ports := strings.Join(c.Ports, ", ")
		fmt.Printf("%-*s  %-*s  %-14s  %s\n", maxName, c.Name, maxImage, c.Image, c.Status, ports)
	}
}

func pidFilePath() string {
	dir := os.Getenv("HOME") + "/.local/share/dk"
	os.MkdirAll(dir, 0755)
	return dir + "/dk.pid"
}

func writePidFile(path string, pid int, port string) {
	content := fmt.Sprintf("%d\n%s\n", pid, port)
	os.WriteFile(path, []byte(content), 0644)
}

func readPidFile(path string) (int, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, ""
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return 0, ""
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return 0, ""
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(path)
		return 0, ""
	}
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		os.Remove(path)
		return 0, ""
	}

	return pid, lines[1]
}

func contains(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func filterOut(args []string, flag string) []string {
	var result []string
	for _, a := range args {
		if a != flag {
			result = append(result, a)
		}
	}
	return result
}
