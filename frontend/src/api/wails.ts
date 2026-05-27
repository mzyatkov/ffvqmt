// Minimal typed wrapper around the Wails runtime bindings.
// At runtime, Wails injects window.go.main.App.<Method> and window.runtime.

export type Metric = "PSNR" | "SSIM" | "VMAF" | "XPSNR";

export interface CliOptions {
  autoSaveResults: boolean;
  autoSaveResultsFile: string;
  duration: number;
  exit: boolean;
  ffmpegDir: string;
  logCommands: boolean;
  logFrames: boolean;
  logFramesDir: string;
  logLevel: string;
  metrics: string[];
  plotUpdateSpeed: string;
  plotWindowManual: boolean;
  project: string;
  run: boolean;
  scalingMethod: string;
  skip: number;
  tempDir: string;
  vmafModel: string;
  vmafPhoneModel: boolean;
  vmafPool: string;
  vmafSubsample: number;
  vmafUpscaleToModel: boolean;
  refFile: string;
  distFiles: string[];
}

export interface ProbeInfo {
  ffmpegPath: string;
  ffprobePath: string;
  version: string;
  filters: string[];
  hasPSNR: boolean;
  hasSSIM: boolean;
  hasVMAF: boolean;
  hasXPSNR: boolean;
  vmafModels: string[];
  inBuildVMAF: boolean;
}

export interface MediaInfo {
  path: string;
  format: string;
  duration: number;
  bitrate: number;
  width: number;
  height: number;
  pixFmt: string;
  colorRange: string;
  frameRate: number;
  frameCount: number;
  videoCodec: string;
  audioCodec: string;
  sizeBytes: number;
}

export interface RunRequest {
  refPath: string;
  distPaths: string[];
  metrics: string[];
  skip: number;
  duration: number;
  startFrame: number;
  endFrame: number;
  scaling: string;
  vmafModel: string;
  vmafPool: string;
  vmafSubsample: number;
  vmafPhone: boolean;
  vmafUpscale: boolean;
  logCommands: boolean;
  logFrames: boolean;
  logFramesDir: string;
  tempDir: string;
  nThreads: number;
  autoSaveResults: boolean;
  autoSaveResultsFile: string;
}

// Wails global typings ------------------------------------------------------

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetInitialOptions(): Promise<CliOptions>;
          DetectFFmpeg(): Promise<ProbeInfo>;
          MediaInfo(path: string): Promise<MediaInfo>;
          MakeThumbnail(path: string): Promise<string>;
          SelectFiles(multi: boolean, title: string): Promise<string[]>;
          SelectDirectory(title: string): Promise<string>;
          SaveFileDialog(title: string, defaultName: string): Promise<string>;
          LoadProject(path: string): Promise<any>;
          SaveProject(path: string, p: any): Promise<void>;
          StartRun(req: RunRequest): Promise<void>;
          CancelRun(): Promise<void>;
        };
      };
    };
    runtime?: {
      EventsOn(name: string, cb: (data: any) => void): () => void;
      EventsOff(name: string): void;
      EventsEmit(name: string, ...data: any[]): void;
      OnFileDrop?(cb: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean): void;
    };
  }
}

const isWails = () => typeof window !== "undefined" && !!window.go?.main?.App;

function notAvailable<T>(name: string): Promise<T> {
  return Promise.reject(new Error(`Wails binding ${name} not available (running in browser?)`));
}

export const API = {
  available: isWails,

  getInitialOptions: (): Promise<CliOptions> =>
    isWails() ? window.go!.main!.App!.GetInitialOptions() : notAvailable("GetInitialOptions"),

  detectFFmpeg: (): Promise<ProbeInfo> =>
    isWails() ? window.go!.main!.App!.DetectFFmpeg() : notAvailable("DetectFFmpeg"),

  mediaInfo: (path: string): Promise<MediaInfo> =>
    isWails() ? window.go!.main!.App!.MediaInfo(path) : notAvailable("MediaInfo"),

  makeThumbnail: (path: string): Promise<string> =>
    isWails() ? window.go!.main!.App!.MakeThumbnail(path) : notAvailable("MakeThumbnail"),

  selectFiles: (multi: boolean, title: string): Promise<string[]> =>
    isWails() ? window.go!.main!.App!.SelectFiles(multi, title) : notAvailable("SelectFiles"),

  selectDir: (title: string): Promise<string> =>
    isWails() ? window.go!.main!.App!.SelectDirectory(title) : notAvailable("SelectDirectory"),

  saveFile: (title: string, defaultName: string): Promise<string> =>
    isWails() ? window.go!.main!.App!.SaveFileDialog(title, defaultName) : notAvailable("SaveFileDialog"),

  loadProject: (path: string): Promise<any> =>
    isWails() ? window.go!.main!.App!.LoadProject(path) : notAvailable("LoadProject"),

  saveProject: (path: string, p: any): Promise<void> =>
    isWails() ? window.go!.main!.App!.SaveProject(path, p) : notAvailable("SaveProject"),

  startRun: (req: RunRequest): Promise<void> =>
    isWails() ? window.go!.main!.App!.StartRun(req) : notAvailable("StartRun"),

  cancelRun: (): Promise<void> =>
    isWails() ? window.go!.main!.App!.CancelRun() : notAvailable("CancelRun"),

  on: (name: string, cb: (data: any) => void): (() => void) => {
    if (window.runtime?.EventsOn) {
      return window.runtime.EventsOn(name, cb);
    }
    return () => {};
  },

  onFileDrop: (cb: (paths: string[]) => void) => {
    if (window.runtime?.OnFileDrop) {
      window.runtime.OnFileDrop((_x, _y, paths) => cb(paths), true);
    }
  },
};
