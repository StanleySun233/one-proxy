import * as readline from 'node:readline/promises';
import { stdin as input, stdout as output } from 'node:process';
export async function promptText(label, streams = {}) {
    const rl = readline.createInterface({ input: streams.input ?? input, output: streams.output ?? output });
    const answer = (await rl.question(label)).trim();
    rl.close();
    return answer;
}
export async function promptPassword(label, streams = {}) {
    const targetInput = streams.input ?? input;
    const targetOutput = streams.output ?? output;
    const mutableOutput = targetOutput;
    const rl = readline.createInterface({
        input: targetInput,
        output: mutableOutput,
        terminal: true
    });
    const originalWrite = mutableOutput.write.bind(mutableOutput);
    mutableOutput.write = ((chunk, encoding, callback) => {
        if (mutableOutput.muted) {
            return true;
        }
        return typeof encoding === 'function'
            ? originalWrite(chunk, encoding)
            : originalWrite(chunk, encoding, callback);
    });
    try {
        targetOutput.write(label);
        mutableOutput.muted = true;
        const answer = await rl.question('');
        mutableOutput.muted = false;
        targetOutput.write('\n');
        return answer.trim();
    }
    finally {
        mutableOutput.muted = false;
        mutableOutput.write = originalWrite;
        rl.close();
    }
}
//# sourceMappingURL=prompt.js.map