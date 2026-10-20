const COLUMN_GAP = 320;
const ROW_GAP = 110;

export interface LayoutEdge {
    source: string;
    target: string;
}

export interface LayoutNode {
    id: string;
}

export interface Position {
    x: number;
    y: number;
}

// Deterministic layered layout: a node sits one column right of its deepest parent, so an
// edge always reads left to right. Cycles cannot be layered — their members pile into one
// column past everything the walk settled rather than vanishing from the canvas.
export function layoutGraph(nodes: LayoutNode[], edges: LayoutEdge[]): Map<string, Position> {
    const ids = nodes.map((node) => node.id);
    const known = new Set(ids);
    const depth = new Map<string, number>(ids.map((id) => [id, 0]));
    const children = new Map<string, string[]>();
    const indegree = new Map<string, number>(ids.map((id) => [id, 0]));

    for (const edge of edges) {
        if (edge.source === edge.target || !known.has(edge.source) || !known.has(edge.target)) {
            continue;
        }

        children.set(edge.source, [...(children.get(edge.source) ?? []), edge.target]);
        indegree.set(edge.target, (indegree.get(edge.target) ?? 0) + 1);
    }

    const queue = ids.filter((id) => (indegree.get(id) ?? 0) === 0).sort(compareIds);
    const settled = new Set<string>();

    while (queue.length > 0) {
        const id = queue.shift()!;

        settled.add(id);

        for (const child of children.get(id) ?? []) {
            depth.set(child, Math.max(depth.get(child) ?? 0, (depth.get(id) ?? 0) + 1));
            indegree.set(child, (indegree.get(child) ?? 0) - 1);

            if ((indegree.get(child) ?? 0) === 0) {
                queue.push(child);
            }
        }
    }

    const maxSettledDepth = Math.max(0, ...[...settled].map((id) => depth.get(id) ?? 0));

    for (const id of ids) {
        if (!settled.has(id)) {
            depth.set(id, maxSettledDepth + 1);
        }
    }

    const columns = new Map<number, string[]>();

    for (const id of ids) {
        const column = depth.get(id) ?? 0;

        columns.set(column, [...(columns.get(column) ?? []), id]);
    }

    const positions = new Map<string, Position>();

    for (const [column, members] of columns) {
        const sorted = [...members].sort(compareIds);

        sorted.forEach((id, index) => {
            positions.set(id, { x: column * COLUMN_GAP, y: (index - (sorted.length - 1) / 2) * ROW_GAP });
        });
    }

    return positions;
}

function compareIds(a: string, b: string): number {
    const numeric = Number(a) - Number(b);

    if (Number.isFinite(numeric) && numeric !== 0) {
        return numeric;
    }

    return a.localeCompare(b);
}
