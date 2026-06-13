import * as readline from 'node:readline/promises';
import { stdin as input, stdout as output } from 'node:process';

type PromptInput = typeof input;
type PromptOutput = typeof output;

type MutableOutput = PromptOutput & {
  muted?: boolean;
};

export async function promptText(label: string, streams: { input?: PromptInput; output?: PromptOutput } = {}): Promise<string> {
  const rl = readline.createInterface({ input: streams.input ?? input, output: streams.output ?? output });
  const answer = (await rl.question(label)).trim();
  rl.close();
  return answer;
}

export async function promptPassword(label: string, streams: { input?: PromptInput; output?: PromptOutput } = {}): Promise<string> {
  const targetInput = streams.input ?? input;
  const targetOutput = streams.output ?? output;
  const mutableOutput = targetOutput as MutableOutput;
  const rl = readline.createInterface({
    input: targetInput,
    output: mutableOutput,
    terminal: true
  });
  const originalWrite = mutableOutput.write.bind(mutableOutput);
  mutableOutput.write = ((chunk: string | Uint8Array, encoding?: BufferEncoding | ((error?: Error | null) => void), callback?: (error?: Error | null) => void) => {
    if (mutableOutput.muted) {
      return true;
    }
    return typeof encoding === 'function'
      ? originalWrite(chunk, encoding)
      : originalWrite(chunk, encoding, callback);
  }) as typeof mutableOutput.write;
  try {
    targetOutput.write(label);
    mutableOutput.muted = true;
    const answer = await rl.question('');
    mutableOutput.muted = false;
    targetOutput.write('\n');
    return answer.trim();
  } finally {
    mutableOutput.muted = false;
    mutableOutput.write = originalWrite;
    rl.close();
  }
}
