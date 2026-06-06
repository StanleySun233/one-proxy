import type { CliContext } from './main.ts';
import {
  activeProfileName,
  addProfile,
  readProfilesIndex,
  useProfile
} from './storage.ts';

function optionValue(args: string[], name: string): string | undefined {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
}

function print(value: unknown, context: CliContext): void {
  if (context.json) {
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
    return;
  }
  process.stdout.write(`${String(value)}\n`);
}

export async function profileCommand(args: string[], context: CliContext): Promise<void> {
  const action = args[0];
  if (action === 'add') {
    const name = args[1];
    const controlPlaneUrl = optionValue(args, '--control-plane') || args[2];
    if (!name || !controlPlaneUrl) {
      throw Object.assign(new Error('profile add requires <name> and --control-plane <url>.'), { code: 'SYNTAX_ERROR', exitCode: 2 });
    }
    const profile = await addProfile(name, controlPlaneUrl);
    print(context.json ? { profile } : `Active profile: ${profile.name}`, context);
    return;
  }
  if (action === 'use') {
    const name = args[1];
    if (!name) {
      throw Object.assign(new Error('profile use requires <name>.'), { code: 'SYNTAX_ERROR', exitCode: 2 });
    }
    const profile = await useProfile(name);
    print(context.json ? { profile } : `Active profile: ${profile.name}`, context);
    return;
  }
  if (action === 'list') {
    const index = await readProfilesIndex();
    const profiles = Object.values(index.profiles).sort((left, right) => left.name.localeCompare(right.name));
    if (context.json) {
      print({ activeProfile: index.activeProfile ?? activeProfileName(), profiles }, context);
      return;
    }
    print(profiles.map((profile) => `${profile.name === (index.activeProfile ?? activeProfileName()) ? '*' : ' '}\t${profile.name}\t${profile.controlPlaneUrl}`).join('\n'), context);
    return;
  }
  if (action === 'current') {
    const index = await readProfilesIndex();
    const name = index.activeProfile ?? activeProfileName();
    const profile = index.profiles[name] ?? null;
    print(context.json ? { activeProfile: name, profile } : `${name}${profile ? `\t${profile.controlPlaneUrl}` : ''}`, context);
    return;
  }
  throw Object.assign(new Error('profile requires add, use, list, or current'), { code: 'SYNTAX_ERROR', exitCode: 2 });
}
