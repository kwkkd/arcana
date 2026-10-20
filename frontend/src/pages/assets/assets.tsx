import { Network } from 'lucide-react';

import { AppHeader, AppHeaderContent, AppHeaderTitle } from '@/components/layouts/app/app-header';
import AssetGraphView from '@/features/assets/asset-graph-view';

function Assets() {
    return (
        <>
            <AppHeader>
                <AppHeaderContent>
                    <AppHeaderTitle icon={<Network className="size-4 shrink-0" />}>Assets</AppHeaderTitle>
                </AppHeaderContent>
            </AppHeader>
            <div className="flex h-[calc(100dvh-3rem)] w-full max-w-full flex-col p-4">
                <AssetGraphView />
            </div>
        </>
    );
}

export default Assets;
