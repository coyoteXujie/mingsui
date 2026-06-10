export namespace client {
	
	export class RuntimeMetrics {
	    active_connections: number;
	    total_connections: number;
	    upload_bytes: number;
	    download_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active_connections = source["active_connections"];
	        this.total_connections = source["total_connections"];
	        this.upload_bytes = source["upload_bytes"];
	        this.download_bytes = source["download_bytes"];
	    }
	}
	export class RuntimeStatus {
	    running: boolean;
	    local_addr: string;
	    http_addr?: string;
	    relay_addr: string;
	    // Go type: time
	    started_at?: any;
	    last_error?: string;
	    metrics: RuntimeMetrics;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.local_addr = source["local_addr"];
	        this.http_addr = source["http_addr"];
	        this.relay_addr = source["relay_addr"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.last_error = source["last_error"];
	        this.metrics = this.convertValues(source["metrics"], RuntimeMetrics);
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

export namespace config {
	
	export class ClientAuthConfig {
	    enabled: boolean;
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new ClientAuthConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class RelaySubscription {
	    name: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new RelaySubscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	    }
	}
	export class ProxyProfile {
	    name: string;
	    protocol: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.url = source["url"];
	    }
	}
	export class ClientTLSConfig {
	    enabled: boolean;
	    server_name: string;
	    ca_file: string;
	    insecure_skip_verify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ClientTLSConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.server_name = source["server_name"];
	        this.ca_file = source["ca_file"];
	        this.insecure_skip_verify = source["insecure_skip_verify"];
	    }
	}
	export class RelayProfile {
	    name: string;
	    relay_addr: string;
	    token: string;
	    tls: ClientTLSConfig;
	
	    static createFrom(source: any = {}) {
	        return new RelayProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relay_addr = source["relay_addr"];
	        this.token = source["token"];
	        this.tls = this.convertValues(source["tls"], ClientTLSConfig);
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
	export class ClientConfig {
	    local_addr: string;
	    http_addr: string;
	    relay_addr: string;
	    token: string;
	    dial_timeout_seconds: number;
	    active_profile?: string;
	    active_proxy_profile?: string;
	    profiles?: RelayProfile[];
	    proxy_profiles?: ProxyProfile[];
	    subscriptions?: RelaySubscription[];
	    local_auth: ClientAuthConfig;
	    tls: ClientTLSConfig;
	
	    static createFrom(source: any = {}) {
	        return new ClientConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.local_addr = source["local_addr"];
	        this.http_addr = source["http_addr"];
	        this.relay_addr = source["relay_addr"];
	        this.token = source["token"];
	        this.dial_timeout_seconds = source["dial_timeout_seconds"];
	        this.active_profile = source["active_profile"];
	        this.active_proxy_profile = source["active_proxy_profile"];
	        this.profiles = this.convertValues(source["profiles"], RelayProfile);
	        this.proxy_profiles = this.convertValues(source["proxy_profiles"], ProxyProfile);
	        this.subscriptions = this.convertValues(source["subscriptions"], RelaySubscription);
	        this.local_auth = this.convertValues(source["local_auth"], ClientAuthConfig);
	        this.tls = this.convertValues(source["tls"], ClientTLSConfig);
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
	
	export class RelayProfileRequest {
	    name: string;
	    relay_addr: string;
	    token: string;
	    tls: config.ClientTLSConfig;
	    replace: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RelayProfileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relay_addr = source["relay_addr"];
	        this.token = source["token"];
	        this.tls = this.convertValues(source["tls"], config.ClientTLSConfig);
	        this.replace = source["replace"];
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
	export class SubscriptionRequest {
	    name: string;
	    url: string;
	    replace: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.replace = source["replace"];
	    }
	}

}

export namespace systemproxy {
	
	export class Status {
	    supported: boolean;
	    enabled: boolean;
	    mode?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.message = source["message"];
	    }
	}

}

