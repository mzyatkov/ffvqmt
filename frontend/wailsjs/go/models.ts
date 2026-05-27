export namespace cli {
	
	export class Options {
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
	
	    static createFrom(source: any = {}) {
	        return new Options(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoSaveResults = source["autoSaveResults"];
	        this.autoSaveResultsFile = source["autoSaveResultsFile"];
	        this.duration = source["duration"];
	        this.exit = source["exit"];
	        this.ffmpegDir = source["ffmpegDir"];
	        this.logCommands = source["logCommands"];
	        this.logFrames = source["logFrames"];
	        this.logFramesDir = source["logFramesDir"];
	        this.logLevel = source["logLevel"];
	        this.metrics = source["metrics"];
	        this.plotUpdateSpeed = source["plotUpdateSpeed"];
	        this.plotWindowManual = source["plotWindowManual"];
	        this.project = source["project"];
	        this.run = source["run"];
	        this.scalingMethod = source["scalingMethod"];
	        this.skip = source["skip"];
	        this.tempDir = source["tempDir"];
	        this.vmafModel = source["vmafModel"];
	        this.vmafPhoneModel = source["vmafPhoneModel"];
	        this.vmafPool = source["vmafPool"];
	        this.vmafSubsample = source["vmafSubsample"];
	        this.vmafUpscaleToModel = source["vmafUpscaleToModel"];
	        this.refFile = source["refFile"];
	        this.distFiles = source["distFiles"];
	    }
	}

}

export namespace ffmpeg {
	
	export class MediaInfo {
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
	    isRaw: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.format = source["format"];
	        this.duration = source["duration"];
	        this.bitrate = source["bitrate"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.pixFmt = source["pixFmt"];
	        this.colorRange = source["colorRange"];
	        this.frameRate = source["frameRate"];
	        this.frameCount = source["frameCount"];
	        this.videoCodec = source["videoCodec"];
	        this.audioCodec = source["audioCodec"];
	        this.sizeBytes = source["sizeBytes"];
	        this.isRaw = source["isRaw"];
	    }
	}
	export class ProbeInfo {
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
	
	    static createFrom(source: any = {}) {
	        return new ProbeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ffmpegPath = source["ffmpegPath"];
	        this.ffprobePath = source["ffprobePath"];
	        this.version = source["version"];
	        this.filters = source["filters"];
	        this.hasPSNR = source["hasPSNR"];
	        this.hasSSIM = source["hasSSIM"];
	        this.hasVMAF = source["hasVMAF"];
	        this.hasXPSNR = source["hasXPSNR"];
	        this.vmafModels = source["vmafModels"];
	        this.inBuildVMAF = source["inBuildVMAF"];
	    }
	}
	export class RawFormat {
	    format: string;
	    pixFmt: string;
	    width: number;
	    height: number;
	    frameRate: number;
	
	    static createFrom(source: any = {}) {
	        return new RawFormat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.pixFmt = source["pixFmt"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.frameRate = source["frameRate"];
	    }
	}

}

export namespace metrics {
	
	export class RunRequest {
	    refPath: string;
	    distPaths: string[];
	    refRaw?: ffmpeg.RawFormat;
	    distRaw?: Record<string, ffmpeg.RawFormat>;
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
	
	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.refPath = source["refPath"];
	        this.distPaths = source["distPaths"];
	        this.refRaw = this.convertValues(source["refRaw"], ffmpeg.RawFormat);
	        this.distRaw = this.convertValues(source["distRaw"], ffmpeg.RawFormat, true);
	        this.metrics = source["metrics"];
	        this.skip = source["skip"];
	        this.duration = source["duration"];
	        this.startFrame = source["startFrame"];
	        this.endFrame = source["endFrame"];
	        this.scaling = source["scaling"];
	        this.vmafModel = source["vmafModel"];
	        this.vmafPool = source["vmafPool"];
	        this.vmafSubsample = source["vmafSubsample"];
	        this.vmafPhone = source["vmafPhone"];
	        this.vmafUpscale = source["vmafUpscale"];
	        this.logCommands = source["logCommands"];
	        this.logFrames = source["logFrames"];
	        this.logFramesDir = source["logFramesDir"];
	        this.tempDir = source["tempDir"];
	        this.nThreads = source["nThreads"];
	        this.autoSaveResults = source["autoSaveResults"];
	        this.autoSaveResultsFile = source["autoSaveResultsFile"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace project {
	
	export class DistEntry {
	    path: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DistEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.active = source["active"];
	    }
	}
	export class Project {
	    version: number;
	    refFile: string;
	    distFiles: DistEntry[];
	    metrics: string[];
	    skip: number;
	    duration: number;
	    scaling: string;
	    vmafModel: string;
	    vmafPool: string;
	    vmafSubsample: number;
	    vmafPhone: boolean;
	    vmafUpscale: boolean;
	    extra?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.refFile = source["refFile"];
	        this.distFiles = this.convertValues(source["distFiles"], DistEntry);
	        this.metrics = source["metrics"];
	        this.skip = source["skip"];
	        this.duration = source["duration"];
	        this.scaling = source["scaling"];
	        this.vmafModel = source["vmafModel"];
	        this.vmafPool = source["vmafPool"];
	        this.vmafSubsample = source["vmafSubsample"];
	        this.vmafPhone = source["vmafPhone"];
	        this.vmafUpscale = source["vmafUpscale"];
	        this.extra = source["extra"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

