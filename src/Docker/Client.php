<?php

namespace Dk\Docker;

class Client
{
    /**
     * Run a docker command and return the output.
     */
    public function run(string $command, array $args = []): string
    {
        $escaped = array_map('escapeshellarg', $args);
        $full = 'docker ' . $command . ' ' . implode(' ', $escaped);
        $output = shell_exec($full . ' 2>&1') ?? '';
        return trim($output);
    }

    /**
     * Run a docker command via proc_open (for interactive/passthrough).
     */
    public function passthrough(string $command, array $args = []): int
    {
        $escaped = array_map('escapeshellarg', $args);
        $full = 'docker ' . $command . ' ' . implode(' ', $escaped);

        $proc = proc_open($full, [STDIN, STDOUT, STDERR], $pipes);
        if (!is_resource($proc)) {
            return 1;
        }
        return proc_close($proc);
    }

    /**
     * Get all containers as structured data.
     * Returns array of container arrays with keys: id, name, image, status, state, ports, project, service
     */
    public function listContainers(bool $all = false): array
    {
        $format = '{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.Ports}}\t{{.Label "com.docker.compose.project"}}\t{{.Label "com.docker.compose.service"}}';
        $args = ['--format', $format];
        if ($all) {
            array_unshift($args, '-a');
        }

        $output = $this->run('ps', $args);
        if (empty($output)) {
            return [];
        }

        $containers = [];
        foreach (explode("\n", $output) as $line) {
            $parts = explode("\t", $line);
            if (count($parts) < 8) continue;

            $containers[] = [
                'id'      => $parts[0],
                'name'    => $parts[1],
                'image'   => $parts[2],
                'status'  => $parts[3],
                'state'   => $parts[4],
                'ports'   => $this->parsePorts($parts[5]),
                'project' => $parts[6] ?: null,
                'service' => $parts[7] ?: null,
            ];
        }

        return $containers;
    }

    /**
     * Group containers by Compose project.
     * Returns ['projects' => [name => containers], 'ungrouped' => containers]
     */
    public function listGrouped(bool $all = false): array
    {
        $containers = $this->listContainers($all);
        $projects = [];
        $ungrouped = [];

        foreach ($containers as $container) {
            if ($container['project']) {
                $projects[$container['project']][] = $container;
            } else {
                $ungrouped[] = $container;
            }
        }

        ksort($projects);
        return ['projects' => $projects, 'ungrouped' => $ungrouped];
    }

    /**
     * Find containers matching a partial name.
     */
    public function findContainers(string $search, bool $runningOnly = true): array
    {
        $containers = $this->listContainers(!$runningOnly);
        $search = strtolower($search);

        return array_values(array_filter($containers, function ($c) use ($search) {
            return str_contains(strtolower($c['name']), $search)
                || ($c['service'] && str_contains(strtolower($c['service']), $search));
        }));
    }

    /**
     * Start a container by name or ID.
     */
    public function start(string $container): string
    {
        return $this->run('start', [$container]);
    }

    /**
     * Stop a container by name or ID.
     */
    public function stop(string $container): string
    {
        return $this->run('stop', [$container]);
    }

    /**
     * Restart a container by name or ID.
     */
    public function restart(string $container): string
    {
        return $this->run('restart', [$container]);
    }

    /**
     * Remove a container by name or ID.
     */
    public function remove(string $container, bool $force = false): string
    {
        $args = $force ? ['-f', $container] : [$container];
        return $this->run('rm', $args);
    }

    /**
     * Remove an image by name or ID.
     */
    public function removeImage(string $image, bool $force = false): string
    {
        $args = $force ? ['-f', $image] : [$image];
        return $this->run('rmi', $args);
    }

    /**
     * List images.
     */
    public function listImages(): array
    {
        $format = '{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}';
        $output = $this->run('images', ['--format', $format]);
        if (empty($output)) {
            return [];
        }

        $images = [];
        foreach (explode("\n", $output) as $line) {
            $parts = explode("\t", $line);
            if (count($parts) < 5) continue;
            $images[] = [
                'id'      => $parts[0],
                'repo'    => $parts[1],
                'tag'     => $parts[2],
                'size'    => $parts[3],
                'created' => $parts[4],
            ];
        }

        return $images;
    }

    /**
     * System prune — remove stopped containers, dangling images, unused volumes.
     */
    public function clean(): string
    {
        return $this->run('system prune', ['-f', '--volumes']);
    }

    /**
     * Get system disk usage summary.
     */
    public function diskUsage(): string
    {
        return $this->run('system df', []);
    }

    /**
     * Check if a container has bash available, otherwise fall back to sh.
     */
    public function detectShell(string $container): string
    {
        $result = $this->run('exec', [$container, 'which', 'bash']);
        return str_contains($result, '/bash') ? 'bash' : 'sh';
    }

    /**
     * Shell into a container (interactive passthrough).
     */
    public function shell(string $container): int
    {
        $shellCmd = $this->detectShell($container);
        return $this->passthrough('exec -it', [$container, $shellCmd]);
    }

    /**
     * Tail logs for a container (interactive passthrough).
     */
    public function logs(string $container, bool $follow = false, int $tail = 100): int
    {
        $cmd = 'logs --tail ' . $tail;
        if ($follow) {
            $cmd .= ' -f';
        }
        return $this->passthrough($cmd, [$container]);
    }

    /**
     * Parse Docker port string into a cleaner format.
     * Input like "0.0.0.0:3000->3000/tcp, :::3000->3000/tcp"
     * Returns array of "3000->3000" style strings (deduplicated).
     */
    private function parsePorts(string $raw): array
    {
        if (empty($raw)) return [];

        $ports = [];
        foreach (explode(', ', $raw) as $mapping) {
            // Extract host_port->container_port, strip IP and protocol
            if (preg_match('/(?:\d+\.\d+\.\d+\.\d+|:::?)?:?(\d+)->(\d+)/', $mapping, $m)) {
                $clean = $m[1] . '->' . $m[2];
                $ports[$clean] = true;
            }
        }

        return array_keys($ports);
    }

    /**
     * Get a display-friendly name for a container within a group.
     * Strips the project prefix and trailing -1 suffix.
     */
    public static function shortName(array $container): string
    {
        if ($container['service']) {
            return $container['service'];
        }
        return $container['name'];
    }
}
