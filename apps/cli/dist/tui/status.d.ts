import type { RouteResult } from '../daemon/router.ts';
import type { TuiStatusSnapshot } from './footer.ts';
export type TuiStatusInput = {
    route?: RouteResult | null;
    pingMs?: number | null;
    uploadBytes?: number | null;
    downloadBytes?: number | null;
};
export declare function collectTuiStatusSnapshot(input?: TuiStatusInput): Promise<TuiStatusSnapshot>;
