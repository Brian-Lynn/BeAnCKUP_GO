export namespace types {
	
	export class TreeNode {
	    name: string;
	    path: string;
	    isDir: boolean;
	    status?: string;
	    children?: TreeNode[];
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.status = source["status"];
	        this.children = this.convertValues(source["children"], TreeNode);
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
	export class Episode {
	    id: string;
	    name: string;
	    seriesId: string;
	    // Go type: time
	    createdAt: any;
	    status: string;
	    packagePath: string;
	    fileCount: number;
	    totalSize: number;
	    estimatedSize: number;
	
	    static createFrom(source: any = {}) {
	        return new Episode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.seriesId = source["seriesId"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.status = source["status"];
	        this.packagePath = source["packagePath"];
	        this.fileCount = source["fileCount"];
	        this.totalSize = source["totalSize"];
	        this.estimatedSize = source["estimatedSize"];
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
	export class BackupPreparationResult {
	    episodes: Episode[];
	    fileTree: TreeNode[];
	    // Go type: struct { NewCount int "json:\"newCount\""; ModifiedCount int "json:\"modifiedCount\""; DeletedCount int "json:\"deletedCount\""; TotalSize int64 "json:\"totalSize\"" }
	    changeInfo: any;
	
	    static createFrom(source: any = {}) {
	        return new BackupPreparationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.episodes = this.convertValues(source["episodes"], Episode);
	        this.fileTree = this.convertValues(source["fileTree"], TreeNode);
	        this.changeInfo = this.convertValues(source["changeInfo"], Object);
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
	
	export class TaskStatus {
	    isRunning: boolean;
	    currentPhase: string;
	    progress: number;
	    processedFiles: number;
	    totalFiles: number;
	    processedSize: number;
	    totalSize: number;
	    speed: number;
	    elapsedTime: number;
	    estimatedTime: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.currentPhase = source["currentPhase"];
	        this.progress = source["progress"];
	        this.processedFiles = source["processedFiles"];
	        this.totalFiles = source["totalFiles"];
	        this.processedSize = source["processedSize"];
	        this.totalSize = source["totalSize"];
	        this.speed = source["speed"];
	        this.elapsedTime = source["elapsedTime"];
	        this.estimatedTime = source["estimatedTime"];
	    }
	}

}

