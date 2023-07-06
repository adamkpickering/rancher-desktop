/**
 * This module describes the various paths we use to store state & data.
 */
import { spawnSync } from 'child_process';
import os from 'os';
import path from 'path';

import electron from 'electron';

export interface Paths {
  /** appHome: the location of the main appdata directory. */
  appHome: string;
  /** altAppHome is a secondary directory for application data. */
  altAppHome: string;
  /** Directory which holds configuration. */
  config: string;
  /** Directory which holds logs. */
  logs: string;
  /** Directory which holds caches that may be removed. */
  cache: string;
  /** Directory that holds resource files in the RD installation. */
  resources: string;
  /** Directory holding Lima state (macOS-specific). */
  lima: string;
  /** Directory holding provided binary resources */
  integration: string;
  /** The directory that used to hold provided binary integrations */
  oldIntegration: string;
  /** Deployment Profile System-wide startup settings path. */
  deploymentProfileSystem: string;
  /** Deployment Profile User startup settings path. */
  deploymentProfileUser: string;
  /** Directory that will hold extension data. */
  readonly extensionRoot: string;
  /** Directory holding the WSL distribution (Windows-specific). */
  wslDistro: string;
  /** Directory holding the WSL data distribution (Windows-specific). */
  wslDistroData: string;
}

export class UnixPaths implements Paths {
  appHome = '';
  altAppHome = '';
  config = '';
  logs = '';
  cache = '';
  resources = '';
  lima = '';
  oldIntegration = '';
  integration = '';
  deploymentProfileSystem = '';
  deploymentProfileUser = '';
  extensionRoot = '';

  constructor(pathsData: object) {
    Object.assign(this, pathsData);
  }

  get wslDistro(): string {
    throw new Error('wslDistro not available for Unix');
  }

  get wslDistroData(): string {
    throw new Error('wslDistroData not available for Unix');
  }
}

export class WindowsPaths implements Paths {
  appHome = '';
  altAppHome = '';
  config = '';
  logs = '';
  cache = '';
  resources = '';
  extensionRoot = '';
  wslDistro = '';
  wslDistroData = '';

  constructor(pathsData: object) {
    Object.assign(this, pathsData);
  }

  get lima(): string {
    throw new Error('lima not available for Windows');
  }

  get oldIntegration(): string {
    throw new Error('Internal error: oldIntegration path not available for Windows');
  }

  get integration(): string {
    throw new Error('Internal error: integration path not available for Windows');
  }

  get deploymentProfileSystem(): string {
    throw new Error('Internal error: Windows profiles will be read from Registry');
  }

  get deploymentProfileUser(): string {
    throw new Error('Internal error: Windows profiles will be read from Registry');
  }
}

// Gets the path to rdctl. Returns null if rdctl cannot be found.
function getRdctlPath(): string {
  let basePath: string;

  // If we are running as a script (i.e. npm run postinstall), electron.app is undefined
  if (electron.app === undefined) {
    basePath = process.cwd();
  } else {
    basePath = electron.app.isPackaged ? process.resourcesPath : electron.app.getAppPath();
  }
  const osSpecificName = os.platform().startsWith('win') ? `rdctl.exe` : 'rdctl';

  return path.join(basePath, 'resources', os.platform(), 'bin', osSpecificName);
}

function getPaths(): Paths {
  let pathsData: Partial<Paths>;

  try {
    const result = spawnSync(getRdctlPath(), ['paths'], { encoding: 'utf8' });

    if (result.status !== 0) {
      throw new Error(`rdctl paths failed: ${ JSON.stringify(result) }`);
    }
    pathsData = JSON.parse(result.stdout);
  } catch {
    // In some cases rdctl may not yet be present, or may be an
    // older version that does not know about paths. In this case,
    // use default paths.
    pathsData = {
      resources: path.join(process.cwd(), 'resources'),
      logs:      path.join(process.cwd(), 'script_logs'),
    };
  }

  switch (process.platform) {
  case 'darwin':
    return new UnixPaths(pathsData);
  case 'linux':
    return new UnixPaths(pathsData);
  case 'win32':
    return new WindowsPaths(pathsData);
  default:
    throw new Error(`Platform "${ process.platform }" is not supported.`);
  }
}

export default getPaths();
