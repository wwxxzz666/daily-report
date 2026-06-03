export namespace config {
	
	export class BrowserConfig {
	    Enabled: boolean;
	    Browsers: string[];
	    HistoryHours: number;
	
	    static createFrom(source: any = {}) {
	        return new BrowserConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = source["Enabled"];
	        this.Browsers = source["Browsers"];
	        this.HistoryHours = source["HistoryHours"];
	    }
	}
	export class ReportConfig {
	    OutputDir: string;
	    WeeklyDay: number;
	    WeeklyTime: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.OutputDir = source["OutputDir"];
	        this.WeeklyDay = source["WeeklyDay"];
	        this.WeeklyTime = source["WeeklyTime"];
	    }
	}
	export class LLMConfig {
	    Provider: string;
	    Endpoint: string;
	    Model: string;
	    APIKey: string;
	    Temperature: number;
	    MaxTokens: number;
	    Timeout: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Provider = source["Provider"];
	        this.Endpoint = source["Endpoint"];
	        this.Model = source["Model"];
	        this.APIKey = source["APIKey"];
	        this.Temperature = source["Temperature"];
	        this.MaxTokens = source["MaxTokens"];
	        this.Timeout = source["Timeout"];
	    }
	}
	export class WorkTimeConfig {
	    Start: string;
	    End: string;
	    Weekdays: number[];
	
	    static createFrom(source: any = {}) {
	        return new WorkTimeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Start = source["Start"];
	        this.End = source["End"];
	        this.Weekdays = source["Weekdays"];
	    }
	}
	export class Config {
	    WorkTime: WorkTimeConfig;
	    SampleInterval: string;
	    MonitoredApps: string[];
	    IgnoredApps: string[];
	    SensitiveWords: string[];
	    Browser: BrowserConfig;
	    LLM: LLMConfig;
	    Report: ReportConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.WorkTime = this.convertValues(source["WorkTime"], WorkTimeConfig);
	        this.SampleInterval = source["SampleInterval"];
	        this.MonitoredApps = source["MonitoredApps"];
	        this.IgnoredApps = source["IgnoredApps"];
	        this.SensitiveWords = source["SensitiveWords"];
	        this.Browser = this.convertValues(source["Browser"], BrowserConfig);
	        this.LLM = this.convertValues(source["LLM"], LLMConfig);
	        this.Report = this.convertValues(source["Report"], ReportConfig);
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

export namespace main {
	
	export class AppUsage {
	    processName: string;
	    totalSec: number;
	    duration: string;
	
	    static createFrom(source: any = {}) {
	        return new AppUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.processName = source["processName"];
	        this.totalSec = source["totalSec"];
	        this.duration = source["duration"];
	    }
	}
	export class LLMConfigRequest {
	    provider: string;
	    api_key: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	    }
	}
	export class ProviderInfo {
	    name: string;
	    label: string;
	    defaultModel: string;
	    needsKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.defaultModel = source["defaultModel"];
	        this.needsKey = source["needsKey"];
	    }
	}
	export class ReportConfigRequest {
	    weekly_day: number;
	
	    static createFrom(source: any = {}) {
	        return new ReportConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weekly_day = source["weekly_day"];
	    }
	}
	export class ReportInfo {
	    fileName: string;
	    reportType: string;
	    generatedAt: string;
	    filePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.reportType = source["reportType"];
	        this.generatedAt = source["generatedAt"];
	        this.filePath = source["filePath"];
	    }
	}
	export class WorkTimeConfigRequest {
	    start: string;
	    end: string;
	    weekdays: number[];
	
	    static createFrom(source: any = {}) {
	        return new WorkTimeConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.weekdays = source["weekdays"];
	    }
	}
	export class SaveConfigRequest {
	    llm?: LLMConfigRequest;
	    work_time?: WorkTimeConfigRequest;
	    report?: ReportConfigRequest;
	
	    static createFrom(source: any = {}) {
	        return new SaveConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llm = this.convertValues(source["llm"], LLMConfigRequest);
	        this.work_time = this.convertValues(source["work_time"], WorkTimeConfigRequest);
	        this.report = this.convertValues(source["report"], ReportConfigRequest);
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
	export class Status {
	    isRunning: boolean;
	    countdown: string;
	    recordedTime: string;
	    appCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.countdown = source["countdown"];
	        this.recordedTime = source["recordedTime"];
	        this.appCount = source["appCount"];
	    }
	}
	export class TodaySummary {
	    apps: AppUsage[];
	
	    static createFrom(source: any = {}) {
	        return new TodaySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apps = this.convertValues(source["apps"], AppUsage);
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

