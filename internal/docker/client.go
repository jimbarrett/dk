package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type Container struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Status  string   `json:"status"`
	State   string   `json:"state"`
	Ports   []string `json:"ports"`
	Project string   `json:"project"`
	Service string   `json:"service"`
	Display string   `json:"display"`
}

type Image struct {
	ID      string `json:"id"`
	Repo    string `json:"repo"`
	Tag     string `json:"tag"`
	Size    string `json:"size"`
	Created string `json:"created"`
}

type ProjectGroup struct {
	Name       string      `json:"name"`
	Running    int         `json:"running"`
	Total      int         `json:"total"`
	Containers []Container `json:"containers"`
}

type GroupedContainers struct {
	Projects  []ProjectGroup `json:"projects"`
	Ungrouped []Container    `json:"ungrouped"`
}

// Run executes a docker command and returns stdout.
func Run(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// Passthrough runs a docker command attached to the terminal.
func Passthrough(args ...string) int {
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

// ListContainers returns all containers as structured data.
func ListContainers(all bool) ([]Container, error) {
	format := "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.Ports}}\t{{.Label \"com.docker.compose.project\"}}\t{{.Label \"com.docker.compose.service\"}}"
	args := []string{"ps", "--format", format}
	if all {
		args = append(args, "-a")
	}

	output, err := Run(args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	var containers []Container
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "\t", 8)
		if len(parts) < 8 {
			continue
		}

		project := parts[6]
		service := parts[7]
		display := parts[1]
		if service != "" {
			display = service
		}

		containers = append(containers, Container{
			ID:      parts[0],
			Name:    parts[1],
			Image:   parts[2],
			Status:  parts[3],
			State:   parts[4],
			Ports:   parsePorts(parts[5]),
			Project: project,
			Service: service,
			Display: display,
		})
	}

	return containers, nil
}

// ListGrouped returns containers grouped by Compose project.
func ListGrouped(all bool) (*GroupedContainers, error) {
	containers, err := ListContainers(all)
	if err != nil {
		return nil, err
	}

	projectMap := make(map[string][]Container)
	var ungrouped []Container

	for _, c := range containers {
		if c.Project != "" {
			projectMap[c.Project] = append(projectMap[c.Project], c)
		} else {
			ungrouped = append(ungrouped, c)
		}
	}

	var projects []ProjectGroup
	for name, ctrs := range projectMap {
		running := 0
		for _, c := range ctrs {
			if c.State == "running" {
				running++
			}
		}
		projects = append(projects, ProjectGroup{
			Name:       name,
			Running:    running,
			Total:      len(ctrs),
			Containers: ctrs,
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return &GroupedContainers{Projects: projects, Ungrouped: ungrouped}, nil
}

// FindContainers returns containers matching a partial name.
func FindContainers(search string, runningOnly bool) ([]Container, error) {
	containers, err := ListContainers(!runningOnly)
	if err != nil {
		return nil, err
	}

	search = strings.ToLower(search)
	var matches []Container
	for _, c := range containers {
		if strings.Contains(strings.ToLower(c.Name), search) ||
			(c.Service != "" && strings.Contains(strings.ToLower(c.Service), search)) {
			matches = append(matches, c)
		}
	}
	return matches, nil
}

// Start starts a container.
func Start(name string) (string, error) {
	return Run("start", name)
}

// Stop stops a container.
func Stop(name string) (string, error) {
	return Run("stop", name)
}

// Restart restarts a container.
func Restart(name string) (string, error) {
	return Run("restart", name)
}

// Remove removes a container.
func Remove(name string, force bool) (string, error) {
	if force {
		return Run("rm", "-f", name)
	}
	return Run("rm", name)
}

// RemoveImage removes an image.
func RemoveImage(id string, force bool) (string, error) {
	if force {
		return Run("rmi", "-f", id)
	}
	return Run("rmi", id)
}

// ListImages returns all images.
func ListImages() ([]Image, error) {
	format := "{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}"
	output, err := Run("images", "--format", format)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	var images []Image
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 5 {
			continue
		}
		images = append(images, Image{
			ID:      parts[0],
			Repo:    parts[1],
			Tag:     parts[2],
			Size:    parts[3],
			Created: parts[4],
		})
	}
	return images, nil
}

// Clean prunes stopped containers, dangling images, and unused volumes.
func Clean() (string, error) {
	return Run("system", "prune", "-f", "--volumes")
}

// DiskUsage returns docker system df output.
func DiskUsage() (string, error) {
	return Run("system", "df")
}

// DetectShell checks if bash is available in a container, falls back to sh.
func DetectShell(container string) string {
	result, err := Run("exec", container, "which", "bash")
	if err == nil && strings.Contains(result, "/bash") {
		return "bash"
	}
	return "sh"
}

// Shell opens an interactive shell in a container.
func Shell(container string) int {
	shell := DetectShell(container)
	return Passthrough("exec", "-it", container, shell)
}

// Logs tails logs for a container.
func Logs(container string, follow bool, tail int) int {
	args := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, container)
	return Passthrough(args...)
}

// ShortName returns a display-friendly name for a container within a group.
func ShortName(c Container) string {
	if c.Service != "" {
		return c.Service
	}
	return c.Name
}

var portRe = regexp.MustCompile(`(?:\d+\.\d+\.\d+\.\d+|:::?)?:?(\d+)->(\d+)`)

func parsePorts(raw string) []string {
	if raw == "" {
		return nil
	}

	seen := make(map[string]bool)
	var ports []string
	for _, mapping := range strings.Split(raw, ", ") {
		matches := portRe.FindStringSubmatch(mapping)
		if len(matches) >= 3 {
			clean := matches[1] + "->" + matches[2]
			if !seen[clean] {
				seen[clean] = true
				ports = append(ports, clean)
			}
		}
	}
	return ports
}
