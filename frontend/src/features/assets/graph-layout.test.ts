import { describe, expect, it } from 'vitest';

import { layoutGraph } from './graph-layout';

const node = (id: string) => ({ id });
const edge = (source: string, target: string) => ({ source, target });

describe('layoutGraph', () => {
    it('places each node in the column after its parent', () => {
        const positions = layoutGraph([node('1'), node('2'), node('3')], [edge('1', '2'), edge('2', '3')]);

        expect(positions.get('1')!.x).toBeLessThan(positions.get('2')!.x);
        expect(positions.get('2')!.x).toBeLessThan(positions.get('3')!.x);
        expect(positions.get('1')!.y).toBe(0);
        expect(positions.get('2')!.y).toBe(0);
        expect(positions.get('3')!.y).toBe(0);
    });

    it('sinks a node below its deepest parent, not its first', () => {
        const positions = layoutGraph(
            [node('1'), node('2'), node('3'), node('4')],
            [edge('1', '2'), edge('1', '3'), edge('2', '4'), edge('3', '4')],
        );

        expect(positions.get('4')!.x).toBe(2 * positions.get('2')!.x);
    });

    it('stacks siblings of one column deterministically', () => {
        const positions = layoutGraph([node('10'), node('2'), node('5')], []);

        const byY = [...positions.entries()].sort((a, b) => a[1].y - b[1].y).map(([id]) => id);

        expect(byY).toEqual(['2', '5', '10']);
        expect(positions.get('5')!.y).toBe(0);
    });

    it('parks a cycle one column past the settled walk instead of losing it', () => {
        const positions = layoutGraph(
            [node('1'), node('2'), node('3')],
            [edge('1', '2'), edge('2', '3'), edge('3', '2')],
        );

        expect(positions.size).toBe(3);
        expect(positions.get('2')!.x).toBe(positions.get('3')!.x);
        expect(positions.get('2')!.x).toBeGreaterThan(positions.get('1')!.x);
        expect(positions.get('2')!.y).not.toBe(positions.get('3')!.y);
    });

    it('ignores edges to unknown nodes and self-loops', () => {
        const positions = layoutGraph([node('1')], [edge('1', '1'), edge('1', '99'), edge('99', '1')]);

        expect(positions.get('1')).toEqual({ x: 0, y: 0 });
    });
});
