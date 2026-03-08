<?php

require_once __DIR__ . '/../src/Docker/Client.php';

use Dk\Docker\Client;

header('Content-Type: application/json');

$docker = new Client();
$action = $_GET['action'] ?? '';

$response = match ($action) {
    'list'    => handleList($docker),
    'start'   => handleAction($docker, 'start'),
    'stop'    => handleAction($docker, 'stop'),
    'restart' => handleAction($docker, 'restart'),
    'remove'  => handleAction($docker, 'remove'),
    default   => ['error' => 'Unknown action'],
};

echo json_encode($response);

function handleList(Client $docker): array
{
    $grouped = $docker->listGrouped(true);

    $projects = [];
    foreach ($grouped['projects'] as $name => $containers) {
        $running = count(array_filter($containers, fn($c) => $c['state'] === 'running'));
        $projects[] = [
            'name'       => $name,
            'running'    => $running,
            'total'      => count($containers),
            'containers' => array_map(fn($c) => formatContainer($c, true), $containers),
        ];
    }

    $ungrouped = array_map(fn($c) => formatContainer($c, false), $grouped['ungrouped']);

    return ['projects' => $projects, 'ungrouped' => $ungrouped];
}

function handleAction(Client $docker, string $action): array
{
    $container = $_GET['container'] ?? '';
    if (empty($container)) {
        return ['error' => 'No container specified'];
    }

    $result = match ($action) {
        'start'   => $docker->start($container),
        'stop'    => $docker->stop($container),
        'restart' => $docker->restart($container),
        'remove'  => $docker->remove($container),
    };

    return ['ok' => true, 'result' => $result];
}

function formatContainer(array $c, bool $useShortName): array
{
    return [
        'id'      => $c['id'],
        'name'    => $c['name'],
        'display' => $useShortName ? Client::shortName($c) : $c['name'],
        'image'   => $c['image'],
        'status'  => $c['status'],
        'state'   => $c['state'],
        'ports'   => $c['ports'],
    ];
}
