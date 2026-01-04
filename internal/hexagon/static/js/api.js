// API client for Hexagon

const API = {
    baseUrl: '',

    async request(method, endpoint, data = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(this.baseUrl + endpoint, options);
            const json = await response.json();
            
            if (!response.ok) {
                throw new Error(json.error || 'Request failed');
            }
            
            return json;
        } catch (error) {
            console.error(`API Error (${endpoint}):`, error);
            throw error;
        }
    },

    // Status
    async getStatus() {
        return this.request('GET', '/api/status');
    },

    // Attacks
    async launchLocust(target, rounds, concurrency) {
        return this.request('POST', '/api/sting/locust', {
            target,
            rounds: parseInt(rounds),
            concurrency: parseInt(concurrency),
        });
    },

    async launchTick(target, sockets, delay) {
        return this.request('POST', '/api/sting/tick', {
            target,
            sockets: parseInt(sockets),
            delay,
        });
    },

    async launchFlySwarm(target, rounds, port, concurrency, packetSize) {
        return this.request('POST', '/api/sting/flyswarm', {
            target,
            rounds: parseInt(rounds),
            port: parseInt(port),
            concurrency: parseInt(concurrency),
            packetSize: parseInt(packetSize),
        });
    },

    async stopAttack(attackId = null) {
        const data = attackId ? { attackId } : {};
        return this.request('POST', '/api/attack/stop', data);
    },
};
