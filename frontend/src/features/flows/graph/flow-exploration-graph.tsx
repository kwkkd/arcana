import ExplorationGraphView from '@/features/assets/exploration-graph-view';
import { useFlow } from '@/providers/flow-provider';

function FlowExplorationGraph() {
    const { flowId } = useFlow();

    if (!flowId) {
        return null;
    }

    return <ExplorationGraphView flowId={flowId} />;
}

export default FlowExplorationGraph;
