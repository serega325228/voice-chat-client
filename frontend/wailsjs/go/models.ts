export namespace main {
	
	export class BootstrapResult {
	    isAuthenticated: boolean;
	    authError?: string;
	    authInfo?: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isAuthenticated = source["isAuthenticated"];
	        this.authError = source["authError"];
	        this.authInfo = source["authInfo"];
	    }
	}
	export class CallStateResult {
	    sessionId?: string;
	    isMuted: boolean;
	    status: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new CallStateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.isMuted = source["isMuted"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}

}

