declare module 'vscode' {
  export type Disposable = unknown;

  export type ExtensionContext = {
    subscriptions: { push(...items: Disposable[]): void };
    secrets: {
      get(key: string): Promise<string | undefined>;
      store(key: string, value: string): Promise<void>;
    };
  };

  export type InputBoxOptions = {
    title?: string;
    value?: string;
    password?: boolean;
    ignoreFocusOut?: boolean;
  };

  export type WorkspaceConfiguration = {
    get<T>(section: string): T | undefined;
  };

  export class Uri {
    static parse(value: string): Uri;
  }

  export const commands: {
    registerCommand(command: string, callback: (...args: unknown[]) => unknown): Disposable;
    executeCommand<T = unknown>(command: string, ...args: unknown[]): Promise<T>;
  };

  export const window: {
    showInputBox(options?: InputBoxOptions): Promise<string | undefined>;
    showInformationMessage(message: string): Promise<string | undefined>;
  };

  export const workspace: {
    getConfiguration(section?: string): WorkspaceConfiguration;
  };

  export const extensions: {
    getExtension(extensionId: string): unknown;
  };
}
