import { type RouteResult } from './daemon/router.ts';
export type CheckStatus = 'pass' | 'warn' | 'fail';
export type DoctorCheck = {
    name: string;
    status: CheckStatus;
    message: string;
    action?: string;
};
export type ProbeResult = {
    name: string;
    status: CheckStatus;
    latencyMs?: number;
    message: string;
};
export type TestResult = {
    route: RouteResult;
    probes: ProbeResult[];
};
export type DoctorResult = {
    summary: {
        status: CheckStatus;
        passed: number;
        warned: number;
        failed: number;
    };
    checks: DoctorCheck[];
};
export declare function probeTarget(target: string): Promise<TestResult>;
export declare function runDoctor(routeTarget?: string): Promise<DoctorResult>;
