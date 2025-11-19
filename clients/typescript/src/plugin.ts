// Auto-generated plugin interface

import { AuthsomeClient } from './client';

export interface ClientPlugin {
  readonly id: string;
  
  // Initialize plugin with base client
  init(client: AuthsomeClient): void;
  
  // Optional: validate configuration
  validate?(): Promise<boolean>;
}
