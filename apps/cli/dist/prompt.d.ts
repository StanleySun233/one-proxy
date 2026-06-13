import { stdin as input, stdout as output } from 'node:process';
type PromptInput = typeof input;
type PromptOutput = typeof output;
export declare function promptText(label: string, streams?: {
    input?: PromptInput;
    output?: PromptOutput;
}): Promise<string>;
export declare function promptPassword(label: string, streams?: {
    input?: PromptInput;
    output?: PromptOutput;
}): Promise<string>;
export {};
